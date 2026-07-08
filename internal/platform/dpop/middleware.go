package dpop

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsContextKey contextKey = "dpop_claims"

// Middleware returns an HTTP middleware that optionally validates a DPoP proof
// when present. When the DPoP header is absent the request is treated as a
// standard Bearer request. When the header is present but invalid, the request
// is rejected with 401.
//
// The validated *Claims are stored in the request context under the key
// exported by ClaimsFromContext.
func Middleware(store JTIStore, getRequestURL func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proof := r.Header.Get("DPoP")
			if proof == "" {
				// No DPoP header — pass through as normal Bearer request.
				next.ServeHTTP(w, r)
				return
			}

			// Determine the access token for ath computation.
			authHeader := r.Header.Get("Authorization")
			accessToken := ""
			if strings.HasPrefix(authHeader, "DPoP ") {
				accessToken = strings.TrimPrefix(authHeader, "DPoP ")
			}

			ath := ""
			if accessToken != "" {
				ath = AccessTokenHash(accessToken)
			}

			requestURL := r.URL.String()
			if getRequestURL != nil {
				requestURL = getRequestURL(r)
			}

			claims, err := ValidateProof(r.Context(), proof, r.Method, requestURL, ath, store)
			if err != nil {
				http.Error(w, `{"error":"invalid_dpop_proof","error_description":"DPoP proof validation failed"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves the DPoP *Claims stored by Middleware.
// Returns nil when the request did not carry a DPoP proof.
func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsContextKey).(*Claims)
	return c
}
