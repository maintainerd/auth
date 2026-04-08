// Package apperror defines structured error types for the service layer.
//
// Every error returned by a service function should be one of the typed errors
// below. The REST handler layer uses [resp.HandleServiceError] to translate
// these into the correct HTTP status codes automatically:
//
//	NotFoundError     → 404
//	ConflictError     → 409
//	ForbiddenError    → 403
//	UnauthorizedError → 401
//	ValidationError   → 400
//	InternalError     → 500 (logged server-side; generic message sent to client)
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

import "fmt"

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
// Examples: "cannot delete system policy", "signup flow must have at least one role".
type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string {
	return e.Reason
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

// NewInternal creates an [InternalError] that wraps an underlying error with context.
// The underlying error is preserved for [errors.Unwrap] and server-side logging.
//
//	apperror.NewInternal("hash password", err)
func NewInternal(reason string, err error) *InternalError {
	return &InternalError{Reason: reason, Err: err}
}
