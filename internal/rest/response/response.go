package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/cookie"
)

type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// Success sends a successful response with HTTP 200 status
func Success(w http.ResponseWriter, data interface{}, message string) {
	writeJSON(w, http.StatusOK, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// SuccessWithCookies sends a successful response with optional cookie delivery
func SuccessWithCookies(w http.ResponseWriter, r *http.Request, data interface{}, message string) {
	// Check if cookies should be set based on X-Token-Delivery header
	if r.Header.Get("X-Token-Delivery") == "cookie" {
		cookie.SetAuthCookies(w, data)
	}

	writeJSON(w, http.StatusOK, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Created sends a successful response with HTTP 201 status
func Created(w http.ResponseWriter, data interface{}, message string) {
	writeJSON(w, http.StatusCreated, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// CreatedWithCookies sends a created response with optional cookie delivery
func CreatedWithCookies(w http.ResponseWriter, r *http.Request, data interface{}, message string) {
	// Check if cookies should be set based on X-Token-Delivery header
	if r.Header.Get("X-Token-Delivery") == "cookie" {
		cookie.SetAuthCookies(w, data)
	}

	writeJSON(w, http.StatusCreated, response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Error sends an error response with the specified status code
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

// ValidationError sends a validation error response with HTTP 400 status
func ValidationError(w http.ResponseWriter, err error) {
	if ve, ok := err.(validation.Errors); ok {
		Error(w, http.StatusBadRequest, "Validation failed", ve)
		return
	}
	Error(w, http.StatusBadRequest, "Validation failed", err.Error())
}

// writeJSON writes a JSON response with the specified status code
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// HandleServiceError inspects the typed error returned by a service method and
// writes the appropriate HTTP response. Internal/unexpected errors are logged
// server-side and a generic message is sent to the client.
func HandleServiceError(w http.ResponseWriter, fallbackMsg string, err error) {
	var notFound *apperror.NotFoundError
	var conflict *apperror.ConflictError
	var forbidden *apperror.ForbiddenError
	var unauthorized *apperror.UnauthorizedError
	var validationErr *apperror.ValidationError
	var internal *apperror.InternalError

	switch {
	case errors.As(err, &notFound):
		Error(w, http.StatusNotFound, notFound.Error())
	case errors.As(err, &conflict):
		Error(w, http.StatusConflict, conflict.Error())
	case errors.As(err, &forbidden):
		Error(w, http.StatusForbidden, forbidden.Error())
	case errors.As(err, &unauthorized):
		Error(w, http.StatusUnauthorized, unauthorized.Error())
	case errors.As(err, &validationErr):
		Error(w, http.StatusBadRequest, validationErr.Error())
	case errors.As(err, &internal):
		slog.Error("internal service error", "error", internal.Error())
		Error(w, http.StatusInternalServerError, fallbackMsg)
	default:
		// Untyped error — log it and return the fallback message.
		slog.Error("unhandled service error", "error", err.Error())
		Error(w, http.StatusInternalServerError, fallbackMsg)
	}
}
