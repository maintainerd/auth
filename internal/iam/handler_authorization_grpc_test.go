package iam

import (
	"context"
	"testing"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationGRPCHandler_Authorize(t *testing.T) {
	h := NewAuthorizationGRPCHandler(&mockAuthorizationService{
		authorizeFn: func(req AuthzRequest) Decision {
			assert.Equal(t, "auth", req.Principal)
			return Decision{Allowed: true, Reason: "matched allow"}
		},
	})
	res, err := h.Authorize(context.Background(), &authv1.AuthorizeRequest{Principal: "auth", Action: "user:read", Resource: "user:*"})
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, "matched allow", res.Reason)
}
