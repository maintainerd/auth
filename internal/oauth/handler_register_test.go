package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOAuthRegisterService struct {
	registerFn func(context.Context, OAuthClientRegistrationRequestDTO) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError)
}

func (m *mockOAuthRegisterService) Register(ctx context.Context, req OAuthClientRegistrationRequestDTO) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
	if m.registerFn != nil {
		return m.registerFn(ctx, req)
	}
	return nil, nil
}

func TestOAuthRegisterHandler_Register(t *testing.T) {
	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/oauth/register")
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}).Register(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/oauth/register", map[string]any{})
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}).Register(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthRegisterService{
			registerFn: func(context.Context, OAuthClientRegistrationRequestDTO) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidRequest("bad redirect_uri")
			},
		}
		r := jsonReq(t, http.MethodPost, "/oauth/register", map[string]any{
			"client_name":          "My App",
			"redirect_uris":        []string{"https://example.com/cb"},
			"identity_provider_id": 1,
		})
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(svc).Register(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthRegisterService{
			registerFn: func(context.Context, OAuthClientRegistrationRequestDTO) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
				return &OAuthClientRegistrationResponseDTO{ClientID: "new-app"}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/oauth/register", map[string]any{
			"client_name":          "My App",
			"redirect_uris":        []string{"https://example.com/cb"},
			"identity_provider_id": 1,
		})
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(svc).Register(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp OAuthClientRegistrationResponseDTO
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, "new-app", resp.ClientID)
	})
}
