package middleware

import (
	"context"

	"github.com/google/uuid"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
)

// GRPCActor resolves the human principal a mutating gRPC call is acting as.
//
// This is THE definition of "who is doing this" on the gRPC surface, and it must
// have exactly one implementation. It previously existed as three private copies
// (client, iam, tenant) that had already diverged: two consulted the raw JWT
// claims when the auth context was empty and one did not, so the same token
// could mutate a client but not a tenant. Duplicated authorization logic that
// disagrees with itself is how a policy fix lands on one surface and silently
// misses three.
//
// The gRPC interceptor populates the auth context from the VERIFIED token
// (see internal/server/grpc_interceptors.go), so that is the single source of
// truth. There is deliberately no fallback to a request-body actor field: that
// was the original cross-tenant takeover, where a caller named any user and the
// tenant check was then evaluated against their identity instead of the
// caller's.
//
// Fails closed. The gRPC surface authenticates SERVICE principals, which carry
// no user identity, so a service token acting alone is refused rather than
// silently attributed to nobody.
func GRPCActor(ctx context.Context, operation string) (*authctx.AuthUser, error) {
	if auth := AuthFromContext(ctx); auth != nil && auth.User != nil {
		return auth.User, nil
	}
	return nil, apperror.ToGRPCError(apperror.NewForbidden(
		operation + " requires a user principal; this token carries no user identity to act as"))
}

// GRPCActorUUID is GRPCActor for the callers that only need the actor's UUID.
func GRPCActorUUID(ctx context.Context, operation string) (uuid.UUID, error) {
	actor, err := GRPCActor(ctx, operation)
	if err != nil {
		return uuid.Nil, err
	}
	return actor.UserUUID, nil
}

// GRPCPrincipal is the verified principal behind a gRPC call when a HUMAN actor
// is not required: either a user actor (User != nil, resolved from the signed
// on_behalf_of claim exactly like GRPCActor) or a bare SERVICE principal
// (ServiceName != "", the token's `svc` claim). Exactly one of the two is set.
type GRPCPrincipal struct {
	User        *authctx.AuthUser
	ServiceName string
	// TenantID is the internal PK of the tenant the TOKEN is bound to (resolved
	// from the verified tenant_id claim by the interceptor). Handlers that admit a
	// service principal MUST compare it against the target tenant themselves:
	// unlike a user actor, a service has no user membership for the service
	// layer's ValidateTenantAccess to check, so this field is the only tenant
	// boundary the service-actor path has.
	TenantID int64
}

// GRPCActorOrService resolves the acting principal as EITHER a user actor OR a
// service principal, for the few operations that deliberately admit an
// autonomous service (e.g. the core orchestrator provisioning a machine
// client's credential for a service it manages).
//
// The user actor is preferred and resolved exactly as GRPCActor does, so a
// token that names a human keeps being attributed to that human. Only when the
// token carries no user identity at all does the service principal come into
// play — and the CALLER of this helper still owns the decision of what a bare
// service token may do; this helper only answers "who is this, verifiably".
//
// Fails closed, like GRPCActor: a context carrying neither a user nor a
// service principal is refused, never resolved to "nobody".
func GRPCActorOrService(ctx context.Context, operation string) (*GRPCPrincipal, error) {
	claims := JWTClaimsFromContext(ctx)
	if auth := AuthFromContext(ctx); auth != nil && auth.User != nil {
		principal := &GRPCPrincipal{User: auth.User}
		if claims != nil {
			principal.TenantID = claims.TenantID
		}
		return principal, nil
	}
	if claims != nil && claims.Service != "" {
		return &GRPCPrincipal{ServiceName: claims.Service, TenantID: claims.TenantID}, nil
	}
	return nil, apperror.ToGRPCError(apperror.NewForbidden(
		operation + " requires a user or service principal; this token carries neither identity to act as"))
}
