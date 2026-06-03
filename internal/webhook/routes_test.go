package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestWebhookEndpointRoute(t *testing.T) {
	t.Run("registers protected routes", func(t *testing.T) {
		router := chi.NewRouter()

		WebhookEndpointRoute(router, NewWebhookEndpointHandler(&mockWebhookEndpointService{}), nil, nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhook-endpoints/", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
