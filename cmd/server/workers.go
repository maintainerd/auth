package main

import (
	"context"
	"log/slog"

	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/authevent"
	appserver "github.com/maintainerd/auth/internal/server"
)

// startBackgroundWorkers launches non-REST runtimes that should stop when the
// bootstrap context is cancelled during shutdown.
func startBackgroundWorkers(ctx context.Context, application *app.App, serverApplication *appserver.Application) {
	// Retention runs periodically and exits when ctx is cancelled.
	go authevent.StartRetentionRunner(ctx, application.AuthEventService, authevent.DefaultRetentionPeriod, authevent.DefaultRetentionInterval)

	// gRPC is best-effort alongside REST: failures are logged here while REST
	// keeps owning process lifetime and graceful shutdown.
	go func() {
		if err := appserver.StartGRPCServer(ctx, serverApplication); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()
}
