package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

type mockOAuthPARService struct {
	pushFn func(context.Context, OAuthPARRequestDTO, OAuthClientCredentials) (*OAuthPARResponseDTO, *apperror.OAuthError)
}

func (m *mockOAuthPARService) Push(ctx context.Context, req OAuthPARRequestDTO, creds OAuthClientCredentials) (*OAuthPARResponseDTO, *apperror.OAuthError) {
	if m.pushFn != nil {
		return m.pushFn(ctx, req, creds)
	}
	return nil, nil
}
func (m *mockOAuthPARService) ConsumeRequestURI(ctx context.Context, requestURI string) (*OAuthAuthorizeRequestDTO, *apperror.OAuthError) {
	return nil, nil
}

func TestOAuthPARHandler_Push(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/oauth/par", errReader{})
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthPARHandler(&mockOAuthPARService{}).Push(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/par", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthPARHandler(&mockOAuthPARService{}).Push(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthPARService{
			pushFn: func(context.Context, OAuthPARRequestDTO, OAuthClientCredentials) (*OAuthPARResponseDTO, *apperror.OAuthError) {
				return &OAuthPARResponseDTO{RequestURI: "urn:ietf:params:oauth:request_uri:abc"}, nil
			},
		}
		r := formPost(t, "/oauth/par", url.Values{
			"response_type":         {"code"},
			"client_id":             {"app"},
			"redirect_uri":          {"https://example.com/cb"},
			"code_challenge":        {strings.Repeat("x", 43)},
			"code_challenge_method": {"S256"},
		})
		w := httptest.NewRecorder()
		NewOAuthPARHandler(svc).Push(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthPARService{
			pushFn: func(context.Context, OAuthPARRequestDTO, OAuthClientCredentials) (*OAuthPARResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidClient("bad client")
			},
		}
		r := formPost(t, "/oauth/par", url.Values{
			"response_type":         {"code"},
			"client_id":             {"app"},
			"redirect_uri":          {"https://example.com/cb"},
			"code_challenge":        {strings.Repeat("x", 43)},
			"code_challenge_method": {"S256"},
		})
		w := httptest.NewRecorder()
		NewOAuthPARHandler(svc).Push(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
