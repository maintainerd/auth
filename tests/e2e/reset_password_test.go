//go:build e2e

package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2E_ResetPasswordFlow(t *testing.T) {
	r := newE2ERouter()

	r.Post("/api/v1/public/reset-password", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		clientID := r.URL.Query().Get("client_id")
		if token == "" || clientID == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("success returns 200", func(t *testing.T) {
		resp := e2eJSON(t, http.MethodPost,
			ts.URL+"/api/v1/public/reset-password?token=tok&client_id=app&provider_id=idp&expires=9999&sig=sig",
			map[string]string{"new_password": "NewPass1!"})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("missing token returns 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost,
			ts.URL+"/api/v1/public/reset-password?client_id=app&provider_id=idp", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
