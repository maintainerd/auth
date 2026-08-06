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
	startRetentionRunner          = authevent.StartRetentionRunner
	startTenantRetentionRunner    = tenant.StartRetentionRunner
	startKeyRotationRunner        = runner.StartKeyRotationRunner
	startSigningKeyRotationRunner = oauth.StartSigningKeyRotationRunner
	startSecretRefreshRunner      = runner.StartSecretRefreshRunner
	startCleanupRunner            = oauth.StartCleanupRunner
	startDataErasureWorker        = user.StartDataErasureWorker
	startGRPCServer               = appserver.StartGRPCServer
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
	// Automatic rotation only happens when the signing_keys table owns the key.
	//
	// With JWT_PRIVATE_KEY set the key is operator-managed and process-local, and
	// rotating it automatically is actively harmful: the new key is never
	// persisted, is not shared with any other replica, and is lost on restart. In
	// a multi-replica deployment — which is what packaging this as an image
	// invites — replica A would start signing with a key that replica B's JWKS
	// never publishes, so B rejects A's tokens. An operator rotates an env-owned
	// key by changing the variable and redeploying; the server must not pretend
	// to do it for them.
	//
	// The DB-backed runner is safe to run on every replica: the key lives in
	// shared state, so all replicas converge on the same active key.
	if len(config.JWTPrivateKey) > 0 {
		slog.Info("key_rotation: JWT_PRIVATE_KEY is set, so the signing key is operator-managed; " +
			"automatic rotation is disabled (rotate by updating the variable and redeploying)")
	} else {
		go startSigningKeyRotationRunner(ctx, application.DB, keyRotationPeriod)
	}

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
