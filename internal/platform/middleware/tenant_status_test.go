package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubTenantStatusResolver struct {
	statusBySlug map[string]string
	err          error
}

func (s *stubTenantStatusResolver) ResolveTenantStatusBySlug(_ context.Context, slug string) (string, bool, error) {
	if s.err != nil {
		return "", false, s.err
	}
	status, ok := s.statusBySlug[slug]
	return status, ok, nil
}

func runTenantStatus(resolver TenantStatusResolver, host string) (*httptest.ResponseRecorder, bool) {
	nextCalled := false
	h := AuthEndpointTenantStatusMiddleware(resolver)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.Host = host
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec, nextCalled
}

// Suspending a tenant in the console used to change nothing: no code path
// compared tenant status before authenticating, so login, registration and
// token issuance all kept working. These pin the enforcement down.
func TestAuthEndpointTenantStatus(t *testing.T) {
	withTenantBases(t) // identity=auth.example.com, console=console.auth.example.com

	t.Run("active tenant passes through", func(t *testing.T) {
		resolver := &stubTenantStatusResolver{statusBySlug: map[string]string{"acme": "active"}}
		rec, next := runTenantStatus(resolver, "acme.auth.example.com")
		assert.True(t, next)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("suspended tenant cannot authenticate", func(t *testing.T) {
		resolver := &stubTenantStatusResolver{statusBySlug: map[string]string{"acme": "suspended"}}
		rec, next := runTenantStatus(resolver, "acme.auth.example.com")
		assert.False(t, next)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("inactive tenant cannot authenticate", func(t *testing.T) {
		resolver := &stubTenantStatusResolver{statusBySlug: map[string]string{"acme": "inactive"}}
		_, next := runTenantStatus(resolver, "acme.auth.example.com")
		assert.False(t, next)
	})

	// Operators must keep console access to the tenant they need to reactivate,
	// so the gate is identity-surface only — same scoping as maintenance.
	t.Run("console surface is never gated", func(t *testing.T) {
		resolver := &stubTenantStatusResolver{statusBySlug: map[string]string{"acme": "suspended"}}
		_, next := runTenantStatus(resolver, "acme.console.auth.example.com")
		assert.True(t, next)
	})

	t.Run("unknown slug is not an auth decision", func(t *testing.T) {
		resolver := &stubTenantStatusResolver{statusBySlug: map[string]string{}}
		_, next := runTenantStatus(resolver, "nobody.auth.example.com")
		assert.True(t, next)
	})

	// "We could not read the tenant's status" must not silently mean "active".
	t.Run("resolver error fails closed", func(t *testing.T) {
		resolver := &stubTenantStatusResolver{err: errors.New("db down")}
		rec, next := runTenantStatus(resolver, "acme.auth.example.com")
		assert.False(t, next)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("unwired resolver is a no-op", func(t *testing.T) {
		_, next := runTenantStatus(nil, "acme.auth.example.com")
		assert.True(t, next)
	})
}
