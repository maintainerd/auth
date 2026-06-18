package authevent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthEventRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	AuthEventRoute(router, NewAuthEventHandler(&mockAuthEventService{}), nil, nil)

	for _, path := range []string{
		"/auth-events/",
		"/auth-events/count",
		"/auth-events/export",
		"/auth-events/" + testResourceUUID.String(),
	} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
