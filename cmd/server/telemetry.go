package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/platform/telemetry"
)

// initTelemetry starts tracing and metrics together and returns one shutdown
// function so run can defer a single cleanup hook.
func initTelemetry(ctx context.Context) (func(), error) {
	tracingShutdown, err := telemetry.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize OpenTelemetry tracing: %w", err)
	}

	metricsShutdown, err := telemetry.InitMetrics(ctx)
	if err != nil {
		shutdownWithTimeout("OpenTelemetry tracing", tracingShutdown)
		return nil, fmt.Errorf("initialize OpenTelemetry metrics: %w", err)
	}

	return func() {
		// Metrics flush first so request counters/histograms are exported before
		// tracing providers are torn down.
		shutdownWithTimeout("OpenTelemetry metrics", metricsShutdown)
		shutdownWithTimeout("OpenTelemetry tracing", tracingShutdown)
	}, nil
}

// shutdownWithTimeout protects process shutdown from hanging forever if an
// exporter is unavailable or slow during final flush.
func shutdownWithTimeout(name string, shutdown func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := shutdown(ctx); err != nil {
		slog.Error(name+" shutdown error", "error", err)
	}
}
