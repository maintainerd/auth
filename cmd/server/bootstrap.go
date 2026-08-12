package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maintainerd/maintainerd-auth/internal/app"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/runner"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/telemetry"
	appserver "github.com/maintainerd/maintainerd-auth/internal/server"
)

// run executes the server bootstrap sequence in dependency order. Keep this as
// orchestration only; reusable infrastructure belongs in internal/platform and
// domain wiring belongs in internal/app.
func run(ctx context.Context) error {
	// Use a temporary JSON logger before config is loaded so early startup
	// failures still emit structured logs.
	initBootstrapLogger()

	if err := config.Init(); err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	// Bring up the OTel log provider before (re)building the logger so the
	// slog→OTLP bridge has a provider to emit through. No-op when OTEL disabled.
	logsShutdown, err := telemetry.InitLogs(ctx)
	if err != nil {
		return fmt.Errorf("initialize OpenTelemetry logging: %w", err)
	}
	defer shutdownWithTimeout("OpenTelemetry logging", logsShutdown)

	// Rebuild the logger once LOG_LEVEL, PII redaction, and OTel logging are known.
	initConfiguredLogger()

	// Announce the security posture so it is never ambiguous which mode is active.
	// APP_ENV defaults to "production" (secure by default): production enforces DB
	// SSL, gRPC TLS, and HTTPS secret stores, sends HSTS, and disables gRPC
	// reflection. "development" relaxes those for local work.
	if config.AppEnv == "production" {
		slog.Info("startup: running in production mode — TLS/SSL enforcement and HSTS active", "app_env", config.AppEnv)
	} else {
		slog.Warn("startup: running in development mode — security hardening relaxed (no HSTS; DB/gRPC TLS and HTTPS secret stores not enforced). Set APP_ENV=production for a hardened deployment.", "app_env", config.AppEnv)
	}

	// Start tracing and metrics before downstream dependencies are initialized
	// so startup failures are observable too.
	telemetryShutdown, err := initTelemetry(ctx)
	if err != nil {
		return err
	}
	defer telemetryShutdown()

	// JWT keys: prefer env-var RSA key pair. If JWT_PRIVATE_KEY is absent the
	// server falls back to a DB-backed auto-generated key after migrations run.
	jwtEnvLoaded := len(config.JWTPrivateKey) > 0
	if jwtEnvLoaded {
		if err := jwt.InitJWTKeys(); err != nil {
			return fmt.Errorf("initialize JWT keys: %w", err)
		}
	}

	// Database and Redis are process-level dependencies shared by repositories,
	// cache, rate limiting, sessions, and background workers.
	db, err := config.InitDB(ctx)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	redisClient, err := config.NewRedisClient(ctx)
	if err != nil {
		return fmt.Errorf("initialize Redis: %w", err)
	}

	// Security helpers use the same Redis client as the app so rate-limit state
	// is shared across transports.
	security.InitRateLimiter(redisClient)

	// Migrations run before dependency wiring so repositories and services only
	// start after the database schema is current.
	if err := runner.RunMigrations(db); err != nil {
		return fmt.Errorf("run database migrations: %w", err)
	}

	// internal/app owns domain composition; cmd/server only receives the fully
	// wired app and adapts it to transport runtime dependencies.
	application, err := app.NewApp(ctx, db, redisClient)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	serverApplication := application.ServerApplication()

	// When JWT env vars were absent, load or auto-generate the signing key from DB.
	if !jwtEnvLoaded {
		if err := oauth.EnsureGlobalSigningKeyFromDB(ctx, db); err != nil {
			return fmt.Errorf("ensure signing key: %w", err)
		}
	}

	// Wire the combined JTI revocation checker:
	//   1. Redis denylist — fast path for logout and refresh-token rotation.
	//   2. DB revocation store — authoritative record for RFC 7009 /revoke calls.
	revocationRepo := application.TokenRevocationRepo
	jwt.SetJTIChecker(func(ctx context.Context, jti string) (bool, error) {
		if denied, err := application.Cache.IsJTIDenied(ctx, jti); denied || err != nil {
			return denied, err
		}
		return revocationRepo.IsRevokedByJTI(jti)
	})

	// Background workers use a child context so they are cancelled after the
	// blocking REST server returns from graceful shutdown.
	bgCtx, cancelBackground := context.WithCancel(ctx)
	defer cancelBackground()

	// Background workers start before REST begins blocking, while REST remains
	// the foreground server that owns process lifetime.
	startBackgroundWorkers(bgCtx, application, serverApplication)
	return appserver.StartRESTServer(serverApplication)
}
