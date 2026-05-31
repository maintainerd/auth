package main

import (
	"context"
	"fmt"

	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/runner"
	"github.com/maintainerd/auth/internal/platform/security"
	appserver "github.com/maintainerd/auth/internal/server"
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
	// Rebuild the logger once LOG_LEVEL and PII redaction settings are known.
	initConfiguredLogger()

	// Start tracing and metrics before downstream dependencies are initialized
	// so startup failures are observable too.
	telemetryShutdown, err := initTelemetry(ctx)
	if err != nil {
		return err
	}
	defer telemetryShutdown()

	// JWT keys are required before the app accepts traffic because token
	// signing and validation depend on the configured RSA key pair.
	if err := jwt.InitJWTKeys(); err != nil {
		return fmt.Errorf("initialize JWT keys: %w", err)
	}

	// Database and Redis are process-level dependencies shared by repositories,
	// cache, rate limiting, sessions, and background workers.
	db, err := config.InitDB()
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	redisClient, err := config.NewRedisClient()
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
	application, err := app.NewApp(db, redisClient)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	serverApplication := application.ServerApplication()

	// Wire the JTI denylist checker so ValidateToken rejects revoked access tokens.
	jwt.JTIChecker = application.Cache.IsJTIDenied

	// Background workers use a child context so they are cancelled after the
	// blocking REST server returns from graceful shutdown.
	bgCtx, cancelBackground := context.WithCancel(ctx)
	defer cancelBackground()

	// Background workers start before REST begins blocking, while REST remains
	// the foreground server that owns process lifetime.
	startBackgroundWorkers(bgCtx, application, serverApplication)
	return appserver.StartRESTServer(serverApplication)
}
