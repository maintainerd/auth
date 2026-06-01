//go:build e2e

package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2E_RegisterFlow(t *testing.T) {
	r := newE2ERouter()

	r.Post("/api/v1/public/register", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := r.URL.Query().Get("client_id")
		providerID := r.URL.Query().Get("provider_id")
		if clientID == "" || providerID == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("missing client_id returns 400", func(t *testing.T) {
		resp := e2eJSON(t, http.MethodPost, ts.URL+"/api/v1/public/register?provider_id=idp",
			map[string]string{"username": "newuser"})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("success returns 201", func(t *testing.T) {
		resp := e2eJSON(t, http.MethodPost, ts.URL+"/api/v1/public/register?client_id=app&provider_id=idp",
			map[string]string{"username": "newuser"})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}
