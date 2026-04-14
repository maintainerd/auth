package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/config"
	grpcserver "github.com/maintainerd/auth/internal/grpc/server"
	"github.com/maintainerd/auth/internal/jwt"
	restserver "github.com/maintainerd/auth/internal/rest/server"
	"github.com/maintainerd/auth/internal/runner"
	"github.com/maintainerd/auth/internal/security"
	"github.com/maintainerd/auth/internal/telemetry"
)

func main() {
	// Configure structured JSON logging for container environments
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// ⚙️ Load configurations
	if err := config.Init(); err != nil {
		slog.Error("Configuration loading failed", "error", err)
		os.Exit(1)
	}

	// ⚙️ Initialise OpenTelemetry tracing (safe no-op when OTEL_ENABLED != true)
	otelShutdown, err := telemetry.Init(context.Background())
	if err != nil {
		slog.Error("OpenTelemetry initialization failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := otelShutdown(ctx); err != nil {
			slog.Error("OpenTelemetry shutdown error", "error", err)
		}
	}()

	// ⚙️ Parse RSA keys (required for token signing)
	if err := jwt.InitJWTKeys(); err != nil {
		slog.Error("Failed to initialize JWT keys", "error", err)
		os.Exit(1)
	}

	// ⚙️ Load database
	db, err := config.InitDB()
	if err != nil {
		slog.Error("Database initialization failed", "error", err)
		os.Exit(1)
	}

	// ⚙️ Load Redis
	redisClient, err := config.NewRedisClient()
	if err != nil {
		slog.Error("Redis initialization failed", "error", err)
		os.Exit(1)
	}

	// ⚙️ Wire Redis-backed rate limiter
	security.InitRateLimiter(redisClient)

	// ⚙️ Run database migrations
	if err := runner.RunMigrations(db); err != nil {
		slog.Error("Database migrations failed", "error", err)
		os.Exit(1)
	}

	// ⚙️ App wiring (handlers, services, etc.)
	application := app.NewApp(db, redisClient)

	// Create a cancellable context for background workers.
	// It is cancelled after the REST servers have drained so that background
	// goroutines also shut down gracefully when an OS signal is received.
	bgCtx, cancelBG := context.WithCancel(context.Background())

	// 🗑️ Auth event retention runner (background)
	go runner.StartRetentionRunner(bgCtx, application.AuthEventService, runner.DefaultRetentionPeriod, runner.DefaultRetentionInterval)

	// 🚀 gRPC server (background) — errors are logged; they don't affect REST.
	go func() {
		if err := grpcserver.StartGRPCServer(bgCtx, application); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// 🚀 REST servers — blocks until OS signal then drains.
	restserver.StartRESTServer(application)

	// Cancel the background context after REST has drained so gRPC and
	// retention runner also shut down.
	cancelBG()
}
