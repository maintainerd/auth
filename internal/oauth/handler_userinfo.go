package oauth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/maintainerd/auth/internal/platform/middleware"
)

// OAuthUserInfoHandler handles the OpenID Connect UserInfo endpoint.
type OAuthUserInfoHandler struct{}

// NewOAuthUserInfoHandler creates a new OAuthUserInfoHandler.
func NewOAuthUserInfoHandler() *OAuthUserInfoHandler {
	return &OAuthUserInfoHandler{}
}

// UserInfo handles GET /oauth/userinfo (OpenID Connect Core §5.3). Returns
// claims about the authenticated user based on the scopes in the access token.
func (h *OAuthUserInfoHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	user := middleware.AuthFromRequest(r).User
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_token",
			"error_description": "the access token is invalid or has expired",
		})
		return
	}

	claims := middleware.JWTClaimsFromRequest(r)
	tokenScopes := scopeSet(parseScopeFields(""))
	sub := user.UserUUID.String()
	if claims != nil {
		tokenScopes = scopeSet(parseScopeFields(claims.Scope))
		if strings.TrimSpace(claims.Sub) != "" {
			sub = claims.Sub
		}
	}

	resp := OAuthUserInfoResponseDTO{
		Sub: sub,
	}

	if _, ok := tokenScopes["email"]; ok {
		resp.Email = user.Email
		resp.EmailVerified = user.IsEmailVerified
	}
	if _, ok := tokenScopes["phone"]; ok {
		resp.Phone = user.Phone
		resp.PhoneVerified = user.IsPhoneVerified
	}
	if _, ok := tokenScopes["profile"]; ok {
		resp.Name = composeUserDisplayName(user)
		resp.UpdatedAt = user.UpdatedAt.Unix()
		if user.Profile != nil && user.Profile.ProfileURL != nil {
			resp.Picture = *user.Profile.ProfileURL
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// composeUserDisplayName returns the OIDC `name` claim for a user.
// users.fullname column was removed; the name lives in profiles now.
// Order of preference: Profile.DisplayName → FirstName + LastName → user.Fullname
// (the latter is a transient field still populated by some legacy code paths).
func composeUserDisplayName(user *User) string {
	if user == nil {
		return ""
	}
	if user.Profile != nil {
		if user.Profile.DisplayName != nil && strings.TrimSpace(*user.Profile.DisplayName) != "" {
			return strings.TrimSpace(*user.Profile.DisplayName)
		}
		parts := []string{}
		if strings.TrimSpace(user.Profile.FirstName) != "" {
			parts = append(parts, strings.TrimSpace(user.Profile.FirstName))
		}
		if user.Profile.LastName != nil && strings.TrimSpace(*user.Profile.LastName) != "" {
			parts = append(parts, strings.TrimSpace(*user.Profile.LastName))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return user.Fullname
}
