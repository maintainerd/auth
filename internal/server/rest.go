package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	securityMiddleware "github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/sms"
	"github.com/maintainerd/maintainerd-auth/internal/webui"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	defaultInternalPort = ":8080"
	defaultPublicPort   = ":8081"
	defaultMgmtPort     = ":8082"
	defaultConsolePort  = ":3000"
	defaultIdentityPort = ":3001"
)

// portOrDefault normalizes an env-provided port, accepting either ":3000" or
// "3000" and falling back to def when empty.
func portOrDefault(v, def string) string {
	if v == "" {
		return def
	}
	if !strings.HasPrefix(v, ":") {
		return ":" + v
	}
	return v
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// StartRESTServer launches the internal and public HTTP servers, blocks until
// a termination signal is received, then drains connections gracefully.
// Returns an error if either server fails to start or shut down cleanly.
func StartRESTServer(application *Application) error {
	h := initHandlers(application)

	securityMiddleware.SetStepUpTTLReader(&stepUpTTLAdapter{svc: application.SecuritySettingService})
	email.RedisClient = application.RedisClient
	sms.RedisClient = application.RedisClient

	// Build the routers once. The API servers wrap them in OTel instrumentation;
	// the SPA servers (release image only) mount the SAME raw routers same-origin
	// behind the bundled frontends, so the console/identity apps reach their APIs
	// in-process without a separate proxy.
	internalRouter := buildInternalRouter(h, application)
	publicRouter := buildPublicRouter(h, application)

	internalSrv := newHTTPServer(defaultInternalPort, otelhttp.NewHandler(internalRouter, "internal"))
	publicSrv := newHTTPServer(defaultPublicPort, otelhttp.NewHandler(publicRouter, "public"))

	managementAddr := config.ManagementPort
	if managementAddr == "" {
		managementAddr = defaultMgmtPort
	}
	managementSrv := newHTTPServer(managementAddr, otelhttp.NewHandler(buildManagementRouter(application), "management"))

	// Servers to run and drain. The two SPA servers are appended only in the
	// embed (release) build; in dev they are absent and the frontends run under
	// vite, so nothing here changes for local development.
	type namedServer struct {
		name string
		srv  *http.Server
	}
	servers := []namedServer{
		{"internal", internalSrv},
		{"public", publicSrv},
		{"management", managementSrv},
	}

	if webui.Enabled {
		consoleSrv := newHTTPServer(
			portOrDefault(os.Getenv("APP_CONSOLE_PORT"), defaultConsolePort),
			webui.Console(internalRouter, publicRouter, os.Getenv("APP_FRONTEND_IDENTITY_HOSTNAME")),
		)
		identitySrv := newHTTPServer(
			portOrDefault(os.Getenv("APP_IDENTITY_PORT"), defaultIdentityPort),
			webui.Identity(publicRouter),
		)
		servers = append(servers,
			namedServer{"console", consoleSrv},
			namedServer{"identity", identitySrv},
		)
	}

	listenErr := make(chan error, len(servers))
	for _, s := range servers {
		s := s
		go func() {
			slog.Info("REST server starting", "name", s.name, "addr", s.srv.Addr)
			if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("REST server error", "name", s.name, "error", err)
				listenErr <- err
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("Shutdown signal received, draining connections...")
	case err := <-listenErr:
		return fmt.Errorf("server listen failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownErr error
	for _, s := range servers {
		if err := s.srv.Shutdown(ctx); err != nil {
			shutdownErr = err
			slog.Error("Server shutdown error", "name", s.name, "error", err)
		}
	}

	if shutdownErr != nil {
		return fmt.Errorf("server shutdown failed: %w", shutdownErr)
	}
	slog.Info("Servers stopped cleanly")
	return nil
}
