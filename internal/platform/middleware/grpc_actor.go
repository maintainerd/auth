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
