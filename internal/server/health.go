package server

import "net/http"

// handleHealth responds to liveness probes. Always returns 200 OK when the
// process is running — no dependency checks.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
}

// handleReady returns an http.HandlerFunc that checks database and Redis
// connectivity. It returns 200 OK when both dependencies are reachable, or
// 503 Service Unavailable when either check fails.
func handleReady(application *Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sqlDB, err := application.DB.DB()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"database connection unavailable"}`)) //nolint:errcheck
			return
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"database ping failed"}`)) //nolint:errcheck
			return
		}

		if err := application.RedisClient.Ping(ctx).Err(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"redis ping failed"}`)) //nolint:errcheck
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`)) //nolint:errcheck
	}
}
