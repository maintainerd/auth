package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
)

// Typed test errors — HandleServiceError maps these to the correct HTTP status.
var (
	errNotFound   = apperror.NewNotFoundWithReason("not found")
	errValidation = apperror.NewValidation("validation error")
)

const tenantID int64 = 1

var (
	testTenantUUID   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserUUID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
)

// withTenant injects an authenticated tenant into the request context.
func withTenant(r *http.Request) *http.Request {
	tenant := &cache.AuthTenant{TenantID: tenantID, TenantUUID: testTenantUUID}
	return middleware.WithAuthContext(r, &middleware.AuthContext{Tenant: tenant})
}

// withUser injects only an authenticated user into the request context.
func withUser(r *http.Request) *http.Request {
	user := &cache.AuthUser{UserUUID: testUserUUID}
	return middleware.WithAuthContext(r, &middleware.AuthContext{User: user})
}

// withChiParam injects a chi URL parameter into the request context.
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// badJSONReq creates an HTTP request with an intentionally malformed JSON body.
func badJSONReq(t *testing.T, method, target string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, target, strings.NewReader("{bad json"))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// jsonReq creates an HTTP request with a JSON-encoded body.
func jsonReq(t *testing.T, method, url string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	r := httptest.NewRequest(method, url, &buf)
	r.Header.Set("Content-Type", "application/json")
	return r
}
