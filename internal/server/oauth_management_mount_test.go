package server

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// The key-lifecycle and dynamic-client-registration surfaces are mounted by a
// SINGLE line in buildInternalRouter. Nothing asserted that line existed, so
// deleting it left RFC 7591/7592 and the entire signing-key lifecycle reachable
// on no port at all — with every package still green.
//
// This walks the router that is actually built, so it fails when the mount line
// is removed, not merely when its handlers are nil.
func TestInternalRouterMountsOAuthManagementSurface(t *testing.T) {
	application := &Application{}
	handlers := initHandlers(application)

	mux, ok := buildInternalRouter(handlers, application).(*chi.Mux)
	if !ok {
		t.Fatalf("internal router is not a *chi.Mux")
	}

	required := []struct{ method, path string }{
		// Signing-key lifecycle: without these an operator cannot rotate, retire,
		// or mark a key compromised through any interface.
		{http.MethodGet, "/api/v1/oauth/signing-keys"},
		{http.MethodPost, "/api/v1/oauth/signing-keys/rotate"},
		{http.MethodPost, "/api/v1/oauth/signing-keys/abc/retire"},
		{http.MethodPost, "/api/v1/oauth/signing-keys/abc/compromise"},
		// Dynamic client registration (RFC 7591 / 7592).
		{http.MethodPost, "/api/v1/oauth/register"},
		{http.MethodGet, "/api/v1/oauth/register/some-client-id"},
	}

	for _, route := range required {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			if !mux.Match(chi.NewRouteContext(), route.method, route.path) {
				t.Fatalf("%s %s is not mounted on the internal router — the OAuth "+
					"management mount in buildInternalRouter is missing", route.method, route.path)
			}
		})
	}
}

// The same surfaces must NOT appear on the public listener. Key rotation and
// client registration are management operations; exposing them to the internet
// would let anyone enumerate or rotate this deployment's signing keys.
func TestPublicRouterOmitsOAuthManagementSurface(t *testing.T) {
	application := &Application{}
	handlers := initHandlers(application)

	mux, ok := buildPublicRouter(handlers, application).(*chi.Mux)
	if !ok {
		t.Fatalf("public router is not a *chi.Mux")
	}

	forbidden := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/oauth/signing-keys"},
		{http.MethodPost, "/api/v1/oauth/signing-keys/rotate"},
		{http.MethodPost, "/api/v1/oauth/signing-keys/abc/retire"},
		{http.MethodPost, "/api/v1/oauth/signing-keys/abc/compromise"},
		{http.MethodPost, "/api/v1/oauth/register"},
		{http.MethodGet, "/api/v1/oauth/register/some-client-id"},
	}

	for _, route := range forbidden {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			if mux.Match(chi.NewRouteContext(), route.method, route.path) {
				t.Fatalf("%s %s must not be reachable on the public listener", route.method, route.path)
			}
		})
	}
}
