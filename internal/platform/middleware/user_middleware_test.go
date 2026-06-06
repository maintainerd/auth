package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMiniredisClient starts an in-process Redis and returns a client pointing at it.
func newMiniredisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, cli
}

// ---------------------------------------------------------------------------
// Mock: UserRepository (middleware package scope)
// ---------------------------------------------------------------------------

// mockContextProvider implements UserContextProvider with ctx support.
type mockContextProvider struct {
	findFn func(sub, cID string) (*authctx.UserContext, error)
}

func (m *mockContextProvider) FindBySubAndClientID(_ context.Context, sub, cID string) (*authctx.UserContext, error) {
	if m.findFn != nil {
		return m.findFn(sub, cID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newFakeCache returns a Cache backed by an unreachable Redis so every
// cache operation fails immediately, exercising the database-fallback path.
func newFakeCache() *cache.Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:        "localhost:0",
		DialTimeout: 20 * time.Millisecond,
		ReadTimeout: 20 * time.Millisecond,
	})
	return cache.New(rdb)
}

// withJWTContext injects sub and clientID into the request context as a
// JWTClaims struct, simulating what JWTAuthMiddleware does at runtime.
func withJWTContext(r *http.Request, sub, clientID string) *http.Request {
	return WithJWTClaims(r, &JWTClaims{Sub: sub, ClientID: clientID})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestUserContextMiddleware(t *testing.T) {
	const sub = "user-sub-123"
	const clientID = "client-abc"
	userUUID := uuid.New()

	cases := []struct {
		name              string
		findBySubClientID func(sub, cID string) (*authctx.UserContext, error)
		wantStatus        int
		checkContext      func(t *testing.T, captured *authctx.AuthUser)
	}{
		{
			name: "user found -> context populated -> 200",
			findBySubClientID: func(_, _ string) (*authctx.UserContext, error) {
				return &authctx.UserContext{User: &authctx.AuthUser{UserID: 1, UserUUID: userUUID}}, nil
			},
			wantStatus: http.StatusOK,
			checkContext: func(t *testing.T, captured *authctx.AuthUser) {
				require.NotNil(t, captured)
				assert.Equal(t, userUUID, captured.UserUUID)
			},
		},
		{
			name:              "nil context -> 401",
			findBySubClientID: func(_, _ string) (*authctx.UserContext, error) { return nil, nil },
			wantStatus:        http.StatusUnauthorized,
		},
		{
			name: "context with nil user -> 401",
			findBySubClientID: func(_, _ string) (*authctx.UserContext, error) {
				return &authctx.UserContext{}, nil
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "db error -> 500",
			findBySubClientID: func(_, _ string) (*authctx.UserContext, error) {
				return nil, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured *authctx.AuthUser
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = AuthFromRequest(r).User
				w.WriteHeader(http.StatusOK)
			})

			repo := &mockContextProvider{findFn: tc.findBySubClientID}
			mw := UserContextMiddleware(repo, newFakeCache())

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withJWTContext(req, sub, clientID)
			rr := httptest.NewRecorder()
			mw(next).ServeHTTP(rr, req)

			assert.Equal(t, tc.wantStatus, rr.Code)
			if tc.checkContext != nil {
				tc.checkContext(t, captured)
			}
		})
	}
}

func TestUserContextMiddleware_CacheHit(t *testing.T) {
	const sub = "user-sub-cache"
	const clientID = "client-cache"
	userUUID := uuid.New()

	// Seed the cache via the cache package
	payload := authctx.UserContext{User: &authctx.AuthUser{UserUUID: userUUID}}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	mr, redisCli := newMiniredisClient(t)
	cacheKey := cache.UserContextKeyFor(sub, clientID)
	require.NoError(t, mr.Set(cacheKey, string(data)))

	appCache := cache.New(redisCli)

	var captured *authctx.AuthUser
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = AuthFromRequest(r).User
		w.WriteHeader(http.StatusOK)
	})

	mw := UserContextMiddleware(&mockContextProvider{}, appCache)
	req := withJWTContext(httptest.NewRequest(http.MethodGet, "/", nil), sub, clientID)
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, captured)
	assert.Equal(t, userUUID, captured.UserUUID)
}

// TestUserContextMiddleware_PopulatesTenant verifies the middleware propagates
// the tenant (and other context fields) resolved by the provider into the
// request AuthContext, so downstream tenant-scoped handlers can read it.
func TestUserContextMiddleware_PopulatesTenant(t *testing.T) {
	const sub = "user-sub-identity"
	const clientID = "my-client-id"
	userUUID := uuid.New()
	tenantUUID := uuid.New()

	resolved := &authctx.UserContext{
		User:   &authctx.AuthUser{UserUUID: userUUID},
		Tenant: &authctx.AuthTenant{TenantID: 42, TenantUUID: tenantUUID},
	}

	var capturedTenant *authctx.AuthTenant
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenant = AuthFromRequest(r).Tenant
		w.WriteHeader(http.StatusOK)
	})

	repo := &mockContextProvider{
		findFn: func(_, _ string) (*authctx.UserContext, error) { return resolved, nil },
	}
	mw := UserContextMiddleware(repo, newFakeCache())
	req := withJWTContext(httptest.NewRequest(http.MethodGet, "/", nil), sub, clientID)
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, capturedTenant)
	assert.Equal(t, int64(42), capturedTenant.TenantID)
	assert.Equal(t, tenantUUID, capturedTenant.TenantUUID)
}
