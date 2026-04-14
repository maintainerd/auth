package apperror

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuthError_Error(t *testing.T) {
	cases := []struct {
		name     string
		err      *OAuthError
		expected string
	}{
		{
			name:     "with description",
			err:      NewOAuthInvalidRequest("missing client_id"),
			expected: "invalid_request: missing client_id",
		},
		{
			name:     "without description",
			err:      &OAuthError{Code: "server_error", StatusCode: 500},
			expected: "server_error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

func TestOAuthError_WriteJSON(t *testing.T) {
	t.Run("writes correct status and headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		oerr := NewOAuthInvalidRequest("bad param")
		oerr.WriteJSON(w)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		assert.Equal(t, "no-cache", w.Header().Get("Pragma"))

		var body map[string]string
		require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
		assert.Equal(t, "invalid_request", body["error"])
		assert.Equal(t, "bad param", body["error_description"])
	})

	t.Run("server error returns 500", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewOAuthServerError("unexpected").WriteJSON(w)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOAuthError_RedirectURI(t *testing.T) {
	cases := []struct {
		name        string
		redirectURI string
		state       string
		err         *OAuthError
		expected    string
	}{
		{
			name:        "basic redirect with state",
			redirectURI: "https://example.com/callback",
			state:       "abc123",
			err:         NewOAuthAccessDenied("user denied"),
			expected:    "https://example.com/callback?error=access_denied&error_description=user denied&state=abc123",
		},
		{
			name:        "redirect without state",
			redirectURI: "https://example.com/callback",
			state:       "",
			err:         NewOAuthInvalidRequest("bad"),
			expected:    "https://example.com/callback?error=invalid_request&error_description=bad",
		},
		{
			name:        "redirect URI already has query params",
			redirectURI: "https://example.com/callback?foo=bar",
			state:       "xyz",
			err:         NewOAuthServerError("oops"),
			expected:    "https://example.com/callback?foo=bar&error=server_error&error_description=oops&state=xyz",
		},
		{
			name:        "error without description",
			redirectURI: "https://example.com/callback",
			state:       "",
			err:         &OAuthError{Code: "server_error", StatusCode: 500},
			expected:    "https://example.com/callback?error=server_error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.err.RedirectURI(tc.redirectURI, tc.state))
		})
	}
}

func TestOAuthErrorConstructors(t *testing.T) {
	cases := []struct {
		name        string
		constructor func(string) *OAuthError
		code        string
		status      int
	}{
		{"InvalidRequest", NewOAuthInvalidRequest, "invalid_request", 400},
		{"UnauthorizedClient", NewOAuthUnauthorizedClient, "unauthorized_client", 401},
		{"AccessDenied", NewOAuthAccessDenied, "access_denied", 403},
		{"UnsupportedResponseType", NewOAuthUnsupportedResponseType, "unsupported_response_type", 400},
		{"InvalidScope", NewOAuthInvalidScope, "invalid_scope", 400},
		{"ServerError", NewOAuthServerError, "server_error", 500},
		{"InvalidGrant", NewOAuthInvalidGrant, "invalid_grant", 400},
		{"UnsupportedGrantType", NewOAuthUnsupportedGrantType, "unsupported_grant_type", 400},
		{"InvalidClient", NewOAuthInvalidClient, "invalid_client", 401},
		{"LoginRequired", NewOAuthLoginRequired, "login_required", 401},
		{"ConsentRequired", NewOAuthConsentRequired, "consent_required", 403},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.constructor("test description")
			assert.Equal(t, tc.code, err.Code)
			assert.Equal(t, tc.status, err.StatusCode)
			assert.Equal(t, "test description", err.Description)
		})
	}
}
