package apperror

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNotFoundError(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		err := NewNotFound("policy")
		assert.Equal(t, "policy not found", err.Error())

		var target *NotFoundError
		assert.True(t, errors.As(err, &target))
	})

	t.Run("custom reason", func(t *testing.T) {
		err := NewNotFoundWithReason("SMS template not found or access denied")
		assert.Equal(t, "SMS template not found or access denied", err.Error())
	})
}

func TestConflictError(t *testing.T) {
	err := NewConflict("username already taken")
	assert.Equal(t, "username already taken", err.Error())

	var target *ConflictError
	assert.True(t, errors.As(err, &target))
}

func TestForbiddenError(t *testing.T) {
	err := NewForbidden("access denied: user does not have access to this tenant")
	assert.Equal(t, "access denied: user does not have access to this tenant", err.Error())

	var target *ForbiddenError
	assert.True(t, errors.As(err, &target))
}

func TestUnauthorizedError(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		err := NewUnauthorized("")
		assert.Equal(t, "authentication failed", err.Error())
	})

	t.Run("custom reason", func(t *testing.T) {
		err := NewUnauthorized("invalid credentials")
		assert.Equal(t, "invalid credentials", err.Error())
	})

	var target *UnauthorizedError
	assert.True(t, errors.As(NewUnauthorized("test"), &target))
}

func TestValidationError(t *testing.T) {
	err := NewValidation("cannot delete system SMS template")
	assert.Equal(t, "cannot delete system SMS template", err.Error())

	var target *ValidationError
	assert.True(t, errors.As(err, &target))
}

func TestTooManyRequestsError(t *testing.T) {
	t.Run("default message", func(t *testing.T) {
		err := NewTooManyRequests("")
		assert.Equal(t, "too many requests", err.Error())
		assert.Zero(t, err.RetryAfter)
	})

	t.Run("custom reason", func(t *testing.T) {
		err := NewTooManyRequests("too many verification attempts")
		assert.Equal(t, "too many verification attempts", err.Error())
	})

	t.Run("carries a retry hint", func(t *testing.T) {
		err := NewTooManyRequestsAfter("too many login attempts", 15*time.Minute)
		assert.Equal(t, "too many login attempts", err.Error())
		assert.Equal(t, 15*time.Minute, err.RetryAfter)
	})

	var target *TooManyRequestsError
	assert.True(t, errors.As(NewTooManyRequests("slow down"), &target))

	// A throttle must never be mistaken for a permanent denial.
	var forbidden *ForbiddenError
	assert.False(t, errors.As(NewTooManyRequests("slow down"), &forbidden))
}

func TestInternalError(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		inner := fmt.Errorf("connection refused")
		err := NewInternal("failed to send invite email", inner)
		assert.Equal(t, "failed to send invite email: connection refused", err.Error())
		assert.True(t, errors.Is(err, inner))
	})

	t.Run("without wrapped error", func(t *testing.T) {
		err := NewInternal("not implemented", nil)
		assert.Equal(t, "not implemented", err.Error())
	})

	var target *InternalError
	assert.True(t, errors.As(NewInternal("x", nil), &target))
}
