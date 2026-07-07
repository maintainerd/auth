package scim

import (
	"context"
	"net/http"
	"strings"
)

// scimTenantIDKey is the context key for the authenticated SCIM tenant ID.
type scimTenantIDKey struct{}

type SCIMBearerAuthenticator interface {
	AuthenticateBearer(ctx context.Context, hash string) (tenantID int64, err error)
}

type scimBearerMiddleware struct {
	repo SCIMConfigurationRepository
}

func NewSCIMBearerMiddleware(repo SCIMConfigurationRepository) func(http.Handler) http.Handler {
	m := &scimBearerMiddleware{repo: repo}
	return m.authenticate
}

func (m *scimBearerMiddleware) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeSCIMError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			writeSCIMError(w, http.StatusUnauthorized, "Empty bearer token")
			return
		}

		hash := hashBearerToken(token)
		cfg, err := m.repo.FindByBearerTokenHash(r.Context(), hash)
		if err != nil || cfg == nil {
			writeSCIMError(w, http.StatusUnauthorized, "Invalid bearer token")
			return
		}

		ctx := context.WithValue(r.Context(), scimTenantIDKey{}, cfg.TenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeSCIMError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	err := newSCIMError(status, detail, "")
	_ = err
	_, _ = w.Write([]byte(`{"schemas":["urn:ietf:params:scim:api:messages:2.0:Error"],"detail":"` + detail + `","status":"` + http.StatusText(status) + `"}`))
}
