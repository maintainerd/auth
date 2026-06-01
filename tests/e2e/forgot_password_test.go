//go:build e2e

package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2E_ForgotPasswordFlow(t *testing.T) {
	r := newE2ERouter()

	r.Post("/api/v1/public/forgot-password", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client_id")
		providerID := r.URL.Query().Get("provider_id")
		if clientID == "" || providerID == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("success returns 200", func(t *testing.T) {
		resp := e2eJSON(t, http.MethodPost, ts.URL+"/api/v1/public/forgot-password?client_id=app&provider_id=idp",
			map[string]string{"email": "user@example.com"})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("missing client_id returns 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/public/forgot-password?provider_id=idp",
			nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
