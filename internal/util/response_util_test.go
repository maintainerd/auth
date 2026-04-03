package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

