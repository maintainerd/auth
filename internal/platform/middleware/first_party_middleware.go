package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	resp "github.com/maintainerd/maintainerd-auth/internal/platform/response"
)

// FirstPartyClientResolver reports whether an OAuth client identifier belongs to
// a client this deployment owns.
type FirstPartyClientResolver interface {
	IsFirstPartyClient(ctx context.Context, clientIdentifier string) bool
}

// RequireFirstPartyClient guards the end-user self-service API.
//
// /account, /profiles, /mfa, trusted devices, data erasure and identity linking
// are authorized on the SUBJECT alone: any valid access token for that user
// passes. That is correct for the hosted login app, and dangerous for anyone
// else — an access token minted for a third-party OAuth client the user
// consented to for `openid profile` could otherwise change their email, rotate
// their password, enumerate and revoke their sessions, or strip their MFA.
// Consenting to sign in with an application must never hand that application
// the account itself.
//
// A cookie-authenticated request is by definition the hosted first-party app
// (a third-party client never holds this server's session cookie), so those
// pass through and are covered by CSRF instead.
//
// Modelled on RequireManagementClient: both aud and client_id must be present
// and agree, so a token with a missing or mismatched private claim cannot slip
// past the allowlist.
func RequireFirstPartyClient(resolver FirstPartyClientResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolver == nil {
				// Fail closed: with no resolver there is no way to tell a
				// first-party token from a third-party one.
				resp.Error(w, http.StatusForbidden, "first-party client check is not configured")
				return
			}

			token, scheme := bearerOrCookieTokenWithScheme(r)

			// Only an ABSENT token falls through, to let the per-route JWT
			// middleware produce the canonical 401.
			//
			// This deliberately does NOT exempt cookie-borne tokens. It used to,
			// on the premise that "a third-party client never holds this server's
			// session cookie" — which is false. A cookie is just a request header
			// the caller sets, jwt_middleware accepts access_token /
			// __Host-access_token as a bearer, and the CSRF double-submit is
			// trivially satisfied by any non-browser caller that sends matching
			// cookie and header values. So the guard could be bypassed verbatim by
			// moving the same third-party token out of Authorization and into a
			// cookie, which returned the full account, the GDPR PII export, and
			// session revocation. RequireManagementClient never had this hole
			// because it checks the token whatever transport carried it.
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			rawClaims, err := jwt.ValidateAccessTokenWithContext(r.Context(), token)
			if err != nil {
				// Not a usable access token — let the per-route JWT middleware
				// produce the canonical 401 rather than deciding here.
				next.ServeHTTP(w, r)
				return
			}
			if err := enforceDPoPBinding(r, scheme, token, rawClaims); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			clientID := strings.TrimSpace(stringClaimValue(rawClaims, "client_id"))
			audience := strings.TrimSpace(stringClaimValue(rawClaims, "aud"))
			if clientID == "" || audience == "" || audience != clientID ||
				!resolver.IsFirstPartyClient(r.Context(), clientID) {
				resp.Error(w, http.StatusForbidden,
					"this token was issued to a third-party application and cannot manage the account")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func stringClaimValue(claims map[string]any, key string) string {
	v, _ := claims[key].(string)
	return v
}
