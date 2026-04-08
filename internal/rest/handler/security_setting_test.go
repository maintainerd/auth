package handler

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

func TestSecuritySettingHandler_GetIPConfig_NoTenant(t *testing.T) {
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

func TestSecuritySettingHandler_UpdateGeneralConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(badJSONReq(t, http.MethodPut, "/security-settings/general"))
	r = withTenant(r)
	w := httptest.NewRecorder()
	h.UpdateGeneralConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateGeneralConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateGeneralConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateGeneralConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateGeneralConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateGeneralConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateGeneralConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateGeneralConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
		getGeneralConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateGeneralConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateGeneralConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateGeneralConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	// withSecurityCtx injects ClientIPKey + UserAgentKey → covers clientIP != nil and
	// userAgentCtx != nil branches (lines 203-205, 206-208).
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"key": "val"})))
	w := httptest.NewRecorder()
	h.UpdateGeneralConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GetPasswordConfig ─────────────────────────────────────────────────────────

func TestSecuritySettingHandler_GetPasswordConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getPasswordConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/password", nil))
	w := httptest.NewRecorder()
	h.GetPasswordConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GetSessionConfig ──────────────────────────────────────────────────────────

func TestSecuritySettingHandler_GetSessionConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getSessionConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/session", nil))
	w := httptest.NewRecorder()
	h.GetSessionConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── GetThreatConfig ───────────────────────────────────────────────────────────

func TestSecuritySettingHandler_GetThreatConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getThreatConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/threat", nil))
	w := httptest.NewRecorder()
	h.GetThreatConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_GetThreatConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getThreatConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/threat", nil))
	w := httptest.NewRecorder()
	h.GetThreatConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GetIPConfig ───────────────────────────────────────────────────────────────

func TestSecuritySettingHandler_GetIPConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getIPConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/ip", nil))
	w := httptest.NewRecorder()
	h.GetIPConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_GetIPConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getIPConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/ip", nil))
	w := httptest.NewRecorder()
	h.GetIPConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdatePasswordConfig ──────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdatePasswordConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdatePasswordConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(withUser(badJSONReq(t, http.MethodPut, "/security-settings/password")))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdatePasswordConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdatePasswordConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updatePasswordConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdatePasswordConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updatePasswordConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
		getPasswordConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdatePasswordConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updatePasswordConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"key": "val"})))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateSessionConfig ───────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateSessionConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateSessionConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(withUser(badJSONReq(t, http.MethodPut, "/security-settings/session")))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateSessionConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateSessionConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateSessionConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateSessionConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateSessionConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
		getSessionConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateSessionConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateSessionConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"key": "val"})))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateThreatConfig ────────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateThreatConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateThreatConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(withUser(badJSONReq(t, http.MethodPut, "/security-settings/threat")))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateThreatConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateThreatConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateThreatConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateThreatConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateThreatConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
		getThreatConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateThreatConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateThreatConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"key": "val"})))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateIPConfig ────────────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateIPConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateIPConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateIPConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(withUser(badJSONReq(t, http.MethodPut, "/security-settings/ip")))
	w := httptest.NewRecorder()
	h.UpdateIPConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateIPConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateIPConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateIPConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateIPConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateIPConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateIPConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateIPConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
		getIPConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"key": "val"}))
	w := httptest.NewRecorder()
	h.UpdateIPConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateIPConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateIPConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
			return &service.SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"key": "val"})))
	w := httptest.NewRecorder()
	h.UpdateIPConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
