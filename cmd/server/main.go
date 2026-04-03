package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/config"
	grpcserver "github.com/maintainerd/auth/internal/grpc"
	restserver "github.com/maintainerd/auth/internal/rest"
	"github.com/maintainerd/auth/internal/runner"
	"github.com/maintainerd/auth/internal/util"
)

func main() {
	// Configure structured JSON logging for container environments
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// ⚙️ Load configurations
	if err := config.Init(); err != nil {
		slog.Error("Configuration loading failed", "error", err)
		os.Exit(1)
	}

	// ⚙️ Parse RSA keys (required for token signing)
	if err := util.InitJWTKeys(); err != nil {
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
	util.InitRateLimiter(redisClient)

	// ⚙️ Run database migrations
	if err := runner.RunMigrations(db); err != nil {
		slog.Error("Database migrations failed", "error", err)
		os.Exit(1)
	}

	// ⚙️ App wiring (handlers, services, etc.)
	application := app.NewApp(db, redisClient)

	// Create a cancellable context for the gRPC server.
	// It is cancelled after the REST servers have drained so that gRPC also
	// shuts down gracefully when an OS signal is received.
	grpcCtx, cancelGRPC := context.WithCancel(context.Background())

	// 🚀 gRPC server (background) — errors are logged; they don't affect REST.
	go func() {
		if err := grpcserver.StartGRPCServer(grpcCtx, application); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// 🚀 REST servers — blocks until OS signal then drains.
	restserver.StartRESTServer(application)

	// Cancel the gRPC context after REST has drained so gRPC also shuts down.
	cancelGRPC()
}
