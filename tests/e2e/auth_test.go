//go:build e2e

// Package e2e_test contains end-to-end tests for the auth service API.
// These tests start an httptest.Server with the actual chi router and
// middleware chain, send real HTTP requests, and assert on responses.
//
// Run with: go test ./tests/e2e/... -tags e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_LoginFlow(t *testing.T) {
	r := newE2ERouter()

	// Public login endpoint
	r.Post("/api/v1/public/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		clientID := r.FormValue("client_id")
		if clientID == "" {
			response.Error(w, http.StatusBadRequest, "client_id required")
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" || password == "" {
			response.Error(w, http.StatusBadRequest, "username and password required")
			return
		}
		if username == "valid" && password == "secret" {
			response.Success(w, map[string]any{
				"access_token":  "e2e-jwt-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "e2e-refresh",
			}, "Login successful")
			return
		}
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
	}))

	// Protected endpoint behind JWT auth
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)
		r.Get("/api/v1/protected", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response.Success(w, map[string]string{"message": "authenticated"}, "")
		}))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("successful login returns tokens", func(t *testing.T) {
		resp := e2eForm(t, http.MethodPost, ts.URL+"/api/v1/public/login?client_id=app", map[string]string{
			"username": "valid",
			"password": "secret",
		})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body struct {
			Data struct {
				AccessToken string `json:"access_token"`
				TokenType   string `json:"token_type"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "e2e-jwt-token", body.Data.AccessToken)
		assert.Equal(t, "Bearer", body.Data.TokenType)
	})

	t.Run("invalid credentials returns 401", func(t *testing.T) {
		resp := e2eForm(t, http.MethodPost, ts.URL+"/api/v1/public/login?client_id=app", map[string]string{
			"username": "invalid",
			"password": "wrong",
		})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("missing client_id returns 400", func(t *testing.T) {
		resp := e2eForm(t, http.MethodPost, ts.URL+"/api/v1/public/login", map[string]string{
			"username": "valid",
			"password": "secret",
		})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("protected endpoint without token returns 401", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/api/v1/protected")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("protected endpoint with bad token returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/protected", nil)
		req.Header.Set("Authorization", "Bearer bad-token")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
