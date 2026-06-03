package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

type mockOAuthTokenExchangeService struct {
	exchangeFn func(context.Context, OAuthTokenExchangeRequestDTO, OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError)
}

func (m *mockOAuthTokenExchangeService) Exchange(ctx context.Context, req OAuthTokenExchangeRequestDTO, creds OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
	if m.exchangeFn != nil {
		return m.exchangeFn(ctx, req, creds)
	}
	return nil, nil
}

func TestOAuthTokenExchangeHandler_Exchange(t *testing.T) {
	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/token", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}).Exchange(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthTokenExchangeService{
			exchangeFn: func(context.Context, OAuthTokenExchangeRequestDTO, OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidClient("bad client")
			},
		}
		r := formPost(t, "/oauth/token", url.Values{
			"client_id":          {"app"},
			"subject_token":      {"subject"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(svc).Exchange(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthTokenExchangeService{
			exchangeFn: func(_ context.Context, req OAuthTokenExchangeRequestDTO, creds OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
				assert.Equal(t, "subject", req.SubjectToken)
				assert.Equal(t, "app", req.ClientID)
				assert.Equal(t, "app", creds.ClientID)
				return &OAuthTokenExchangeResponseDTO{AccessToken: "token", TokenType: "Bearer", ExpiresIn: 60}, nil
			},
		}
		r := formPost(t, "/oauth/token", url.Values{
			"client_id":          {"app"},
			"subject_token":      {"subject"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(svc).Exchange(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
