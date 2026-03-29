package main

import (
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
	config.Init()

	// ⚙️ Parse RSA keys (required for token signing)
	if err := util.InitJWTKeys(); err != nil {
		slog.Error("Failed to initialize JWT keys", "error", err)
		os.Exit(1)
	}

	// ⚙️ Load database
	db := config.InitDB()

	// ⚙️ Load Redis
	redisClient := config.NewRedisClient()

	// ⚙️ Wire Redis-backed rate limiter
	util.InitRateLimiter(redisClient)

	// ⚙️ Run database migrations
	if err := runner.RunMigrations(db); err != nil {
		slog.Error("Database migrations failed", "error", err)
		os.Exit(1)
	}

	// ⚙️ App wiring (handlers, services, etc.)
	application := app.NewApp(db, redisClient)

	// 🚀 gRPC server (background)
	go grpcserver.StartGRPCServer(application)

	// 🚀 REST servers with graceful shutdown
	restserver.StartRESTServer(application)
}
