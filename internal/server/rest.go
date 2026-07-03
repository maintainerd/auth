package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	securityMiddleware "github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/sms"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	defaultInternalPort = ":8080"
	defaultPublicPort   = ":8081"
	defaultMgmtPort     = ":8082"
)

// StartRESTServer launches the internal and public HTTP servers, blocks until
// a termination signal is received, then drains connections gracefully.
// Returns an error if either server fails to start or shut down cleanly.
func StartRESTServer(application *Application) error {
	h := initHandlers(application)

	securityMiddleware.SetStepUpTTLReader(&stepUpTTLAdapter{svc: application.SecuritySettingService})
	email.RedisClient = application.RedisClient
	sms.RedisClient = application.RedisClient

	internalSrv := &http.Server{
		Addr:         defaultInternalPort,
		Handler:      otelhttp.NewHandler(buildInternalRouter(h, application), "internal"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	publicSrv := &http.Server{
		Addr:         defaultPublicPort,
		Handler:      otelhttp.NewHandler(buildPublicRouter(h, application), "public"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	managementAddr := config.ManagementPort
	if managementAddr == "" {
		managementAddr = defaultMgmtPort
	}
	managementSrv := &http.Server{
		Addr:         managementAddr,
		Handler:      otelhttp.NewHandler(buildManagementRouter(application), "management"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	listenErr := make(chan error, 3)

	go func() {
		slog.Info("Internal REST server starting", "addr", internalSrv.Addr)
		if err := internalSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Internal REST server error", "error", err)
			listenErr <- err
		}
	}()

	go func() {
		slog.Info("Public REST server starting", "addr", publicSrv.Addr)
		if err := publicSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Public REST server error", "error", err)
			listenErr <- err
		}
	}()

	go func() {
		slog.Info("Management REST server starting", "addr", managementSrv.Addr)
		if err := managementSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Management REST server error", "error", err)
			listenErr <- err
		}
	}()

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
	if err := internalSrv.Shutdown(ctx); err != nil {
		shutdownErr = err
		slog.Error("Internal server shutdown error", "error", err)
	}
	if err := publicSrv.Shutdown(ctx); err != nil {
		shutdownErr = err
		slog.Error("Public server shutdown error", "error", err)
	}
	if err := managementSrv.Shutdown(ctx); err != nil {
		shutdownErr = err
		slog.Error("Management server shutdown error", "error", err)
	}

	if shutdownErr != nil {
		return fmt.Errorf("server shutdown failed: %w", shutdownErr)
	}
	slog.Info("Servers stopped cleanly")
	return nil
}
