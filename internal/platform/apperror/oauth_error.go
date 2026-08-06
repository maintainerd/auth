package apperror

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// OAuthError represents an error response following the OAuth 2.0 error format
// defined in RFC 6749 §5.2. It carries an error code, human-readable description,
// optional URI pointing to error documentation, and the HTTP status code to use.
type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description,omitempty"`
	URI         string `json:"error_uri,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	StatusCode  int    `json:"-"`
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
	// Values are escaped rather than concatenated: `state` reaches us
	// URL-decoded, so a raw `&` or `#` in it would re-partition the callback
	// query — letting a caller inject their own parameters or truncate ours into
	// a fragment. Parse failure falls back to the caller's string unchanged
	// rather than emitting a malformed URI.
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("error", e.Code)
	if e.Description != "" {
		q.Set("error_description", e.Description)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
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

// NewOAuthInvalidTarget creates an error when the requested resource or audience
// is not one the caller may address. RFC 8693 §2.2.2 defines invalid_target for
// exactly this; returning invalid_request instead would tell a caller its syntax
// was wrong rather than its target.
func NewOAuthInvalidTarget(description string) *OAuthError {
	return &OAuthError{
		Code:        "invalid_target",
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

// NewOAuthInteractionRequired creates an error when an authorization request
// cannot complete without user interaction (for example, an upstream broker
// redirect requested together with prompt=none).
func NewOAuthInteractionRequired(description string) *OAuthError {
	return &OAuthError{
		Code:        "interaction_required",
		Description: description,
		StatusCode:  http.StatusForbidden,
	}
}

// NewOAuthUseDPoPNonce creates the RFC 9449 §8 error instructing a DPoP client
// to retry the request including the server-provided nonce (sent in the
// DPoP-Nonce response header) in its DPoP proof.
func NewOAuthUseDPoPNonce(description string) *OAuthError {
	return &OAuthError{
		Code:        "use_dpop_nonce",
		Description: description,
		StatusCode:  http.StatusBadRequest,
	}
}
