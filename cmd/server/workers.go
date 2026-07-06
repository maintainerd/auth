package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/app"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/runner"
	appserver "github.com/maintainerd/maintainerd-auth/internal/server"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

var (
	startRetentionRunner       = authevent.StartRetentionRunner
	startTenantRetentionRunner = tenant.StartRetentionRunner
	startKeyRotationRunner     = runner.StartKeyRotationRunner
	startSecretRefreshRunner   = runner.StartSecretRefreshRunner
	startCleanupRunner         = oauth.StartCleanupRunner
	startDataErasureWorker     = user.StartDataErasureWorker
	startGRPCServer            = appserver.StartGRPCServer
)

// startBackgroundWorkers launches non-REST runtimes that should stop when the
// bootstrap context is cancelled during shutdown.
func startBackgroundWorkers(ctx context.Context, application *app.App, serverApplication *appserver.Application) {
	// Retention runs periodically and exits when ctx is cancelled.
	go startRetentionRunner(ctx, application.AuthEventService, application.DB, authevent.DefaultRetentionPeriod, authevent.DefaultRetentionInterval)
	go startTenantRetentionRunner(ctx, application.DB, tenant.DefaultTenantRetentionPeriod, tenant.DefaultTenantRetentionInterval)

	// Ephemeral-row cleanup runs at 5-minute intervals to prevent unbounded
	// growth of short-lived tables (OAuth codes, tokens, challenges, OTPs, etc.).
	go startCleanupRunner(ctx, application.DB, 5*time.Minute)

	// GDPR Article 17 erasure worker: anonymizes users whose erasure request is
	// due and not under legal hold. Distinct from the DELETE-expired cleanup jobs
	// because erasure is a multi-table anonymization.
	go startDataErasureWorker(ctx, application.DataErasureService, 15*time.Minute)

	// Partition manager pre-creates next month's auth_events partition daily.
	authevent.StartPartitionManager(ctx, application.DB, 24*time.Hour)

	keyRotationPeriod := time.Duration(config.JWTKeyRotationPeriodSeconds) * time.Second
	if keyRotationPeriod <= 0 {
		keyRotationPeriod = 24 * time.Hour
	}
	go startKeyRotationRunner(ctx, keyRotationPeriod)

	secretRefreshPeriod := time.Duration(config.SecretRefreshPeriodSeconds) * time.Second
	if secretRefreshPeriod <= 0 {
		secretRefreshPeriod = 5 * time.Minute
	}
	go startSecretRefreshRunner(ctx, secretRefreshPeriod)

	// gRPC is best-effort alongside REST: failures are logged here while REST
	// keeps owning process lifetime and graceful shutdown.
	go func() {
		if err := startGRPCServer(ctx, serverApplication); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()
}
