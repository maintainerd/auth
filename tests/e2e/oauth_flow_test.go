//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_OAuthFlow(t *testing.T) {
	r := newE2ERouter()

	r.Post("/api/v1/oauth/token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		grantType := r.FormValue("grant_type")
		if grantType == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
			return
		}
		if grantType == "authorization_code" {
			code := r.FormValue("code")
			if code == "expired" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant", "error_description": "code expired"})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "e2e-oauth-at",
				"token_type":    "Bearer",
				"expires_in":    900,
				"refresh_token": "e2e-oauth-rt",
				"scope":         "openid profile",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unsupported_grant_type"})
	}))

	r.Get("/.well-known/openid-configuration", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 "https://auth.example.com",
			"authorization_endpoint": "https://auth.example.com/api/v1/oauth/authorize",
			"token_endpoint":         "https://auth.example.com/api/v1/oauth/token",
			"grant_types_supported":  []string{"authorization_code", "refresh_token"},
		})
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("auth code grant returns tokens", func(t *testing.T) {
		resp := e2eForm(t, http.MethodPost, ts.URL+"/api/v1/oauth/token", map[string]string{
			"grant_type":    "authorization_code",
			"code":          "valid-code",
			"redirect_uri":  "https://app.example.com/cb",
			"code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			"client_id":     "app",
		})
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "e2e-oauth-at", body["access_token"])
		assert.Equal(t, "Bearer", body["token_type"])
	})

	t.Run("expired auth code returns 400", func(t *testing.T) {
		resp := e2eForm(t, http.MethodPost, ts.URL+"/api/v1/oauth/token", map[string]string{
			"grant_type":    "authorization_code",
			"code":          "expired",
			"redirect_uri":  "https://app.example.com/cb",
			"code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			"client_id":     "app",
		})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var body map[string]string
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "invalid_grant", body["error"])
	})

	t.Run("missing grant_type returns 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/oauth/token", nil)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("discovery returns well-known config", func(t *testing.T) {
		resp := e2eDo(t, http.MethodGet, ts.URL+"/.well-known/openid-configuration")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.NotEmpty(t, body["authorization_endpoint"])
	})
}
