package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// StartRESTServer launches the internal and public HTTP servers, blocks until
// a termination signal is received, then drains connections gracefully.
func StartRESTServer(application *Application) {
	h := initHandlers(application)

	internalSrv := &http.Server{
		Addr:         ":8080",
		Handler:      otelhttp.NewHandler(buildInternalRouter(h, application), "internal"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	publicSrv := &http.Server{
		Addr:         ":8081",
		Handler:      otelhttp.NewHandler(buildPublicRouter(h, application), "public"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		slog.Info("Internal REST server starting", "addr", internalSrv.Addr)
		if err := internalSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Internal REST server error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		defer wg.Done()
		slog.Info("Public REST server starting", "addr", publicSrv.Addr)
		if err := publicSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Public REST server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutdown signal received, draining connections...")

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

	wg.Wait()

	if shutdownErr != nil {
		os.Exit(1)
	}
	slog.Info("Servers stopped cleanly")
}
