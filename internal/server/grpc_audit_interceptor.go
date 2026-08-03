package server

import (
	"context"
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
		if claims == nil {
			// Unauthenticated calls were rejected before reaching a handler.
			return resp, err
		}

		outcome := "success"
		if err != nil {
			outcome = "failure"
		}

		_ = application.AuditLogger.Log(ctx, auditlog.LogEntry{
			TenantID:     claims.TenantID,
			Action:       action,
			ResourceType: resourceType,
			// The calling principal, as carried by the token. The internal
			// actor_client_id is not resolvable here without a lookup per call, so the
			// service name is recorded in the change payload instead.
			ResourceID: "",
			Changes:    grpcAuditChanges(info.FullMethod, claims),
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
	default:
		// Get/List/Introspect/Authorize and anything else non-mutating.
		return "", "", false
	}

	if resourceType == "" {
		resourceType = "unknown"
	}
	return action, strings.ToLower(resourceType), true
}

// grpcAuditChanges records the calling principal and the exact method, so a reviewer
// can tell which control-plane service made the call.
func grpcAuditChanges(fullMethod string, claims *middleware.JWTClaims) string {
	principal := claims.Service
	if principal == "" {
		principal = claims.Sub
	}
	return `{"transport":"grpc","method":"` + fullMethod + `","principal":"` + principal + `"}`
}
