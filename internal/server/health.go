package server

import (
	"encoding/json"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
)

type healthResponse struct {
	Status     string            `json:"status"`
	Version    string            `json:"version,omitempty"`
	Dependency *dependencyStatus `json:"dependency,omitempty"`
	Reason     string            `json:"reason,omitempty"`
}

type dependencyStatus struct {
	Database string `json:"database"`
	Redis    string `json:"redis"`
	JWKS     string `json:"jwks"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func handleLivez(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: config.AppVersion,
	})
}

func handleReady(application *Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deps := &dependencyStatus{Database: "ok", Redis: "ok", JWKS: "ok"}
		allOk := true

		sqlDB, err := application.DB.DB()
		if err != nil {
			deps.Database = "unavailable"
			allOk = false
		} else if err := sqlDB.PingContext(ctx); err != nil {
			deps.Database = "unreachable"
			allOk = false
		}

		if application.RedisClient != nil {
			if err := application.RedisClient.Ping(ctx).Err(); err != nil {
				deps.Redis = "unreachable"
				allOk = false
			}
		} else {
			deps.Redis = "not configured"
		}

		if jwt.GetPublicKey() == nil {
			deps.JWKS = "not loaded"
			allOk = false
		}

		status := "ready"
		httpStatus := http.StatusOK
		if !allOk {
			status = "not ready"
			httpStatus = http.StatusServiceUnavailable
		}

		writeJSON(w, httpStatus, healthResponse{
			Status:     status,
			Version:    config.AppVersion,
			Dependency: deps,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
