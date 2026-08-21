package server

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"google.golang.org/grpc"
)

// grpcAuditUnaryInterceptor writes a management-audit row for every MUTATING gRPC
// call.
//
// Why here and not in each handler: the REST surface logs from 20-odd handler call
// sites, but none of the 12 gRPC handlers did — so the control plane, the actor most
// worth auditing, was completely invisible in management_audit_log. Doing it in the
// interceptor covers every existing RPC and every future one by construction, and it
// cannot be forgotten when a handler is added.
//
// The trade-off is deliberate: the interceptor cannot see the resource UUID (that
// lives in the request body), so it records WHO called WHAT and whether it
// succeeded. Handlers that need resource-level detail can still log their own row;
// this is the floor, not a replacement.
func grpcAuditUnaryInterceptor(application *Application) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		resp, err := handler(ctx, req)

		if application == nil || application.AuditLogger == nil {
			return resp, err
		}
		action, resourceType, ok := grpcAuditTarget(info.FullMethod)
		if !ok {
			// Reads are not audited: they would swamp the log and the REST surface
			// does not audit them either.
			return resp, err
		}

		claims := middleware.JWTClaimsFromContext(ctx)
		var tenantID int64
		var principal string
		switch {
		case claims != nil:
			tenantID = claims.TenantID
			principal = grpcAuditPrincipal(claims)
		default:
			// No JWT. The one legitimate way to reach a handler without claims is
			// the setup window: bootstrap RPCs authenticate by x-setup-token + mTLS
			// (see authorizeSetupBootstrap), yet they perform the most
			// security-relevant mutations of an install's life — system tenant,
			// first admin, control service — so skipping them left the whole setup
			// window invisible in management_audit_log. The principal is the mTLS
			// peer identity the setup interceptor already relies on; tenant stays 0
			// (recorded as NULL) because these mutations happen before any tenant
			// exists. Every other claims-less method keeps the existing bail:
			// unauthenticated calls were rejected before reaching a handler.
			if _, isBootstrap := grpcBootstrapMethods[info.FullMethod]; !isBootstrap {
				return resp, err
			}
			principal = grpcBootstrapCaller(ctx)
			if principal == "" {
				// Never log a blank "who". The call was still authorized by the
				// pre-shared credential, so name the credential class.
				principal = "setup-token"
			}
		}

		outcome := "success"
		if err != nil {
			outcome = "failure"
		}

		_ = application.AuditLogger.Log(ctx, auditlog.LogEntry{
			TenantID:     tenantID,
			Action:       action,
			ResourceType: resourceType,
			// The calling principal, as carried by the token (or the mTLS peer on
			// setup RPCs). The internal actor_client_id is not resolvable here
			// without a lookup per call, so the principal is recorded in the change
			// payload instead.
			ResourceID: "",
			Changes:    grpcAuditChanges(info.FullMethod, principal),
			Outcome:    outcome,
		})

		return resp, err
	}
}

// grpcAuditTarget maps a gRPC method to an (action, resourceType) pair, and reports
// whether the method mutates anything at all.
func grpcAuditTarget(fullMethod string) (action string, resourceType string, mutating bool) {
	idx := strings.LastIndex(fullMethod, "/")
	if idx < 0 || idx == len(fullMethod)-1 {
		return "", "", false
	}
	name := fullMethod[idx+1:]

	switch {
	case strings.HasPrefix(name, "Create"):
		action, resourceType = "create", strings.TrimPrefix(name, "Create")
	case strings.HasPrefix(name, "Update"):
		action, resourceType = "update", strings.TrimPrefix(name, "Update")
	case strings.HasPrefix(name, "Delete"):
		action, resourceType = "delete", strings.TrimPrefix(name, "Delete")
	case strings.HasPrefix(name, "Set"):
		action, resourceType = "update", strings.TrimPrefix(name, "Set")
	case strings.HasPrefix(name, "Assign"):
		action, resourceType = "assign", strings.TrimPrefix(name, "Assign")
	case strings.HasPrefix(name, "Remove"):
		action, resourceType = "remove", strings.TrimPrefix(name, "Remove")
	case strings.HasPrefix(name, "Rotate"):
		action, resourceType = "rotate", strings.TrimPrefix(name, "Rotate")
	case strings.HasPrefix(name, "Register"):
		action, resourceType = "create", strings.TrimPrefix(name, "Register")
	// Ensure* is get-or-create (the SetupService convergence RPCs: EnsureRole,
	// EnsureControlClient, ...) and Complete* finalizes state (CompleteSetup
	// activates the system tenant; CompleteUserAccount flips account state).
	// Both mutate, and without these prefixes the setup window's client/role/API
	// provisioning never produced a row.
	case strings.HasPrefix(name, "Ensure"):
		action, resourceType = "create", strings.TrimPrefix(name, "Ensure")
	case strings.HasPrefix(name, "Complete"):
		action, resourceType = "update", strings.TrimPrefix(name, "Complete")
	default:
		// Get/List/Introspect/Authorize and anything else non-mutating.
		return "", "", false
	}

	if resourceType == "" {
		resourceType = "unknown"
	}
	return action, strings.ToLower(resourceType), true
}

// grpcAuditPrincipal names the calling principal as carried by the token: the
// service (`svc`) claim when present, the bare subject otherwise — never blank.
func grpcAuditPrincipal(claims *middleware.JWTClaims) string {
	if claims.Service != "" {
		return claims.Service
	}
	return claims.Sub
}

// grpcAuditChanges records the calling principal and the exact method, so a reviewer
// can tell which control-plane service (or, on setup RPCs, which mTLS peer) made
// the call. Marshalled rather than concatenated: the setup principal embeds a
// certificate CommonName, which the deployment's operator controls, and a quote
// in it would otherwise corrupt the JSON and make the logger DROP the row.
func grpcAuditChanges(fullMethod string, principal string) string {
	payload, marshalErr := json.Marshal(map[string]string{
		"transport": "grpc",
		"method":    fullMethod,
		"principal": principal,
	})
	if marshalErr != nil {
		// Unreachable for a map of strings; keep the row rather than dropping it.
		return `{"transport":"grpc"}`
	}
	return string(payload)
}
