package server

import (
	"net/http"
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
