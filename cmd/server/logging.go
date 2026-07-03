package main

import (
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/logging"
	"github.com/maintainerd/maintainerd-auth/internal/platform/telemetry"
)

// initBootstrapLogger installs a minimal structured logger before configuration
// is available. This keeps config-loading failures machine-readable.
func initBootstrapLogger() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}

// initConfiguredLogger applies runtime log level and wraps output with the PII
// redaction handler used by the rest of the application.
//
// When OpenTelemetry is enabled it additionally bridges slog to the OTel
// LoggerProvider (installed by telemetry.InitLogs) so logs are exported over
// OTLP alongside stdout — keeping logging vendor-neutral. PII redaction sits at
// the top, so both sinks receive already-sanitised records.
func initConfiguredLogger() {
	level := parseSlogLevel(config.LogLevel)
	stdout := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})

	var sink slog.Handler = stdout
	if telemetry.Enabled() {
		serviceName := config.GetEnvOrDefault("OTEL_SERVICE_NAME", "maintainerd-auth")
		otelHandler := otelslog.NewHandler(serviceName)
		sink = logging.NewFanoutHandler(level, stdout, otelHandler)
	}

	slog.SetDefault(slog.New(logging.NewPIIRedactHandler(sink)))
}

// parseSlogLevel maps LOG_LEVEL to slog levels. Unknown values fall back to
// info so a typo does not make the server unexpectedly silent.
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
