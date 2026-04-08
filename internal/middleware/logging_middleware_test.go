package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	resp "github.com/maintainerd/auth/internal/rest/response"
)

// withRequestID returns a request whose context already carries a request_id,
// as SecurityContextMiddleware would set it.
func withRequestID(r *http.Request, id string) *http.Request {
	ctx := context.WithValue(r.Context(), RequestIDKey, id)
	return r.WithContext(ctx)
}

func TestLoggingMiddleware_EmitsAccessLog(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil))) })

	req := withRequestID(httptest.NewRequest(http.MethodGet, "/api/test", nil), "req-abc")
	rr := httptest.NewRecorder()

	LoggingMiddleware(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	out := buf.String()
	assert.Contains(t, out, `"request_id":"req-abc"`)
	assert.Contains(t, out, `"method":"GET"`)
	assert.Contains(t, out, `"path":"/api/test"`)
	assert.Contains(t, out, `"status":200`)
	assert.Contains(t, out, `"latency_ms"`)
}

func TestLoggingMiddleware_RecordsNon200Status(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil))) })

	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := withRequestID(httptest.NewRequest(http.MethodDelete, "/resource/1", nil), "req-404")
	rr := httptest.NewRecorder()

	LoggingMiddleware(notFoundHandler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, buf.String(), `"status":404`)
}

func TestLoggingMiddleware_AttachesLoggerToContext(t *testing.T) {
	var capturedLogger *slog.Logger

	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedLogger = resp.LoggerFromContext(r.Context())
	})

	req := withRequestID(httptest.NewRequest(http.MethodPost, "/login", nil), "req-ctx")
	rr := httptest.NewRecorder()

	LoggingMiddleware(next).ServeHTTP(rr, req)

	require.NotNil(t, capturedLogger)
	// The attached logger must not be the default logger — it is a seeded copy.
	assert.NotSame(t, slog.Default(), capturedLogger)
}

func TestLoggingMiddleware_NoRequestID_DoesNotPanic(t *testing.T) {
	// If SecurityContextMiddleware hasn't run yet there is no request_id.
	// LoggingMiddleware must still work gracefully with an empty string.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		LoggingMiddleware(okHandler()).ServeHTTP(rr, req)
	})
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestStatusRecorder_DefaultsTo200(t *testing.T) {
	// When the handler never calls WriteHeader, the recorder should report 200.
	silent := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// write body without calling WriteHeader
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	LoggingMiddleware(silent).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestStatusRecorder_Unwrap(t *testing.T) {
	inner := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: inner, status: http.StatusOK}
	assert.Equal(t, inner, sr.Unwrap())
}
