package oauth

import (
	"context"
	"testing"

	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
)

type testOAuthTokenService struct {
	introspectFn func(ctx context.Context, req OAuthIntrospectRequestDTO, creds OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError)
}

func (m *testOAuthTokenService) Exchange(ctx context.Context, req OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
	return nil, nil
}
func (m *testOAuthTokenService) Revoke(ctx context.Context, req OAuthRevokeRequestDTO, creds OAuthClientCredentials) *apperror.OAuthError {
	return nil
}
func (m *testOAuthTokenService) Introspect(ctx context.Context, req OAuthIntrospectRequestDTO, creds OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
	return m.introspectFn(ctx, req, creds)
}

func TestOAuthIntrospectionGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()

	t.Run("active token", func(t *testing.T) {
		svc := &testOAuthTokenService{
			introspectFn: func(ctx context.Context, req OAuthIntrospectRequestDTO, creds OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
				return &OAuthIntrospectResponseDTO{Active: true, ClientID: "client-1", Scope: "read", TokenType: "Bearer"}, nil
			},
		}
		h := NewOAuthIntrospectionGRPCHandler(svc)
		res, err := h.Introspect(ctx, &authv1.IntrospectRequest{Token: "valid-token"})
		if err != nil { t.Fatal(err) }
		if !res.Active { t.Error("expected active") }
	})

	t.Run("inactive token", func(t *testing.T) {
		svc := &testOAuthTokenService{
			introspectFn: func(ctx context.Context, req OAuthIntrospectRequestDTO, creds OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
				return nil, &apperror.OAuthError{Code: "invalid_token"}
			},
		}
		h := NewOAuthIntrospectionGRPCHandler(svc)
		res, err := h.Introspect(ctx, &authv1.IntrospectRequest{Token: "bad-token"})
		if err != nil { t.Fatal(err) }
		if res.Active { t.Error("expected inactive") }
	})
}
