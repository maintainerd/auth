package oauth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/dpop"
)

// OAuthTokenHandler handles the OAuth 2.0 token, revocation, and
// introspection endpoints.
type OAuthTokenHandler struct {
	tokenService OAuthTokenService
	nonceManager *dpop.NonceManager
	dpopStore    dpop.JTIStore
	// nonceIssuer + dpopResolver enable the RFC 9449 §8 DPoP server-nonce gate
	// for clients with dpop_required=TRUE. Both nil → gate disabled (default).
	nonceIssuer  *dpop.StoreNonceManager
	dpopResolver DPoPRequirementResolver
}

// NewOAuthTokenHandler creates a new OAuthTokenHandler.
func NewOAuthTokenHandler(tokenService OAuthTokenService, nonceManager *dpop.NonceManager, dpopStore dpop.JTIStore) *OAuthTokenHandler {
	return &OAuthTokenHandler{tokenService: tokenService, nonceManager: nonceManager, dpopStore: dpopStore}
}

// SetDPoPNonceGate installs the store-backed DPoP nonce manager and the client
// DPoP-requirement resolver used by the token endpoint's RFC 9449 §8 nonce gate.
// When either is nil the gate is disabled. Call once at startup.
func (h *OAuthTokenHandler) SetDPoPNonceGate(nonceIssuer *dpop.StoreNonceManager, resolver DPoPRequirementResolver) {
	h.nonceIssuer = nonceIssuer
	h.dpopResolver = resolver
}

// Token handles POST /oauth/token (RFC 6749 §4.1.3, §6, §4.4). The
// request body is application/x-www-form-urlencoded.
func (h *OAuthTokenHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest("malformed request body")
		oerr.WriteJSON(w)
		return
	}

	req := OAuthTokenRequestDTO{
		GrantType:           r.PostFormValue("grant_type"),
		Code:                r.PostFormValue("code"),
		RedirectURI:         r.PostFormValue("redirect_uri"),
		CodeVerifier:        r.PostFormValue("code_verifier"),
		RefreshToken:        r.PostFormValue("refresh_token"),
		Scope:               r.PostFormValue("scope"),
		ClientID:            r.PostFormValue("client_id"),
		ClientSecret:        r.PostFormValue("client_secret"),
		ClientAssertionType: r.PostFormValue("client_assertion_type"),
		ClientAssertion:     r.PostFormValue("client_assertion"),
	}

	if err := req.Validate(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest(err.Error())
		oerr.WriteJSON(w)
		return
	}

	dpopProof := r.Header.Get("DPoP")
	if dpopProof != "" && h.dpopStore != nil {
		requestURL := config.AppPublicHostname + "/api/v1/oauth/token"
		claims, err := dpop.ValidateProof(r.Context(), dpopProof, "POST", requestURL, "", h.dpopStore)
		if err != nil {
			if h.nonceManager != nil {
				h.nonceManager.SetNonceHeader(w)
			}
			oerr := apperror.NewOAuthInvalidRequest("invalid_dpop_proof: " + err.Error())
			oerr.WriteJSON(w)
			return
		}
		req.DPoPThumbprint = claims.Thumbprint
	}

	creds := extractClientCredentials(r, req)

	// DPoP server-nonce gate (RFC 9449 §8) — only for clients with
	// dpop_required=TRUE. When the client requires DPoP and the proof carries no
	// valid server nonce, issue one and reply 400 use_dpop_nonce so the client
	// retries with the nonce in its proof.
	if h.enforceDPoPNonce(w, r, creds) {
		return
	}

	result, oerr := h.tokenService.Exchange(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	if dpopProof != "" && h.nonceManager != nil {
		h.nonceManager.SetNonceHeader(w)
	}

	writeTokenResponse(w, result)
}

