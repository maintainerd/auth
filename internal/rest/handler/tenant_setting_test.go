package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Helpers — one pair (Get/Update) per JSONB section; same pattern throughout
// ---------------------------------------------------------------------------

func tenantSettingResult() *service.TenantSettingServiceDataResult {
	return &service.TenantSettingServiceDataResult{
		TenantSettingUUID: uuid.New(),
		RateLimitConfig:   map[string]any{"max": 200},
		AuditConfig:       map[string]any{"enabled": true},
		MaintenanceConfig: map[string]any{"active": false},
		FeatureFlags:      map[string]any{"beta": true},
	}
}

// ---------------------------------------------------------------------------
// RateLimit
// ---------------------------------------------------------------------------

func TestTenantSettingHandler_GetRateLimitConfig_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	r := httptest.NewRequest(http.MethodGet, "/tenant-settings/rate-limit", nil)
	w := httptest.NewRecorder()
	h.GetRateLimitConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_GetRateLimitConfig_ServiceError(t *testing.T) {
	svc := &mockTenantSettingService{
		getRateLimitConfigFn: func(_ int64) (map[string]any, error) { return nil, assert.AnError },
	}
	h := NewTenantSettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/tenant-settings/rate-limit", nil))
	w := httptest.NewRecorder()
	h.GetRateLimitConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_GetRateLimitConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		getRateLimitConfigFn: func(_ int64) (map[string]any, error) { return map[string]any{"max": 100}, nil },
	}
	h := NewTenantSettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/tenant-settings/rate-limit", nil))
	w := httptest.NewRecorder()
	h.GetRateLimitConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantSettingHandler_UpdateRateLimitConfig_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	r := httptest.NewRequest(http.MethodPut, "/tenant-settings/rate-limit", nil)
	w := httptest.NewRecorder()
	h.UpdateRateLimitConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_UpdateRateLimitConfig_BadJSON(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	r := withTenant(badJSONReq(t, http.MethodPut, "/tenant-settings/rate-limit"))
	w := httptest.NewRecorder()
	h.UpdateRateLimitConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateRateLimitConfig_ValidationError(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	// Empty body means empty map → validation fails
	r := withTenant(jsonReq(t, http.MethodPut, "/tenant-settings/rate-limit", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateRateLimitConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateRateLimitConfig_UpdateError(t *testing.T) {
	svc := &mockTenantSettingService{
		updateRateLimitConfigFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantSettingHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPut, "/tenant-settings/rate-limit", map[string]any{"k": "v"}))
	w := httptest.NewRecorder()
	h.UpdateRateLimitConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateRateLimitConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		updateRateLimitConfigFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return tenantSettingResult(), nil
		},
	}
	h := NewTenantSettingHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPut, "/tenant-settings/rate-limit", map[string]any{"max": 200}))
	w := httptest.NewRecorder()
	h.UpdateRateLimitConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

func TestTenantSettingHandler_GetAuditConfig_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.GetAuditConfig(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_GetAuditConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		getAuditConfigFn: func(_ int64) (map[string]any, error) { return map[string]any{}, nil },
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.GetAuditConfig(w, withTenant(httptest.NewRequest(http.MethodGet, "/", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantSettingHandler_GetAuditConfig_ServiceError(t *testing.T) {
	svc := &mockTenantSettingService{
		getAuditConfigFn: func(_ int64) (map[string]any, error) { return nil, assert.AnError },
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.GetAuditConfig(w, withTenant(httptest.NewRequest(http.MethodGet, "/", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateAuditConfig_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateAuditConfig(w, httptest.NewRequest(http.MethodPut, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_UpdateAuditConfig_BadJSON(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateAuditConfig(w, withTenant(badJSONReq(t, http.MethodPut, "/")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateAuditConfig_ValidationError(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateAuditConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{})))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateAuditConfig_UpdateError(t *testing.T) {
	svc := &mockTenantSettingService{
		updateAuditConfigFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateAuditConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"k": "v"})))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateAuditConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		updateAuditConfigFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return tenantSettingResult(), nil
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateAuditConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"k": "v"})))
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

func TestTenantSettingHandler_GetMaintenanceConfig_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.GetMaintenanceConfig(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_GetMaintenanceConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		getMaintenanceConfigFn: func(_ int64) (map[string]any, error) { return map[string]any{}, nil },
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.GetMaintenanceConfig(w, withTenant(httptest.NewRequest(http.MethodGet, "/", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantSettingHandler_GetMaintenanceConfig_ServiceError(t *testing.T) {
	svc := &mockTenantSettingService{
		getMaintenanceConfigFn: func(_ int64) (map[string]any, error) { return nil, assert.AnError },
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.GetMaintenanceConfig(w, withTenant(httptest.NewRequest(http.MethodGet, "/", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, httptest.NewRequest(http.MethodPut, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_BadJSON(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, withTenant(badJSONReq(t, http.MethodPut, "/")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_ValidationError(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{})))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_UpdateError(t *testing.T) {
	svc := &mockTenantSettingService{
		updateMaintenanceConfigFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"k": "v"})))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		updateMaintenanceConfigFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return tenantSettingResult(), nil
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"k": "v"})))
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// FeatureFlags
// ---------------------------------------------------------------------------

func TestTenantSettingHandler_GetFeatureFlags_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.GetFeatureFlags(w, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_GetFeatureFlags_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		getFeatureFlagsFn: func(_ int64) (map[string]any, error) { return map[string]any{}, nil },
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.GetFeatureFlags(w, withTenant(httptest.NewRequest(http.MethodGet, "/", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantSettingHandler_GetFeatureFlags_ServiceError(t *testing.T) {
	svc := &mockTenantSettingService{
		getFeatureFlagsFn: func(_ int64) (map[string]any, error) { return nil, assert.AnError },
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.GetFeatureFlags(w, withTenant(httptest.NewRequest(http.MethodGet, "/", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateFeatureFlags_NoTenant(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateFeatureFlags(w, httptest.NewRequest(http.MethodPut, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenantSettingHandler_UpdateFeatureFlags_BadJSON(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateFeatureFlags(w, withTenant(badJSONReq(t, http.MethodPut, "/")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateFeatureFlags_ValidationError(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	w := httptest.NewRecorder()
	h.UpdateFeatureFlags(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{})))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateFeatureFlags_UpdateError(t *testing.T) {
	svc := &mockTenantSettingService{
		updateFeatureFlagsFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateFeatureFlags(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"k": "v"})))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateFeatureFlags_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		updateFeatureFlagsFn: func(_ int64, _ map[string]any) (*service.TenantSettingServiceDataResult, error) {
			return tenantSettingResult(), nil
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateFeatureFlags(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"k": "v"})))
	assert.Equal(t, http.StatusOK, w.Code)
}
