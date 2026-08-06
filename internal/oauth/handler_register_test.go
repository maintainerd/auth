package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockOAuthRegisterService struct {
	registerFn func(context.Context, OAuthClientRegistrationRequestDTO, int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError)
	readFn     func(context.Context, string, int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError)
}

func (m *mockOAuthRegisterService) Register(ctx context.Context, req OAuthClientRegistrationRequestDTO, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
	if m.registerFn != nil {
		return m.registerFn(ctx, req, tenantID)
	}
	return nil, nil
}

func (m *mockOAuthRegisterService) Read(ctx context.Context, clientIdentifier string, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
	if m.readFn != nil {
		return m.readFn(ctx, clientIdentifier, tenantID)
	}
	return nil, nil
}

func TestOAuthRegisterHandler_Register(t *testing.T) {
	// Every case now goes through withUser: registration is authenticated and
	// tenant-scoped (RFC 7591 §3 initial access token), so an anonymous request
	// can no longer reach the service at all.
	t.Run("unauthenticated returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/oauth/register", map[string]any{
			"client_name":   "My App",
			"redirect_uris": []string{"https://example.com/cb"},
		})
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}).Register(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		r := withUser(badJSONReq(t, http.MethodPost, "/oauth/register"))
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}).Register(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withUser(jsonReq(t, http.MethodPost, "/oauth/register", map[string]any{}))
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}).Register(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthRegisterService{
			registerFn: func(context.Context, OAuthClientRegistrationRequestDTO, int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidRequest("bad redirect_uri")
			},
		}
		r := withUser(jsonReq(t, http.MethodPost, "/oauth/register", map[string]any{
			"client_name":          "My App",
			"redirect_uris":        []string{"https://example.com/cb"},
			"identity_provider_id": 1,
		}))
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(svc).Register(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("the caller's tenant is what reaches the service", func(t *testing.T) {
		var got int64
		svc := &mockOAuthRegisterService{
			registerFn: func(_ context.Context, _ OAuthClientRegistrationRequestDTO, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
				got = tenantID
				return &OAuthClientRegistrationResponseDTO{ClientID: "new-app"}, nil
			},
		}
		r := withUser(jsonReq(t, http.MethodPost, "/oauth/register", map[string]any{
			"client_name":          "My App",
			"redirect_uris":        []string{"https://example.com/cb"},
			"identity_provider_id": 1,
		}))
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(svc).Register(w, r)

		// 201 Created, not 200: registration creates a resource (RFC 7591 §3.2.1).
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, int64(1), got)
		var resp OAuthClientRegistrationResponseDTO
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, "new-app", resp.ClientID)
	})
}

func TestOAuthRegisterHandler_Read(t *testing.T) {
	t.Run("unauthenticated returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/oauth/register/app-1", nil)
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}).Read(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing client_id returns 400", func(t *testing.T) {
		r := withUser(httptest.NewRequest(http.MethodGet, "/oauth/register/", nil))
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(&mockOAuthRegisterService{}).Read(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns the registered metadata", func(t *testing.T) {
		svc := &mockOAuthRegisterService{
			readFn: func(_ context.Context, clientIdentifier string, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
				assert.Equal(t, "app-1", clientIdentifier)
				assert.Equal(t, int64(1), tenantID)
				return &OAuthClientRegistrationResponseDTO{ClientID: "app-1"}, nil
			},
		}
		r := withChiParam(withUser(httptest.NewRequest(http.MethodGet, "/oauth/register/app-1", nil)), "client_id", "app-1")
		w := httptest.NewRecorder()
		NewOAuthRegisterHandler(svc).Read(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp OAuthClientRegistrationResponseDTO
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		assert.Equal(t, "app-1", resp.ClientID)
		// The secret is issued once at registration and never readable again.
		assert.Empty(t, resp.ClientSecret)
	})
}
