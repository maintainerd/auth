package apperror

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// OAuthError represents an error response following the OAuth 2.0 error format
// defined in RFC 6749 §5.2. It carries an error code, human-readable description,
// optional URI pointing to error documentation, and the HTTP status code to use.
type OAuthError struct {
	// Code is the OAuth error code (e.g. "invalid_request", "unauthorized_client").
	Code string `json:"error"`
	// Description is a human-readable explanation of the error.
	Description string `json:"error_description,omitempty"`
	// URI is an optional link to a page describing the error in more detail.
	URI string `json:"error_uri,omitempty"`
	// StatusCode is the HTTP status code to return (not serialized to JSON).
	StatusCode int `json:"-"`
}

// Error implements the error interface.
func (e *OAuthError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

// WriteJSON writes the OAuth error response to the http.ResponseWriter.
func (e *OAuthError) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(e.StatusCode)
	_ = json.NewEncoder(w).Encode(e)
}

// RedirectURI builds the redirect URI with error parameters appended as query
// parameters per RFC 6749 §4.1.2.1.
func (e *OAuthError) RedirectURI(redirectURI, state string) string {
	sep := "?"
	for _, c := range redirectURI {
		if c == '?' {
			sep = "&"
			break
		}
	}
	uri := redirectURI + sep + "error=" + e.Code
	if e.Description != "" {
		uri += "&error_description=" + e.Description
	}
	if state != "" {
		uri += "&state=" + state
	}
	return uri
}

// Standard OAuth 2.0 error codes (RFC 6749 §4.1.2.1 and §5.2).

// NewOAuthInvalidRequest creates an error for malformed or missing parameters.
func NewOAuthInvalidRequest(description string) *OAuthError {
	return &OAuthError{
		Code:        "invalid_request",
		Description: description,
		StatusCode:  http.StatusBadRequest,
	}
}

// NewOAuthUnauthorizedClient creates an error when the client is not allowed
// to use the requested grant type or method.
func NewOAuthUnauthorizedClient(description string) *OAuthError {
	return &OAuthError{
		Code:        "unauthorized_client",
		Description: description,
		StatusCode:  http.StatusUnauthorized,
	}
}

// NewOAuthAccessDenied creates an error when the resource owner or server
// denied the request.
func NewOAuthAccessDenied(description string) *OAuthError {
	return &OAuthError{
		Code:        "access_denied",
		Description: description,
		StatusCode:  http.StatusForbidden,
	}
}

// NewOAuthUnsupportedResponseType creates an error when the response_type is
// not supported.
func NewOAuthUnsupportedResponseType(description string) *OAuthError {
	return &OAuthError{
		Code:        "unsupported_response_type",
		Description: description,
		StatusCode:  http.StatusBadRequest,
	}
}

// NewOAuthInvalidScope creates an error when the requested scope is invalid,
// unknown, or malformed.
func NewOAuthInvalidScope(description string) *OAuthError {
	return &OAuthError{
		Code:        "invalid_scope",
		Description: description,
		StatusCode:  http.StatusBadRequest,
	}
}

// NewOAuthServerError creates an error for unexpected internal errors. The
// description should NOT leak internal details.
func NewOAuthServerError(description string) *OAuthError {
	return &OAuthError{
		Code:        "server_error",
		Description: description,
		StatusCode:  http.StatusInternalServerError,
	}
}

// NewOAuthInvalidGrant creates an error when an authorization code, refresh
// token, or other credential is invalid, expired, or revoked.
func NewOAuthInvalidGrant(description string) *OAuthError {
	return &OAuthError{
		Code:        "invalid_grant",
		Description: description,
		StatusCode:  http.StatusBadRequest,
	}
}

// NewOAuthUnsupportedGrantType creates an error when the grant_type is not
// supported by the authorization server.
func NewOAuthUnsupportedGrantType(description string) *OAuthError {
	return &OAuthError{
		Code:        "unsupported_grant_type",
		Description: description,
		StatusCode:  http.StatusBadRequest,
	}
}

// NewOAuthInvalidClient creates an error when client authentication fails.
func NewOAuthInvalidClient(description string) *OAuthError {
	return &OAuthError{
		Code:        "invalid_client",
		Description: description,
		StatusCode:  http.StatusUnauthorized,
	}
}

// NewOAuthLoginRequired creates an error when the user is not authenticated
// and must log in first. Used by the authorization endpoint.
func NewOAuthLoginRequired(description string) *OAuthError {
	return &OAuthError{
		Code:        "login_required",
		Description: description,
		StatusCode:  http.StatusUnauthorized,
	}
}

// NewOAuthConsentRequired creates an error when user consent is required but
// has not been given. Used by the authorization endpoint.
func NewOAuthConsentRequired(description string) *OAuthError {
	return &OAuthError{
		Code:        "consent_required",
		Description: description,
		StatusCode:  http.StatusForbidden,
	}
}
