package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/maintainerd/auth/internal/config"
)

const (
	defaultServiceName = "maintainerd-auth"
	shutdownTimeout    = 5 * time.Second
)

// Init bootstraps the OpenTelemetry TracerProvider.
//
// When OTEL_ENABLED is "true" it connects an OTLP/gRPC exporter to the
// collector at OTEL_EXPORTER_OTLP_ENDPOINT (default: localhost:4317) and
// registers a BatchSpanProcessor. All standard OTEL_* env vars (endpoint,
// headers, TLS, etc.) are respected automatically by the SDK.
//
// When OTEL_ENABLED is missing or any other value, a no-op TracerProvider is
// installed so the rest of the code can call otel.Tracer() safely without
// branching.
//
// It returns a shutdown function that must be called before the process exits
// (e.g. deferred in main) to flush buffered spans.
func Init(ctx context.Context) (shutdown func(context.Context) error, err error) {
	serviceName := config.GetEnvOrDefault("OTEL_SERVICE_NAME", defaultServiceName)
	appVersion := config.AppVersion

	if config.GetEnvOrDefault("OTEL_ENABLED", "false") != "true" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		slog.Info("OpenTelemetry tracing disabled (OTEL_ENABLED != true)")
		return noopShutdown, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(appVersion),
		),
	)
	if err != nil {
		return noopShutdown, fmt.Errorf("telemetry: build resource: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return noopShutdown, fmt.Errorf("telemetry: create OTLP exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("OpenTelemetry tracing enabled",
		"service", serviceName,
		"version", appVersion,
	)

	return tp.Shutdown, nil
}

// TraceIDFromContext extracts the W3C trace ID and span ID from the current
// span in ctx. Returns empty strings when there is no active span.
func TraceIDFromContext(ctx context.Context) (traceID, spanID string) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasTraceID() {
		traceID = sc.TraceID().String()
	}
	if sc.HasSpanID() {
		spanID = sc.SpanID().String()
	}
	return
}

func noopShutdown(context.Context) error { return nil }
