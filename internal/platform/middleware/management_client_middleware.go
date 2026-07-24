package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// ManagementClientResolver reports whether a client identifier belongs to a
// first-party management client permitted on the internal (management) API
// surface. client.ClientService satisfies this interface.
type ManagementClientResolver interface {
	IsManagementClient(ctx context.Context, clientIdentifier string) bool
}

// RequireManagementClient guards the internal management API. That surface is
// only intended for the first-party admin console (the auth-console client); a
// token minted for any other client must not be accepted there, even if its
// subject happens to hold the required permissions.
//
// It enforces only when a JWT is actually presented. Token-less requests (setup,
// public tenant reads, health probes) and API-key requests pass through so their
// own auth rules apply, and invalid tokens pass through so the per-route JWT
// middleware returns the canonical 401. A valid JWT whose client_id is not a
// management client is rejected with 403.
func RequireManagementClient(resolver ManagementClientResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, scheme := bearerOrCookieTokenWithScheme(r)

			// No JWT, or an API key: defer to the per-route auth rules.
			if token == "" || strings.HasPrefix(token, "ak_") {
				next.ServeHTTP(w, r)
				return
			}

			rawClaims, err := jwt.ValidateTokenWithContext(r.Context(), token)
			if err != nil {
				// Let the per-route JWT middleware produce the canonical 401.
				next.ServeHTTP(w, r)
				return
			}

			// A bound token without a valid proof must not be treated as a valid JWT
			// for the management-client decision; the per-route JWT middleware will
			// produce the canonical 401.
			if err := enforceDPoPBinding(r, scheme, token, rawClaims); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			clientID, _ := rawClaims["client_id"].(string)
			audience, _ := rawClaims["aud"].(string)
			clientID = strings.TrimSpace(clientID)
			audience = strings.TrimSpace(audience)
			// Access tokens minted by this server bind both aud and client_id to
			// the relying-party identifier. Requiring both prevents a valid token
			// with a missing or mismatched private claim from bypassing the
			// management-client allowlist.
			if clientID == "" || audience == "" || audience != clientID ||
				!resolver.IsManagementClient(r.Context(), clientID) {
				resp.Error(w, http.StatusForbidden, "token is not valid for the management API")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
