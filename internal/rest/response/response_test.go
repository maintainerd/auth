package response

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/apperror"
)

type responseBody struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
	Details json.RawMessage `json:"details,omitempty"`
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) responseBody {
	t.Helper()
	var body responseBody
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	return body
}

func TestSuccess(t *testing.T) {
	rr := httptest.NewRecorder()
	Success(rr, map[string]string{"token": "abc"}, "ok")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	body := decodeBody(t, rr)
	assert.True(t, body.Success)
	assert.Equal(t, "ok", body.Message)
	assert.NotEmpty(t, body.Data)
}

func TestCreated(t *testing.T) {
	rr := httptest.NewRecorder()
	Created(rr, map[string]string{"id": "1"}, "created")

	assert.Equal(t, http.StatusCreated, rr.Code)
	body := decodeBody(t, rr)
	assert.True(t, body.Success)
	assert.Equal(t, "created", body.Message)
}

func TestError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		errMsg     string
		details    []any
		wantStatus int
	}{
		{"bad request no details", http.StatusBadRequest, "bad input", nil, http.StatusBadRequest},
		{"unauthorized", http.StatusUnauthorized, "unauthorized", nil, http.StatusUnauthorized},
		{"with details", http.StatusBadRequest, "validation failed", []any{"field required"}, http.StatusBadRequest},
		{"internal server error", http.StatusInternalServerError, "internal error", nil, http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			Error(rr, tc.status, tc.errMsg, tc.details...)

			assert.Equal(t, tc.wantStatus, rr.Code)
			body := decodeBody(t, rr)
			assert.False(t, body.Success)
			assert.Equal(t, tc.errMsg, body.Error)
			if len(tc.details) > 0 {
				assert.NotEmpty(t, body.Details)
			}
		})
	}
}

func TestValidationError_WithOzzoErrors(t *testing.T) {
	rr := httptest.NewRecorder()
	ve := validation.Errors{"email": validation.ErrRequired}
	ValidationError(rr, ve)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	assert.False(t, body.Success)
	assert.Equal(t, "Validation failed", body.Error)
	assert.NotEmpty(t, body.Details)
}

func TestValidationError_WithPlainError(t *testing.T) {
	rr := httptest.NewRecorder()
	ValidationError(rr, assert.AnError)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	body := decodeBody(t, rr)
	assert.False(t, body.Success)
	assert.Equal(t, "Validation failed", body.Error)
}

func TestSuccessWithCookies_NoCookieHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	SuccessWithCookies(rr, req, map[string]interface{}{"access_token": "tok"}, "ok")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Result().Cookies(), "no cookies should be set without X-Token-Delivery: cookie")
}

func TestSuccessWithCookies_WithCookieHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set("X-Token-Delivery", "cookie")
	SuccessWithCookies(rr, req, map[string]interface{}{"access_token": "my-token"}, "ok")

	assert.Equal(t, http.StatusOK, rr.Code)
	var found bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == "access_token" {
			found = true
			assert.Equal(t, "my-token", c.Value)
		}
	}
	assert.True(t, found, "access_token cookie should be set")
}

func TestCreatedWithCookies_WithCookieHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	req.Header.Set("X-Token-Delivery", "cookie")
	CreatedWithCookies(rr, req, map[string]interface{}{"access_token": "reg-token"}, "registered")

	assert.Equal(t, http.StatusCreated, rr.Code)
	var found bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == "access_token" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestWithLogger_and_LoggerFromContext(t *testing.T) {
	t.Run("stores and retrieves logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := WithLogger(context.Background(), logger)
		got := LoggerFromContext(ctx)
		assert.Same(t, logger, got)
	})

	t.Run("falls back to slog.Default when none set", func(t *testing.T) {
		got := LoggerFromContext(context.Background())
		assert.Same(t, slog.Default(), got)
	})

	t.Run("nil logger value falls back to default", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), loggerContextKey{}, (*slog.Logger)(nil))
		got := LoggerFromContext(ctx)
		assert.Same(t, slog.Default(), got)
	})
}

func TestHandleServiceError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{"not found", apperror.NewNotFound("tenant"), http.StatusNotFound, "tenant not found"},
		{"conflict", apperror.NewConflict("email already registered"), http.StatusConflict, "email already registered"},
		{"forbidden", apperror.NewForbidden("profile does not belong to user"), http.StatusForbidden, "profile does not belong to user"},
		{"unauthorized", apperror.NewUnauthorized("invalid credentials"), http.StatusUnauthorized, "invalid credentials"},
		{"validation", apperror.NewValidation("cannot delete system policy"), http.StatusBadRequest, "cannot delete system policy"},
		{"internal", apperror.NewInternal("hash password", errors.New("bcrypt failed")), http.StatusInternalServerError, "fallback message"},
		{"untyped", errors.New("unexpected db error"), http.StatusInternalServerError, "fallback message"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			HandleServiceError(rr, req, "fallback message", tc.err)

			assert.Equal(t, tc.wantStatus, rr.Code)
			body := decodeBody(t, rr)
			assert.False(t, body.Success)
			assert.Equal(t, tc.wantError, body.Error)
		})
	}
}

func TestHandleServiceError_UsesContextLogger(t *testing.T) {
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := WithLogger(req.Context(), logger)

	HandleServiceError(rr, req.WithContext(ctx), "oops", apperror.NewInternal("op", errors.New("boom")))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, buf.String(), "internal service error")
	assert.Contains(t, buf.String(), "boom")
}
