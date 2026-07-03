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
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if parts := strings.SplitN(authHeader, " ", 2); len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}
	for _, name := range []string{"access_token", "__Host-access_token"} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return cookie.Value
		}
	}
	return ""
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
			token := bearerOrCookieToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			rawClaims, err := jwt.ValidateTokenWithContext(r.Context(), token)
			if err != nil {
				// Expired or invalid session — treat as unauthenticated.
				next.ServeHTTP(w, r)
				return
			}

			claims := buildJWTClaims(rawClaims)
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
