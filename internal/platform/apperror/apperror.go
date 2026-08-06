// Package apperror defines structured error types for the service layer.
//
// Every error returned by a service function should be one of the typed errors
// below. The REST handler layer uses [resp.HandleServiceError] to translate
// these into the correct HTTP status codes automatically:
//
//	NotFoundError         → 404
//	ConflictError         → 409
//	ForbiddenError        → 403
//	UnauthorizedError     → 401
//	ValidationError       → 400
//	TooManyRequestsError  → 429 (+ Retry-After when the error carries one)
//	InternalError         → 500 (logged server-side; generic message sent to client)
//
// Usage in a service:
//
//	return nil, apperror.NewNotFound("tenant")          // "tenant not found"
//	return nil, apperror.NewConflict("email already registered")
//	return nil, apperror.NewInternal("hash password", err)
//
// The handler does not need to inspect the error — HandleServiceError does it:
//
//	if err != nil {
//	    resp.HandleServiceError(w, r, "Failed to create tenant", err)
//	    return
//	}
package apperror

import (
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// NotFoundError
// ---------------------------------------------------------------------------

// NotFoundError indicates that a requested resource does not exist.
//
// Use [NewNotFound] when you have an entity name (produces "<entity> not found"),
// or [NewNotFoundWithReason] for a custom message.
type NotFoundError struct {
	// Entity is the name of the resource, e.g. "tenant", "user".
	Entity string
	// Reason is an optional custom message. When set, it takes precedence over Entity.
	Reason string
}

func (e *NotFoundError) Error() string {
	if e.Reason != "" {
		return e.Reason
	}
	return e.Entity + " not found"
}

// ---------------------------------------------------------------------------
// ConflictError
// ---------------------------------------------------------------------------

// ConflictError indicates a resource already exists or a uniqueness constraint
// was violated. For example: "tenant with this name already exists".
type ConflictError struct {
	Reason string
}

func (e *ConflictError) Error() string {
	return e.Reason
}

// ---------------------------------------------------------------------------
// ForbiddenError
// ---------------------------------------------------------------------------

// ForbiddenError indicates the caller is authenticated but does not have
// permission to perform the requested operation.
type ForbiddenError struct {
	Reason string
}

func (e *ForbiddenError) Error() string {
	return e.Reason
}

// ---------------------------------------------------------------------------
// UnauthorizedError
// ---------------------------------------------------------------------------

// UnauthorizedError indicates invalid or missing authentication credentials.
// When Reason is empty the default message "authentication failed" is used.
type UnauthorizedError struct {
	Reason string
}

func (e *UnauthorizedError) Error() string {
	if e.Reason != "" {
		return e.Reason
	}
	return "authentication failed"
}

// ---------------------------------------------------------------------------
// ValidationError
// ---------------------------------------------------------------------------

// ValidationError indicates a business-rule validation failure that is NOT an
// input-format problem (those are caught earlier by DTO validation in the handler).
// Examples: "cannot delete system policy", "registration flow must have at least one role".
type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string {
	return e.Reason
}

// ---------------------------------------------------------------------------
// TooManyRequestsError
// ---------------------------------------------------------------------------

// TooManyRequestsError indicates the caller exceeded a rate limit, throttle or
// quota and should back off.
//
// It is deliberately distinct from [ForbiddenError]: 403 tells a client the
// request will never succeed, so well-behaved HTTP clients and SDK retry
// policies stop. A throttled request WILL succeed later, and only 429
// (RFC 6585 §4) communicates that — returning 403 from a rate limiter both
// mislabels the failure and defeats client-side retry.
//
// When Reason is empty the default message "too many requests" is used.
type TooManyRequestsError struct {
	Reason string
	// RetryAfter, when > 0, is how long the caller should wait before retrying.
	// The REST layer emits it as the Retry-After header (RFC 9110 §10.2.3) and
	// the gRPC layer as an errdetails.RetryInfo. Zero means "unspecified" —
	// omit the hint rather than guess one.
	RetryAfter time.Duration
}

func (e *TooManyRequestsError) Error() string {
	if e.Reason != "" {
		return e.Reason
	}
	return "too many requests"
}

// ---------------------------------------------------------------------------
// InternalError
// ---------------------------------------------------------------------------

// InternalError wraps an unexpected internal failure (e.g. database or
// third-party call) with a human-readable reason and the original error.
// The original error is available via [errors.Unwrap] for logging.
type InternalError struct {
	Reason string
	Err    error
}

func (e *InternalError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Reason, e.Err)
	}
	return e.Reason
}

