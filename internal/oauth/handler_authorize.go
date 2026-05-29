package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/middleware"
	resp "github.com/maintainerd/auth/internal/platform/response"
)

// OAuthAuthorizeHandler handles the OAuth 2.0 authorization and consent
// endpoints.
type OAuthAuthorizeHandler struct {
	authorizeService OAuthAuthorizeService
}

// NewOAuthAuthorizeHandler creates a new OAuthAuthorizeHandler.
func NewOAuthAuthorizeHandler(authorizeService OAuthAuthorizeService) *OAuthAuthorizeHandler {
	return &OAuthAuthorizeHandler{authorizeService: authorizeService}
}

// Authorize handles GET /oauth/authorize (RFC 6749 §4.1.1). The user must be
// already authenticated (JWT in Authorization header). If consent is needed, the
// response contains a consent_challenge identifier for the frontend to display.
func (h *OAuthAuthorizeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	q := r.URL.Query()
	req := OAuthAuthorizeRequestDTO{
		ResponseType:        q.Get("response_type"),
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, oerr := h.authorizeService.Authorize(r.Context(), req, user.UserID)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	if result.ConsentChallenge != "" {
		resp.Success(w, OAuthConsentRequiredResponseDTO{
			ConsentChallenge: result.ConsentChallenge,
		}, "Consent required")
		return
	}

	resp.Success(w, OAuthAuthorizeResponseDTO{
		RedirectURI: result.RedirectURI,
	}, "Authorization successful")
}

// GetConsentChallenge handles GET /oauth/consent/{challenge_id}. Returns the
// details of a pending consent challenge so the frontend can render the
// consent screen.
func (h *OAuthAuthorizeHandler) GetConsentChallenge(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	challengeUUID, err := uuid.Parse(chi.URLParam(r, "challenge_id"))
	if err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid challenge ID")
		return
	}

	result, svcErr := h.authorizeService.GetConsentChallenge(r.Context(), challengeUUID, user.UserID)
	if svcErr != nil {
		resp.HandleServiceError(w, r, "Failed to retrieve consent challenge", svcErr)
		return
	}

	resp.Success(w, result, "Consent challenge retrieved")
}

// HandleConsent handles POST /oauth/consent. Processes the user's consent
// decision and returns a redirect URI.
func (h *OAuthAuthorizeHandler) HandleConsent(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		resp.Error(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req OAuthConsentDecisionDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	result, oerr := h.authorizeService.HandleConsent(r.Context(), req, user.UserID)
	if oerr != nil {
		oerr.WriteJSON(w)
		return
	}

	resp.Success(w, OAuthConsentDecisionResponseDTO{
		RedirectURI: result.RedirectURI,
	}, "Consent processed")
}
