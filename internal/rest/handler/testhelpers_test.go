package handler

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
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/require"
)

// tenantID is a shared test tenant ID.
const tenantID int64 = 1

// testTenantUUID is a shared UUID for the test tenant.
var testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// testUserUUID is a shared UUID for the test user.
var testUserUUID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

// testResourceUUID is a generic UUID used for resource identifiers in URL params.
var testResourceUUID = uuid.MustParse("00000000-0000-0000-0000-000000000099")

// withTenant injects a *model.Tenant into the request context.
func withTenant(r *http.Request) *http.Request {
	tenant := &model.Tenant{TenantID: tenantID, TenantUUID: testTenantUUID}
	ctx := context.WithValue(r.Context(), middleware.TenantContextKey, tenant)
	return r.WithContext(ctx)
}

// withUser injects only a *model.User into the request context (no tenant).
// Used to test handlers that fetch user before tenant, where we want tenant to be absent.
func withUser(r *http.Request) *http.Request {
	user := &model.User{UserUUID: testUserUUID}
	ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
	return r.WithContext(ctx)
}

// withTenantAndUser injects both tenant and user into the request context.
func withTenantAndUser(r *http.Request) *http.Request {
	tenant := &model.Tenant{TenantID: tenantID, TenantUUID: testTenantUUID}
	user := &model.User{UserUUID: testUserUUID}
	ctx := context.WithValue(r.Context(), middleware.TenantContextKey, tenant)
	ctx = context.WithValue(ctx, middleware.UserContextKey, user)
	return r.WithContext(ctx)
}

// withChiParam injects a chi URL parameter into the request context.
// It preserves any existing chi URL parameters already set on the request.
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