// enforceDPoPNonce runs the RFC 9449 §8 server-nonce gate. It returns true when
// it has written a response (the caller must stop). It is a no-op (returns
// false) when the gate is not configured, the client cannot be resolved, or the
// client does not require DPoP.
func (h *OAuthTokenHandler) enforceDPoPNonce(w http.ResponseWriter, r *http.Request, creds OAuthClientCredentials) bool {
	if h.nonceIssuer == nil || h.dpopResolver == nil {
		return false
	}
	requirement, ok := h.dpopResolver.ResolveDPoPRequirement(r.Context(), creds.ClientID)
	if !ok || !requirement.Required {
		return false
	}

	nonce := dpop.ExtractProofNonce(r.Header.Get("DPoP"))
	if nonce != "" && h.nonceIssuer.ConsumeNonce(r.Context(), nonce) == nil {
		return false // valid single-use nonce consumed → proceed
	}

	newNonce, err := h.nonceIssuer.IssueNonce(r.Context(), requirement.TenantID, requirement.InternalClientID)
	if err != nil {
		apperror.NewOAuthServerError("failed to issue DPoP nonce").WriteJSON(w)
		return true
	}
	w.Header().Set("DPoP-Nonce", newNonce)
	apperror.NewOAuthUseDPoPNonce("a valid DPoP nonce is required; retry with the provided nonce").WriteJSON(w)
	return true
}

// Revoke handles POST /oauth/revoke (RFC 7009). The request body is
// application/x-www-form-urlencoded.
func (h *OAuthTokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest("malformed request body")
		oerr.WriteJSON(w)
		return
	}

	req := OAuthRevokeRequestDTO{
		Token:         r.PostFormValue("token"),
		TokenTypeHint: r.PostFormValue("token_type_hint"),
		ClientID:      r.PostFormValue("client_id"),
		ClientSecret:  r.PostFormValue("client_secret"),
	}

	if err := req.Validate(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest(err.Error())
		oerr.WriteJSON(w)
		return
	}

	creds := extractClientCredentials(r, OAuthTokenRequestDTO{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
	})

	oerr := h.tokenService.Revoke(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	// RFC 7009 §2.2: respond with 200 OK and empty body.
	w.WriteHeader(http.StatusOK)
}

// Introspect handles POST /oauth/introspect (RFC 7662). The request body is
// application/x-www-form-urlencoded. Client authentication is required per
// RFC 7662 §2.1.
func (h *OAuthTokenHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest("malformed request body")
		oerr.WriteJSON(w)
		return
	}

	req := OAuthIntrospectRequestDTO{
		Token:         r.PostFormValue("token"),
		TokenTypeHint: r.PostFormValue("token_type_hint"),
		ClientID:      r.PostFormValue("client_id"),
		ClientSecret:  r.PostFormValue("client_secret"),
	}

	if err := req.Validate(); err != nil {
		oerr := apperror.NewOAuthInvalidRequest(err.Error())
		oerr.WriteJSON(w)
		return
	}

	creds := extractClientCredentials(r, OAuthTokenRequestDTO{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
	})

	result, oerr := h.tokenService.Introspect(r.Context(), req, creds)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	writeOAuthJSON(w, http.StatusOK, result)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// extractClientCredentials resolves client_id and client_secret from either
// the HTTP Basic Authorization header (RFC 6749 §2.3.1) or the form body
// (§2.3.1 alternative). HTTP Basic takes precedence.
func extractClientCredentials(r *http.Request, req OAuthTokenRequestDTO) OAuthClientCredentials {
	if username, password, ok := parseBasicAuth(r); ok {
		return OAuthClientCredentials{
			ClientID:     username,
			ClientSecret: password,
		}
	}
	return OAuthClientCredentials{
		ClientID:            req.ClientID,
		ClientSecret:        req.ClientSecret,
		ClientAssertionType: req.ClientAssertionType,
		ClientAssertion:     req.ClientAssertion,
	}
}

// parseBasicAuth extracts the username and password from an HTTP Basic
// Authorization header. Returns ("", "", false) when no valid header is present.
func parseBasicAuth(r *http.Request) (string, string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", "", false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// writeTokenResponse writes the token response with the required OAuth 2.0
// cache headers.
func writeTokenResponse(w http.ResponseWriter, result *OAuthTokenResult) {
	resp := OAuthTokenResponseDTO{
		AccessToken:  result.AccessToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		RefreshToken: result.RefreshToken,
		IDToken:      result.IDToken,
		Scope:        result.Scope,
	}
	writeOAuthJSON(w, http.StatusOK, resp)
}

// writeOAuthJSON writes a JSON response with OAuth-required cache headers.
func writeOAuthJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
