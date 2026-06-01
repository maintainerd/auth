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
}
