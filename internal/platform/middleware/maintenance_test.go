package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/authctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tenantMaintenanceReaderStub struct {
	config map[string]any
	err    error
	calls  int
}

func (s *tenantMaintenanceReaderStub) GetMaintenanceConfig(context.Context, int64) (map[string]any, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.config, nil
}

func tenantMaintenanceHandler(cfg map[string]any) http.Handler {
	return TenantMaintenanceMiddleware(&tenantMaintenanceReaderStub{config: cfg})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
}

func tenantMaintenanceRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	return WithAuthContext(req, &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: 42},
	})
}

func TestOptionalMiddleware_ChainsAllMiddleware(t *testing.T) {
	calls := []string{}
	first := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "first")
			next.ServeHTTP(w, r)
		})
	}
	second := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "second")
			next.ServeHTTP(w, r)
		})
	}
	handler := OptionalMiddleware(first, nil, second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler")
		w.WriteHeader(http.StatusNoContent)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []string{"first", "second", "handler"}, calls)
}

func TestTenantMaintenanceMiddleware_DisabledConfigPasses(t *testing.T) {
	handler := tenantMaintenanceHandler(map[string]any{"enabled": false})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, tenantMaintenanceRequest("/api/v1/profile/"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantMaintenanceMiddleware_NoTenantPasses(t *testing.T) {
	handler := tenantMaintenanceHandler(map[string]any{"enabled": true})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/profile/", nil))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantMaintenanceMiddleware_ActiveConfigBlocks(t *testing.T) {
	handler := tenantMaintenanceHandler(map[string]any{
		"enabled": true,
		"message": "Planned upgrade in progress",
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, tenantMaintenanceRequest("/api/v1/profile/"))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "maintenance_mode")
	assert.Contains(t, w.Body.String(), "Planned upgrade in progress")
}

func TestTenantMaintenanceMiddleware_FutureStartPasses(t *testing.T) {
	handler := tenantMaintenanceHandler(map[string]any{
		"enabled":         true,
		"scheduled_start": time.Now().Add(time.Hour).Format(time.RFC3339),
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, tenantMaintenanceRequest("/api/v1/profile/"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantMaintenanceMiddleware_PastEndPasses(t *testing.T) {
	handler := tenantMaintenanceHandler(map[string]any{
		"enabled":       true,
		"scheduled_end": time.Now().Add(-time.Hour).Format(time.RFC3339),
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, tenantMaintenanceRequest("/api/v1/profile/"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantMaintenanceMiddleware_ActiveWindowSetsRetryAfter(t *testing.T) {
	handler := tenantMaintenanceHandler(map[string]any{
		"enabled":         true,
		"scheduled_start": time.Now().Add(-time.Minute).Format(time.RFC3339),
		"scheduled_end":   time.Now().Add(time.Hour).Format(time.RFC3339),
	})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, tenantMaintenanceRequest("/api/v1/profile/"))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestTenantMaintenanceMiddleware_HealthPathExcluded(t *testing.T) {
	handler := tenantMaintenanceHandler(map[string]any{"enabled": true})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, tenantMaintenanceRequest("/health"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantMaintenanceMiddleware_ReaderErrorPasses(t *testing.T) {
	handler := TenantMaintenanceMiddleware(&tenantMaintenanceReaderStub{err: errors.New("db down")})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, tenantMaintenanceRequest("/api/v1/profile/"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantMaintenanceConfigCache_CachesDecodedConfig(t *testing.T) {
	reader := &tenantMaintenanceReaderStub{config: map[string]any{"enabled": true}}
	cache := newTenantMaintenanceConfigCache(time.Minute)

	first := cache.get(context.Background(), reader, 42)
	second := cache.get(context.Background(), reader, 42)

	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.Equal(t, 1, reader.calls)
	assert.True(t, second.Enabled)
}
