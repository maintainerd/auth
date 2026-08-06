package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientRoute_ProtectedRoutesRequireAuth(t *testing.T) {
	router := chi.NewRouter()
	ClientRoute(router, NewClientHandler(&mockClientService{}), nil, nil)

	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/clients/"},
		{"GET", "/clients/uuid-1"},
		{"POST", "/clients/uuid-1/rotate-secret"},
		{"GET", "/clients/uuid-1/config"},
		{"POST", "/clients/"},
		{"PUT", "/clients/uuid-1"},
		{"PUT", "/clients/uuid-1/status"},
		{"DELETE", "/clients/uuid-1"},
		{"GET", "/clients/uuid-1/uris"},
		{"POST", "/clients/uuid-1/uris"},
		{"PUT", "/clients/uuid-1/uris/uuid-uri"},
		{"DELETE", "/clients/uuid-1/uris/uuid-uri"},
		{"GET", "/clients/uuid-1/apis"},
		{"POST", "/clients/uuid-1/apis"},
		{"DELETE", "/clients/uuid-1/apis/uuid-2"},
		{"GET", "/clients/uuid-1/apis/uuid-2/permissions"},
		{"POST", "/clients/uuid-1/apis/uuid-2/permissions"},
		{"DELETE", "/clients/uuid-1/apis/uuid-2/permissions/uuid-3"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// GET /clients/{uuid}/secret used to be routed behind the client:secret:read
// permission plus a step-up, and its handler answered 410 Gone unconditionally.
// TestClientHandler_GetSecretByUUID asserted exactly that 410, which encoded the
// broken shape as intended behaviour: a permission the seeder grants, a step-up
// the operator must satisfy, and nothing reachable behind either. Secrets are
// bcrypt hashed at rest, so no read is possible at all — the route is gone and
// must stay gone.
func TestClientRoute_HasNoSecretReadRoute(t *testing.T) {
	router := chi.NewRouter()
	ClientRoute(router, NewClientHandler(&mockClientService{}), nil, nil)

	// Walk the registered routes rather than issuing a request: the group's
	// JWTAuthMiddleware answers 401 before chi's NotFound handler runs, so an
	// unregistered path is indistinguishable from a registered one by status code
	// alone.
	registered := map[string]bool{}
	require.NoError(t, chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	}))

	assert.False(t, registered["GET /clients/{client_uuid}/secret"], "the secret-read route must not come back")
	assert.True(t, registered["POST /clients/{client_uuid}/rotate-secret"], "rotation is the only secret surface")
}