// Unwrap returns the underlying error so callers can use [errors.Is] / [errors.As].
func (e *InternalError) Unwrap() error {
	return e.Err
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// NewNotFound creates a [NotFoundError] for the given entity name.
// The error message will be "<entity> not found".
//
//	apperror.NewNotFound("tenant") // → "tenant not found"
func NewNotFound(entity string) *NotFoundError {
	return &NotFoundError{Entity: entity}
}

// NewNotFoundWithReason creates a [NotFoundError] with a fully custom message,
// useful when the default "<entity> not found" pattern doesn't fit.
//
//	apperror.NewNotFoundWithReason("no admin user found")
func NewNotFoundWithReason(reason string) *NotFoundError {
	return &NotFoundError{Reason: reason}
}

// NewConflict creates a [ConflictError] with the given reason.
//
//	apperror.NewConflict("email already registered")
func NewConflict(reason string) *ConflictError {
	return &ConflictError{Reason: reason}
}

// NewForbidden creates a [ForbiddenError] with the given reason.
//
//	apperror.NewForbidden("profile does not belong to user")
func NewForbidden(reason string) *ForbiddenError {
	return &ForbiddenError{Reason: reason}
}

// NewUnauthorized creates an [UnauthorizedError] with the given reason.
// Pass an empty string to use the default "authentication failed" message.
//
//	apperror.NewUnauthorized("invalid credentials")
func NewUnauthorized(reason string) *UnauthorizedError {
	return &UnauthorizedError{Reason: reason}
}

// NewValidation creates a [ValidationError] with the given reason.
//
//	apperror.NewValidation("cannot delete system policy")
func NewValidation(reason string) *ValidationError {
	return &ValidationError{Reason: reason}
}

// NewTooManyRequests creates a [TooManyRequestsError] with the given reason and
// no retry hint. Pass an empty string to use the default "too many requests".
//
//	apperror.NewTooManyRequests("too many verification attempts")
func NewTooManyRequests(reason string) *TooManyRequestsError {
	return &TooManyRequestsError{Reason: reason}
}

// NewTooManyRequestsAfter creates a [TooManyRequestsError] that also tells the
// caller when to retry. Use it wherever the limiter already knows the window,
// so the client backs off for exactly that long instead of guessing.
//
//	apperror.NewTooManyRequestsAfter("too many login attempts", 15*time.Minute)
func NewTooManyRequestsAfter(reason string, retryAfter time.Duration) *TooManyRequestsError {
	return &TooManyRequestsError{Reason: reason, RetryAfter: retryAfter}
}

// NewInternal creates an [InternalError] that wraps an underlying error with context.
// The underlying error is preserved for [errors.Unwrap] and server-side logging.
//
//	apperror.NewInternal("hash password", err)
func NewInternal(reason string, err error) *InternalError {
	return &InternalError{Reason: reason, Err: err}
}

// ---------------------------------------------------------------------------
// ServiceUnavailableError
// ---------------------------------------------------------------------------

// ServiceUnavailableError indicates the request could not be served because a
// dependency this operation requires is unavailable — not because anything is
// wrong with the request.
//
// It is deliberately distinct from [InternalError]: 500 says the server hit a
// fault and the client can do nothing useful, while 503 says the condition is
// transient and retrying later is the correct behaviour. It is equally distinct
// from [TooManyRequestsError]: 429 blames the caller's rate, and telling a user
// they made too many attempts when the real problem is that the rate-limit store
// is down is a lie that sends them to support instead of back in a minute.
//
// The motivating case is a credential path failing closed: when the lockout
// store cannot be read, login refuses rather than let attempts through
// unmetered, and that refusal is a 503.
type ServiceUnavailableError struct {
	Reason string
}

func (e *ServiceUnavailableError) Error() string {
	if e.Reason == "" {
		return "service temporarily unavailable"
	}
	return e.Reason
}

// NewServiceUnavailable builds a 503 for a transient dependency failure.
func NewServiceUnavailable(reason string) *ServiceUnavailableError {
	return &ServiceUnavailableError{Reason: reason}
}
