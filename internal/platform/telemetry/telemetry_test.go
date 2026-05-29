package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestInit_Disabled(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "false")

	shutdown, err := Init(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	tp := otel.GetTracerProvider()
	assert.IsType(t, noop.NewTracerProvider(), tp)

	assert.NoError(t, shutdown(context.Background()))
}

func TestInit_Enabled(t *testing.T) {
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:0")
	t.Setenv("OTEL_SERVICE_NAME", "test-service")

	shutdown, err := Init(context.Background())
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	tp := otel.GetTracerProvider()
	assert.IsType(t, &sdktrace.TracerProvider{}, tp)

	assert.NoError(t, shutdown(context.Background()))
}

func TestTraceIDFromContext_WithActiveSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-op")
	defer span.End()

	traceID, spanID := TraceIDFromContext(ctx)
	assert.NotEmpty(t, traceID)
	assert.NotEmpty(t, spanID)
	assert.Len(t, traceID, 32)
	assert.Len(t, spanID, 16)
}

func TestTraceIDFromContext_WithoutSpan(t *testing.T) {
	traceID, spanID := TraceIDFromContext(context.Background())
	assert.Empty(t, traceID)
	assert.Empty(t, spanID)
}

func TestTraceIDFromContext_NoopSpan(t *testing.T) {
	ctx := trace.ContextWithSpan(context.Background(), noop.Span{})
	traceID, spanID := TraceIDFromContext(ctx)
	assert.Empty(t, traceID)
	assert.Empty(t, spanID)
}

func TestNoopShutdown(t *testing.T) {
	assert.NoError(t, noopShutdown(context.Background()))
}
