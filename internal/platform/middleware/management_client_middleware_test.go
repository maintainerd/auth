package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubManagementResolver struct{ mgmt map[string]bool }

func (s stubManagementResolver) IsManagementClient(_ context.Context, id string) bool {
	return s.mgmt[id]
}

func TestRequireManagementClient(t *testing.T) {
	initTestJWTKeys(t)

	resolver := stubManagementResolver{mgmt: map[string]bool{"mgmt-client": true}}
	h := RequireManagementClient(resolver)(okHandler())

	mint := func(clientID string) string {
		// aud and client_id both carry the client identifier, mirroring the real
		// token mint (service_token.go sets audience = *client.Identifier).
		tok, err := jwt.GenerateAccessToken(
			uuid.New().String(), "read", "https://auth.example.com",
			clientID, clientID, "provider-1",
		)
		require.NoError(t, err)
		return tok
	}
	mintWithAudience := func(clientID, audience string) string {
		tok, err := jwt.GenerateAccessToken(
			uuid.New().String(), "read", "https://auth.example.com",
			audience, clientID, "provider-1",
		)
		require.NoError(t, err)
		return tok
	}

	serve := func(setup func(*http.Request)) int {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
		setup(r)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}

	t.Run("no token passes through to per-route auth", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serve(func(*http.Request) {}))
	})

	t.Run("API key passes through", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serve(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer ak_secret")
		}))
	})

	t.Run("invalid token passes through (per-route JWT returns the 401)", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serve(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer not-a-jwt")
		}))
	})

	t.Run("management client token is accepted", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serve(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+mint("mgmt-client"))
		}))
	})

	t.Run("non-management client token is rejected with 403", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, serve(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+mint("some-other-client"))
		}))
	})

	t.Run("management client claim with a different audience is rejected", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, serve(func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+mintWithAudience("mgmt-client", "other-client"))
		}))
	})

	t.Run("management client token via cookie is accepted", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serve(func(r *http.Request) {
			r.AddCookie(&http.Cookie{Name: "__Host-access_token", Value: mint("mgmt-client")})
		}))
	})
}
