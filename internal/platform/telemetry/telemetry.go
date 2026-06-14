package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	logglobal "go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/maintainerd/auth/internal/platform/config"
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

	if !Enabled() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		slog.Info("OpenTelemetry tracing disabled (OTEL_ENABLED != true)")
		return noopShutdown, nil
	}

	res, err := buildResource(serviceName, appVersion)
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

// Enabled reports whether OpenTelemetry export is turned on (OTEL_ENABLED=true).
// All three signal initializers (traces, metrics, logs) and the logging setup
// branch on this so behaviour stays consistent.
func Enabled() bool {
	on, _ := strconv.ParseBool(config.GetEnvOrDefault("OTEL_ENABLED", "false"))
	return on
}

// buildResource builds the OTel resource (service.name/version) shared by all
// signal providers.
func buildResource(serviceName, appVersion string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(appVersion),
		),
	)
}

// InitLogs bootstraps the OpenTelemetry LoggerProvider.
//
// When OTEL_ENABLED is "true" it connects an OTLP/gRPC log exporter to the same
// collector as traces (OTEL_EXPORTER_OTLP_ENDPOINT) and installs the provider
// as the global LoggerProvider, so the slog→OTel bridge in cmd/server ships
// application logs over OTLP. Otherwise a no-op provider is installed.
//
// This keeps logging vendor-neutral: the app only ever speaks OTLP, configured
// purely by the standard OTEL_* env vars.
func InitLogs(ctx context.Context) (shutdown func(context.Context) error, err error) {
	serviceName := config.GetEnvOrDefault("OTEL_SERVICE_NAME", defaultServiceName)
	appVersion := config.AppVersion

	if !Enabled() {
		logglobal.SetLoggerProvider(lognoop.NewLoggerProvider())
		slog.Info("OpenTelemetry logging disabled (OTEL_ENABLED != true)")
		return noopShutdown, nil
	}

	res, err := buildResource(serviceName, appVersion)
	if err != nil {
		return noopShutdown, fmt.Errorf("telemetry: build resource for logs: %w", err)
	}

	exporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return noopShutdown, fmt.Errorf("telemetry: create OTLP log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	logglobal.SetLoggerProvider(lp)

	slog.Info("OpenTelemetry logging enabled (OTLP exporter)",
		"service", serviceName,
		"version", appVersion,
	)

	return lp.Shutdown, nil
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

// InitMetrics bootstraps the OpenTelemetry MeterProvider with a Prometheus
// exporter. All HTTP metrics emitted by otelhttp and any instrumented code
// are exported via the default Prometheus registry, which is served at
// /metrics on the internal port.
//
// It also registers a build_info observable gauge so dashboards can correlate
// metrics with the deployed version.
//
// Returns a shutdown function that flushes pending metric data.
func InitMetrics(ctx context.Context) (shutdown func(context.Context) error, err error) {
	serviceName := config.GetEnvOrDefault("OTEL_SERVICE_NAME", defaultServiceName)
	appVersion := config.AppVersion

	res, err := buildResource(serviceName, appVersion)
	if err != nil {
		return noopShutdown, fmt.Errorf("telemetry: build resource for metrics: %w", err)
	}

	exporter, err := promexporter.New()
	if err != nil {
		return noopShutdown, fmt.Errorf("telemetry: create prometheus exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Register a build_info gauge so dashboards can pin metrics to a version.
	meter := mp.Meter("maintainerd-auth")
	buildInfo, err := meter.Int64ObservableGauge(
		"build_info",
		metric.WithDescription("Build information about the running service"),
	)
	if err != nil {
		return mp.Shutdown, fmt.Errorf("telemetry: register build_info gauge: %w", err)
	}

	buildCommit, buildDate := readBuildInfo()

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		attrs := []attribute.KeyValue{
			attribute.String("version", appVersion),
			attribute.String("service", serviceName),
		}
		if buildCommit != "" {
			attrs = append(attrs, attribute.String("commit", buildCommit))
		}
		if buildDate != "" {
			attrs = append(attrs, attribute.String("build_date", buildDate))
		}
		o.ObserveInt64(buildInfo, 1, metric.WithAttributes(attrs...))
		return nil
	}, buildInfo)
	if err != nil {
		return mp.Shutdown, fmt.Errorf("telemetry: register build_info callback: %w", err)
	}

	slog.Info("OpenTelemetry metrics enabled (Prometheus exporter)",
		"service", serviceName,
		"version", appVersion,
	)

	return mp.Shutdown, nil
}

func noopShutdown(context.Context) error { return nil }

func readBuildInfo() (commit, date string) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.time":
			date = s.Value
		}
	}
	return commit, date
}
