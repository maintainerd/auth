package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestInitHandlersAndRouters(t *testing.T) {
	application := &Application{}
	handlers := initHandlers(application)
	if handlers == nil || handlers.authorization == nil {
		t.Fatal("initHandlers did not create authorization handler")
	}

	tests := []struct {
		name   string
		router http.Handler
		method string
		path   string
	}{
		{"internal", buildInternalRouter(handlers, application), http.MethodPost, "/api/v1/authorize/"},
		{"management", buildManagementRouter(application), http.MethodGet, "/health"},
		{"public", buildPublicRouter(handlers, application), http.MethodPost, "/api/v1/oauth/token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, ok := tt.router.(*chi.Mux)
			if !ok {
				t.Fatalf("router type = %T", tt.router)
			}
			match := chi.NewRouteContext()
			if !mux.Match(match, tt.method, tt.path) {
				t.Fatalf("route %s %s not registered", tt.method, tt.path)
			}
		})
	}
}

func TestInternalRouterDoesNotMountInteractiveAuthRoutes(t *testing.T) {
	application := &Application{}
	handlers := initHandlers(application)
	mux, ok := buildInternalRouter(handlers, application).(*chi.Mux)
	if !ok {
		t.Fatalf("router type was not *chi.Mux")
	}

	removedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/login"},
		{http.MethodPost, "/api/v1/login/mfa/verify"},
		{http.MethodPost, "/api/v1/register"},
		{http.MethodPost, "/api/v1/register/invite"},
		{http.MethodPost, "/api/v1/forgot-password"},
		{http.MethodPost, "/api/v1/reset-password"},
		{http.MethodPost, "/api/v1/email-verification/send"},
		{http.MethodPost, "/api/v1/sms-login/send"},
		{http.MethodPost, "/api/v1/recovery/backup-code"},
	}

	for _, route := range removedRoutes {
		t.Run(route.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			if mux.Match(match, route.method, route.path) {
				t.Fatalf("route %s %s should not be registered on the internal router", route.method, route.path)
			}
		})
	}
}

// The internal listener (8080) is management-only and will sit behind a VPN.
// Nothing that MINTS a credential may be reachable there — token issuance,
// refresh, and the OIDC protocol surface belong on the public listener (8081).
//
// Unlike the list above, this walks every route actually mounted and matches by
// pattern, so a newly added token endpoint fails the build instead of quietly
// widening the internal surface.
func TestInternalRouterMintsNoCredentials(t *testing.T) {
	application := &Application{}
	handlers := initHandlers(application)
	mux, ok := buildInternalRouter(handlers, application).(*chi.Mux)
	if !ok {
		t.Fatalf("router type was not *chi.Mux")
	}

	// Substrings that identify a credential-issuing or interactive-auth route.
	// "/oauth/introspect" is deliberately allowed: it validates a token rather
	// than issuing one, and keeping it off the public listener is the safer side.
	forbidden := []string{
		"/oauth/token", "/oauth/authorize", "/oauth/par", "/oauth/device",
		"/oauth/ciba", "/oauth/callback", "/oauth/broker", "/oauth/userinfo",
		"/refresh-token", "/magic-link", "/sms-login", "/federation/token",
		"/.well-known/",
	}

	var offenders []string
	err := chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		for _, bad := range forbidden {
			if strings.Contains(route, bad) {
				offenders = append(offenders, method+" "+route)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("credential-issuing routes must not be on the internal listener: %v", offenders)
	}
}

// The mirror of the above: the public listener must not expose management.
func TestPublicRouterExposesNoManagement(t *testing.T) {
	application := &Application{}
	handlers := initHandlers(application)
	mux, ok := buildPublicRouter(handlers, application).(*chi.Mux)
	if !ok {
		t.Fatalf("router type was not *chi.Mux")
	}

	// Administrative collections. "/api/v1/tenant/" and "/api/v1/client" are
	// intentionally public: the hosted login page resolves branding through them
	// before anyone is authenticated.
	forbidden := []string{
		"/api/v1/users", "/api/v1/roles", "/api/v1/permissions", "/api/v1/policies",
		"/api/v1/services", "/api/v1/apis", "/api/v1/clients", "/api/v1/identity-providers",
		"/api/v1/security-settings", "/api/v1/tenant-settings", "/api/v1/tenants",
		"/api/v1/setup", "/api/v1/invites", "/api/v1/workload",
	}

	var offenders []string
	err := chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		for _, bad := range forbidden {
			if strings.Contains(route, bad) {
				offenders = append(offenders, method+" "+route)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("management routes must not be on the public listener: %v", offenders)
	}
}
