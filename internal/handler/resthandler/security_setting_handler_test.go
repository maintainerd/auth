package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestSecuritySettingHandler_GetGeneralConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/general", nil)
	w := httptest.NewRecorder()
	h.GetGeneralConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_GetGeneralConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getGeneralConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/general", nil))
	w := httptest.NewRecorder()
	h.GetGeneralConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_GetGeneralConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getGeneralConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{"key": "value"}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/general", nil))
	w := httptest.NewRecorder()
	h.GetGeneralConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecuritySettingHandler_GetPasswordConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/password", nil)
	w := httptest.NewRecorder()
	h.GetPasswordConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_GetPasswordConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getPasswordConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/password", nil))
	w := httptest.NewRecorder()
	h.GetPasswordConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecuritySettingHandler_GetSessionConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/session", nil)
	w := httptest.NewRecorder()
	h.GetSessionConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_GetSessionConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getSessionConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/session", nil))
	w := httptest.NewRecorder()
	h.GetSessionConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecuritySettingHandler_GetThreatConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/threat", nil)
	w := httptest.NewRecorder()
	h.GetThreatConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_GetIpConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/ip", nil)
	w := httptest.NewRecorder()
	h.GetIPConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateGeneralConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	// Provide user but not tenant; handler fetches user first then checks tenant.
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateGeneralConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateGeneralConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateGeneralConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateGeneralConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
