package util

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func Success(w http.ResponseWriter, data interface{}, message string) {
	writeJSON(w, http.StatusOK, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func Created(w http.ResponseWriter, data interface{}, message string) {
	writeJSON(w, http.StatusCreated, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func Error(w http.ResponseWriter, status int, err string, details ...any) {
	resp := response{
		Success: false,
		Error:   err,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	writeJSON(w, status, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
