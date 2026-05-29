package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/logging"
	"github.com/maintainerd/auth/internal/platform/runner"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/telemetry"
	appserver "github.com/maintainerd/auth/internal/server"
)

func main() {
	// Bootstrap structured JSON logging with a temporary INFO level until the
	// config is loaded — then reinitialise with the configured level and PII
	// redaction wrapper.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// ⚙️ Load configurations
	if err := config.Init(); err != nil {
		slog.Error("Configuration loading failed", "error", err)
		os.Exit(1)
	}

	// Reinitialise the logger now that config.LogLevel is available.
	slog.SetDefault(slog.New(
		logging.NewPIIRedactHandler(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: parseSlogLevel(config.LogLevel),
			}),
		),
	))

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

	// ⚙️ Initialise OpenTelemetry metrics (Prometheus exporter always active)
	metricsShutdown, err := telemetry.InitMetrics(context.Background())
	if err != nil {
		slog.Error("OpenTelemetry metrics initialization failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsShutdown(ctx); err != nil {
			slog.Error("OpenTelemetry metrics shutdown error", "error", err)
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
	serverApplication := application.ServerApplication()

	// Wire the JTI denylist checker so ValidateToken rejects revoked access tokens.
	jwt.JTIChecker = application.Cache.IsJTIDenied

	// Create a cancellable context for background workers.
	bgCtx, cancelBG := context.WithCancel(context.Background())

	// 🗑️ Auth event retention runner (background)
	go authevent.StartRetentionRunner(bgCtx, application.AuthEventService, authevent.DefaultRetentionPeriod, authevent.DefaultRetentionInterval)

	// 🚀 gRPC server (background) — errors are logged; they don't affect REST.
	go func() {
		if err := appserver.StartGRPCServer(bgCtx, serverApplication); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	// 🚀 REST servers — blocks until OS signal then drains.
	appserver.StartRESTServer(serverApplication)

	cancelBG()
}

// parseSlogLevel maps a LOG_LEVEL string to the corresponding slog.Level.
func parseSlogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
