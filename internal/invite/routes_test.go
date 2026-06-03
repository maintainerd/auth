package invite

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestInviteRoute_ProtectedRouteRequiresAuth(t *testing.T) {
	router := chi.NewRouter()
	InviteRoute(router, NewInviteHandler(&mockInviteService{}), nil, nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/invite/", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
