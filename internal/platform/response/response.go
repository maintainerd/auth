package response

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cookie"
)

// loggerContextKey is the context key used to store a request-scoped slog.Logger.
type loggerContextKey struct{}

// WithLogger returns a copy of ctx carrying the given logger.
// Called by LoggingMiddleware to propagate a request_id-seeded logger.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext returns the request-scoped logger stored in ctx.
// Falls back to slog.Default() when no logger has been set.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    string      `json:"code,omitempty"`
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

// ErrorWithCode sends an error response carrying a stable, machine-readable
// code alongside the human-readable message, so clients can branch on the
// code (e.g. "step_up_required") without string-matching the message.
func ErrorWithCode(w http.ResponseWriter, status int, code, err string, details ...any) {
	resp := response{
		Success: false,
		Error:   err,
		Code:    code,
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

// BadRequest sends a 400 Bad Request response with the standard generic
// "Invalid request" message.
func BadRequest(w http.ResponseWriter) {
	Error(w, http.StatusBadRequest, "Invalid request")
}

// BadRequestBody sends a 400 Bad Request response with the standard
// "Invalid request body" message.
func BadRequestBody(w http.ResponseWriter) {
	Error(w, http.StatusBadRequest, "Invalid request body")
}

// writeJSON writes a JSON response with the specified status code
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// HandleServiceError inspects the typed error returned by a service method and
// writes the appropriate HTTP response. Internal/unexpected errors are logged
// server-side (with the request-scoped logger so they carry request_id) and a
// generic message is sent to the client.
func HandleServiceError(w http.ResponseWriter, r *http.Request, fallbackMsg string, err error) {
	var notFound *apperror.NotFoundError
	var conflict *apperror.ConflictError
	var forbidden *apperror.ForbiddenError
	var unauthorized *apperror.UnauthorizedError
	var validationErr *apperror.ValidationError
	var throttled *apperror.TooManyRequestsError
	var unavailable *apperror.ServiceUnavailableError
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
	case errors.As(err, &unavailable):
		// Checked before throttled: a limiter outage produces a refusal that is
		// about the service, not the caller's rate, and 429 would blame the user
		// for a fault they did not cause.
		Error(w, http.StatusServiceUnavailable, unavailable.Error())
	case errors.As(err, &throttled):
		// Retry-After is seconds, rounded UP: rounding down hands back a delay
		// that is still inside the window, so the client's first retry is
		// guaranteed to be throttled again (RFC 9110 §10.2.3).
		if throttled.RetryAfter > 0 {
			seconds := int(math.Ceil(throttled.RetryAfter.Seconds()))
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
		}
		Error(w, http.StatusTooManyRequests, throttled.Error())
	// The gorm sentinels are matched BEFORE InternalError because services wrap
	// driver errors in one (apperror.NewInternal) and InternalError unwraps —
	// so a sentinel reaches here inside an InternalError, and testing the typed
	// case first would swallow every one of them as a 500.
	case errors.Is(err, gorm.ErrDuplicatedKey):
		// A unique index rejected the write. Services pre-check uniqueness and
		// return a domain Conflict, but a pre-check cannot close the race between
		// two concurrent writers — this is the backstop so the loser gets 409
		// rather than 500.
		Error(w, http.StatusConflict, "A record with these values already exists")
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		Error(w, http.StatusBadRequest, "A referenced record does not exist")
	case errors.As(err, &internal):
		LoggerFromContext(r.Context()).Error("internal service error", "error", internal.Error())
		Error(w, http.StatusInternalServerError, fallbackMsg)
	default:
		// Untyped error — log it and return the fallback message.
		LoggerFromContext(r.Context()).Error("unhandled service error", "error", err.Error())
		Error(w, http.StatusInternalServerError, fallbackMsg)
	}
}
