package iam

import (
	"context"
	"testing"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthorizationGRPCHandler_Authorize(t *testing.T) {
	// The old version of this test asserted the principal came from the request
	// body and never passed a token at all. That is the bug: the body-supplied
	// principal let any valid token probe decisions for any principal, and because
	// the message carries no tenant, TenantID stayed 0 and PolicyBundle rejected
	// every call as "principal bundle unavailable".
	h := NewAuthorizationGRPCHandler(&mockAuthorizationService{
		authorizeFn: func(req AuthzRequest) Decision {
			assert.Equal(t, "auth", req.Principal)
			assert.Equal(t, int64(77), req.TenantID)
			assert.Equal(t, "user:read", req.Action)
			return Decision{Allowed: true, Reason: "matched allow"}
		},
	})
	ctx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		Service: "auth", SubjectType: "service", TenantID: 77,
	})

	res, err := h.Authorize(ctx, &authv1.AuthorizeRequest{Principal: "impersonated", Action: "user:read", Resource: "user:*"})

	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, "matched allow", res.Reason)
}

func TestAuthorizationGRPCHandler_Authorize_RejectsUnusableTokens(t *testing.T) {
	// Reached only if the handler ignores the token and trusts the body.
	h := NewAuthorizationGRPCHandler(&mockAuthorizationService{
		authorizeFn: func(AuthzRequest) Decision {
			t.Fatal("authorization must not be evaluated without a usable token")
			return Decision{}
		},
	})

	for _, tc := range []struct {
		name   string
		ctx    context.Context
		expect codes.Code
	}{
		{
			name:   "no claims",
			ctx:    context.Background(),
			expect: codes.Unauthenticated,
		},
		{
			// A user token has no service principal to answer for.
			name:   "no principal",
			ctx:    middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{SubjectType: "user", TenantID: 77}),
			expect: codes.PermissionDenied,
		},
		{
			// Tenant 0 would be evaluated against whichever tenant's policies were
			// found first, so it is refused rather than silently answered.
			name:   "no tenant",
			ctx:    middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{Service: "auth", SubjectType: "service"}),
			expect: codes.PermissionDenied,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := h.Authorize(tc.ctx, &authv1.AuthorizeRequest{Principal: "auth", Action: "user:read", Resource: "user:*"})

			assert.Nil(t, res)
			assert.Equal(t, tc.expect, status.Code(err))
		})
	}
}

// Transport parity with the REST twin: a resource claiming the MRN scheme that
// cannot be parsed is refused as InvalidArgument and never reaches the
// evaluator, where falling through to legacy matching could glob-match it.
func TestAuthorizationGRPCHandler_Authorize_RefusesMalformedMRNResource(t *testing.T) {
	h := NewAuthorizationGRPCHandler(&mockAuthorizationService{
		authorizeFn: func(AuthzRequest) Decision {
			t.Fatal("a malformed MRN resource must not reach the evaluator")
			return Decision{}
		},
	})
	ctx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		Service: "auth", SubjectType: "service", TenantID: 77,
	})

	res, err := h.Authorize(ctx, &authv1.AuthorizeRequest{Action: "storage:read", Resource: "mrn:storage:acme"})

	assert.Nil(t, res)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
