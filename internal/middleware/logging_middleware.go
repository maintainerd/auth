package middleware

import (
	"log/slog"
	"net/http"
	"time"

	resp "github.com/maintainerd/auth/internal/rest/response"
)

// statusRecorder wraps http.ResponseWriter to capture the HTTP status code
// written by the downstream handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Unwrap allows middleware that probe for optional interfaces (e.g. http.Flusher)
// to reach the underlying ResponseWriter.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// LoggingMiddleware emits a structured JSON access-log entry for every request.
// It must be registered after SecurityContextMiddleware so that request_id is
// already present in the context. It also attaches a request-scoped slog.Logger
// (seeded with request_id) to the context so that downstream code — including
// resp.HandleServiceError — can log correlated error entries without knowing
// about the HTTP layer.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Read the request_id injected by SecurityContextMiddleware.
		requestID, _ := r.Context().Value(RequestIDKey).(string)

		// Build a request-scoped logger pre-seeded with the request_id.
		logger := slog.Default().With("request_id", requestID)

		// Propagate the seeded logger through the request context.
		ctx := resp.WithLogger(r.Context(), logger)

		// Wrap the ResponseWriter so we can record the status code.
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r.WithContext(ctx))

		// Structured access log — emitted after the handler returns so the
		// status code and latency are both known.
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"latency_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}
