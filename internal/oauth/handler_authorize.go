package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
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

// Authorize handles GET /oauth/authorize (RFC 6749 §4.1.1). It is session-aware:
// when the request carries a valid session it issues a code (or a consent
// challenge); when no session is present it validates the request enough to be
// safe and responds with login_required so the hosted identity app renders the
// login page, after which it re-issues the same request.
func (h *OAuthAuthorizeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := OAuthAuthorizeRequestDTO{
		ResponseType:        q.Get("response_type"),
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Nonce:               q.Get("nonce"),
		IdpHint:             q.Get("idp_hint"),
		Prompt:              q.Get("prompt"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}

	if err := req.Validate(); err != nil {
		resp.ValidationError(w, err)
		return
	}

	// When idp_hint is present, the client is directing the user to a specific
	// upstream provider — start the broker leg unconditionally.
	if req.IdpHint != "" {
		if req.Prompt == "none" {
			if oerr := h.authorizeService.PrepareAuthorize(r.Context(), req); oerr != nil {
				oerr.WriteJSON(w)
				return
			}
			apperror.NewOAuthInteractionRequired("the requested identity provider requires user interaction").WriteJSON(w)
			return
		}
		result, oerr := h.authorizeService.StartBroker(r.Context(), req)
		if oerr != nil {
			oerr.WriteJSON(w)
			return
		}
		resp.Success(w, OAuthAuthorizeResponseDTO{RedirectURI: result.RedirectURI}, "Redirecting to identity provider")
		return
	}

	auth := middleware.AuthFromRequest(r)
	if auth.User == nil || auth.Tenant == nil {
		// No session: validate the client + redirect_uri so we never render a
		// login page for an unknown client or an unregistered redirect, then ask
		// the identity app to log the user in.
		if oerr := h.authorizeService.PrepareAuthorize(r.Context(), req); oerr != nil {
			oerr.WriteJSON(w)
			return
		}
		apperror.NewOAuthLoginRequired("authentication required").WriteJSON(w)
		return
	}

	result, oerr := h.authorizeService.Authorize(r.Context(), req, auth.User.UserID, auth.Tenant.TenantID)
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
		resp.BadRequestBody(w)
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
