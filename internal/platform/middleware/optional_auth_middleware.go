package middleware

import (
	"net/http"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
)

// bearerOrCookieToken extracts a bearer token from the Authorization header or,
// failing that, from the access_token / __Host-access_token cookie. It returns
// an empty string when no token is present. API keys are intentionally not
// handled here — this is the cookie/bearer session path used by the hosted
// identity surface.
func bearerOrCookieToken(r *http.Request) string {
	token, _ := bearerOrCookieTokenWithScheme(r)
	return token
}

// bearerOrCookieTokenWithScheme also reports HOW the token was presented, which
// decides whether a sender-constrained token is acceptable (RFC 9449 §7.1).
// "DPoP" is accepted as a scheme here so a bound token reaches the binding check
// rather than looking like no token at all.
func bearerOrCookieTokenWithScheme(r *http.Request) (token, scheme string) {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if parts := strings.SplitN(authHeader, " ", 2); len(parts) == 2 {
			switch strings.ToLower(parts[0]) {
			case "bearer", "dpop":
				return parts[1], strings.ToLower(parts[0])
			}
		}
	}
	for _, name := range []string{"access_token", "__Host-access_token", "__Secure-access_token"} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return cookie.Value, "cookie"
		}
	}
	return "", ""
}

// OptionalUserContextMiddleware populates the request AuthContext when a valid
// session (bearer token or access-token cookie) is present, but never rejects a
// request when one is absent or expired. It is used by session-aware endpoints
// such as GET /oauth/authorize, where the absence of a session means "render
// the login page" (login_required) rather than an outright 401.
//
// Session revocation is not re-checked here: an expired token fails JWT
// validation and logout clears the auth cookies, so a missing or stale cookie
// naturally resolves to the unauthenticated branch.
func OptionalUserContextMiddleware(userProvider UserContextProvider, appCache *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, scheme := bearerOrCookieTokenWithScheme(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			// An ID token must not stand in for a session here either: this
			// middleware populates the same AuthContext the hard path does, and
			// GET /oauth/authorize skips the login screen on the strength of it.
			rawClaims, err := jwt.ValidateAccessTokenWithContext(r.Context(), token)
			if err != nil {
				// Expired, invalid, or not an access token — treat as unauthenticated.
				next.ServeHTTP(w, r)
				return
			}

			// A sender-constrained token presented without a valid proof is not a
			// session. Treating it as one here would reinstate exactly the bypass the
			// binding exists to close, on every session-aware endpoint.
			if err := enforceDPoPBinding(r, scheme, token, rawClaims); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			claims := buildJWTClaims(r.Context(), rawClaims)
			ctx := ContextWithJWTClaims(r.Context(), claims)
			r = r.WithContext(ctx)

			uc := appCache.GetUserContext(ctx, claims.Sub, claims.ClientID)
			if uc == nil {
				loaded, lerr := userProvider.FindBySubAndClientID(ctx, claims.Sub, claims.ClientID)
				if lerr != nil || loaded == nil || loaded.User == nil {
					// Cannot resolve the user — proceed unauthenticated.
					next.ServeHTTP(w, r)
					return
				}
				appCache.SetUserContext(ctx, claims.Sub, claims.ClientID, loaded)
				uc = loaded
			}

			auth := &authctx.AuthContext{
				User:     uc.User,
				Tenant:   uc.Tenant,
				Provider: uc.Provider,
				Client:   uc.Client,
			}
			next.ServeHTTP(w, WithAuthContext(r, auth))
		})
	}
}
