package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Helpers — one pair (Get/Update) per JSONB section; same pattern throughout
// ---------------------------------------------------------------------------

func tenantSettingResult() *TenantSettingServiceDataResult {
	return &TenantSettingServiceDataResult{
		TenantSettingUUID: uuid.New(),
		RateLimitConfig:   map[string]any{"max": 200},
		AuditConfig:       map[string]any{"enabled": true},
		MaintenanceConfig: map[string]any{"active": false},
	}
}

type mockAuthEventService struct {
	logFn func(authevent.AuthEventInput)
}

func (m *mockAuthEventService) Log(_ context.Context, input authevent.AuthEventInput) {
	if m.logFn != nil {
		m.logFn(input)
	}
}
func (m *mockAuthEventService) FindPaginated(_ context.Context, _ authevent.AuthEventRepositoryGetFilter) (*authevent.PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return &authevent.PaginationResult[authevent.AuthEventServiceDataResult]{}, nil
}
func (m *mockAuthEventService) FindByUUID(_ context.Context, _ int64, _ uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}
func (m *mockAuthEventService) CountByEventType(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (m *mockAuthEventService) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAuthEventService) Shutdown() {}

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

func TestTenantSettingHandler_UpdateRateLimitConfig_RejectsUnknownFields(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{})
	r := withTenant(jsonReq(t, http.MethodPut, "/tenant-settings/rate-limit", map[string]any{"unknown": true}))
	w := httptest.NewRecorder()
	h.UpdateRateLimitConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateRateLimitConfig_UpdateError(t *testing.T) {
	svc := &mockTenantSettingService{
		updateRateLimitConfigFn: func(_ int64, _ map[string]any) (*TenantSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantSettingHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPut, "/tenant-settings/rate-limit", map[string]any{"enabled": true}))
	w := httptest.NewRecorder()
	h.UpdateRateLimitConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateRateLimitConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		updateRateLimitConfigFn: func(_ int64, _ map[string]any) (*TenantSettingServiceDataResult, error) {
			return tenantSettingResult(), nil
		},
	}
	h := NewTenantSettingHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPut, "/tenant-settings/rate-limit", map[string]any{"enabled": true}))
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
		updateAuditConfigFn: func(_ int64, _ map[string]any) (*TenantSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateAuditConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"enabled": true})))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateAuditConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		updateAuditConfigFn: func(_ int64, _ map[string]any) (*TenantSettingServiceDataResult, error) {
			return tenantSettingResult(), nil
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateAuditConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"enabled": true})))
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
		updateMaintenanceConfigFn: func(_ int64, _ map[string]any) (*TenantSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"enabled": true})))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_Success(t *testing.T) {
	svc := &mockTenantSettingService{
		updateMaintenanceConfigFn: func(_ int64, _ map[string]any) (*TenantSettingServiceDataResult, error) {
			return tenantSettingResult(), nil
		},
	}
	h := NewTenantSettingHandler(svc)
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"enabled": true})))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_RejectsRemovedFields(t *testing.T) {
	h := NewTenantSettingHandler(&mockTenantSettingService{
		updateMaintenanceConfigFn: func(_ int64, _ map[string]any) (*TenantSettingServiceDataResult, error) {
			t.Fatal("UpdateMaintenanceConfig should not be called for invalid config")
			return nil, nil
		},
	})
	w := httptest.NewRecorder()
	h.UpdateMaintenanceConfig(w, withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{
		"bypass_ips": []string{"127.0.0.1"},
	})))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantSettingHandler_UpdateMaintenanceConfig_LogsAuditEvent(t *testing.T) {
	svc := &mockTenantSettingService{
		updateMaintenanceConfigFn: func(_ int64, cfg map[string]any) (*TenantSettingServiceDataResult, error) {
			return &TenantSettingServiceDataResult{MaintenanceConfig: cfg}, nil
		},
	}
	var logged *authevent.AuthEventInput
	events := &mockAuthEventService{logFn: func(input authevent.AuthEventInput) {
		logged = &input
	}}
	h := NewTenantSettingHandler(svc, events)
	req := withTenant(jsonReq(t, http.MethodPut, "/", map[string]any{"enabled": true}))
	req.RemoteAddr = "10.0.0.5:1234"
	req.Header.Set("User-Agent", "tenant-test")
	w := httptest.NewRecorder()

	h.UpdateMaintenanceConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	if assert.NotNil(t, logged) {
		assert.Equal(t, tenantID, logged.TenantID)
		assert.Equal(t, authevent.AuthEventCategorySystem, logged.Category)
		assert.Equal(t, authevent.AuthEventTypeMaintenanceConfigUpdated, logged.EventType)
		assert.Equal(t, authevent.AuthEventSeverityWarn, logged.Severity)
		assert.Equal(t, authevent.AuthEventResultSuccess, logged.Result)
		assert.Equal(t, "10.0.0.5", logged.IPAddress)
	}
}
