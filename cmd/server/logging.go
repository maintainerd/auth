package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/logging"
)

// initBootstrapLogger installs a minimal structured logger before configuration
// is available. This keeps config-loading failures machine-readable.
func initBootstrapLogger() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}

// initConfiguredLogger applies runtime log level and wraps output with the PII
// redaction handler used by the rest of the application.
func initConfiguredLogger() {
	slog.SetDefault(slog.New(
		logging.NewPIIRedactHandler(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: parseSlogLevel(config.LogLevel),
			}),
		),
	))
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
