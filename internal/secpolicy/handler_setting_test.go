package secpolicy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecuritySettingHandler_GetMFAConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/general", nil)
	w := httptest.NewRecorder()
	h.GetMFAConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_GetMFAConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getMFAConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/general", nil))
	w := httptest.NewRecorder()
	h.GetMFAConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_GetMFAConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getMFAConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{"key": "value"}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/general", nil))
	w := httptest.NewRecorder()
	h.GetMFAConfig(w, r)
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

func TestSecuritySettingHandler_GetLockoutConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/ip", nil)
	w := httptest.NewRecorder()
	h.GetLockoutConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateMFAConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	// Provide user but not tenant; handler fetches user first then checks tenant.
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"mode": "optional"}))
	w := httptest.NewRecorder()
	h.UpdateMFAConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateMFAConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(badJSONReq(t, http.MethodPut, "/security-settings/general"))
	r = withTenant(r)
	w := httptest.NewRecorder()
	h.UpdateMFAConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateMFAConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateMFAConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateMFAConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateMFAConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"mode": "optional"}))
	w := httptest.NewRecorder()
	h.UpdateMFAConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateMFAConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateMFAConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
		getMFAConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"mode": "optional"}))
	w := httptest.NewRecorder()
	h.UpdateMFAConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateMFAConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateMFAConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	// withSecurityCtx injects ClientIPKey + UserAgentKey → covers clientIP != nil and
	// userAgentCtx != nil branches (lines 203-205, 206-208).
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/general", map[string]any{"mode": "optional"})))
	w := httptest.NewRecorder()
	h.UpdateMFAConfig(w, r)
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

// ── GetLockoutConfig ───────────────────────────────────────────────────────────────

func TestSecuritySettingHandler_GetLockoutConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getLockoutConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/ip", nil))
	w := httptest.NewRecorder()
	h.GetLockoutConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_GetLockoutConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getLockoutConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/ip", nil))
	w := httptest.NewRecorder()
	h.GetLockoutConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdatePasswordConfig ──────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdatePasswordConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"min_length": 12}))
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
		updatePasswordConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"min_length": 12}))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdatePasswordConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updatePasswordConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
		getPasswordConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"min_length": 12}))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdatePasswordConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updatePasswordConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/password", map[string]any{"min_length": 12})))
	w := httptest.NewRecorder()
	h.UpdatePasswordConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateSessionConfig ───────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateSessionConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"idle_timeout_minutes": 20}))
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
		updateSessionConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"idle_timeout_minutes": 20}))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateSessionConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateSessionConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
		getSessionConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"idle_timeout_minutes": 20}))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateSessionConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateSessionConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/session", map[string]any{"idle_timeout_minutes": 20})))
	w := httptest.NewRecorder()
	h.UpdateSessionConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateThreatConfig ────────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateThreatConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"risk_step_up_threshold": 21}))
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
		updateThreatConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"risk_step_up_threshold": 21}))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateThreatConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateThreatConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
		getThreatConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"risk_step_up_threshold": 21}))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateThreatConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateThreatConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/threat", map[string]any{"risk_step_up_threshold": 21})))
	w := httptest.NewRecorder()
	h.UpdateThreatConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateLockoutConfig ────────────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateLockoutConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"max_failed_attempts": 5}))
	w := httptest.NewRecorder()
	h.UpdateLockoutConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateLockoutConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(withUser(badJSONReq(t, http.MethodPut, "/security-settings/ip")))
	w := httptest.NewRecorder()
	h.UpdateLockoutConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateLockoutConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateLockoutConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateLockoutConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateLockoutConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"max_failed_attempts": 5}))
	w := httptest.NewRecorder()
	h.UpdateLockoutConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateLockoutConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateLockoutConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
		getLockoutConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"max_failed_attempts": 5}))
	w := httptest.NewRecorder()
	h.UpdateLockoutConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateLockoutConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateLockoutConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/ip", map[string]any{"max_failed_attempts": 5})))
	w := httptest.NewRecorder()
	h.UpdateLockoutConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GetRegistrationConfig ─────────────────────────────────────────────────────

func TestSecuritySettingHandler_GetRegistrationConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/registration", nil)
	w := httptest.NewRecorder()
	h.GetRegistrationConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_GetRegistrationConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getRegistrationConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/registration", nil))
	w := httptest.NewRecorder()
	h.GetRegistrationConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_GetRegistrationConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getRegistrationConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/registration", nil))
	w := httptest.NewRecorder()
	h.GetRegistrationConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GetTokenConfig ────────────────────────────────────────────────────────────

func TestSecuritySettingHandler_GetTokenConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := httptest.NewRequest(http.MethodGet, "/security-settings/token", nil)
	w := httptest.NewRecorder()
	h.GetTokenConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_GetTokenConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		getTokenConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/token", nil))
	w := httptest.NewRecorder()
	h.GetTokenConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_GetTokenConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		getTokenConfigFn: func(tid int64) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/token", nil))
	w := httptest.NewRecorder()
	h.GetTokenConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateRegistrationConfig ──────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateRegistrationConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/registration", map[string]any{"default_role": "member"}))
	w := httptest.NewRecorder()
	h.UpdateRegistrationConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateRegistrationConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(withUser(badJSONReq(t, http.MethodPut, "/security-settings/registration")))
	w := httptest.NewRecorder()
	h.UpdateRegistrationConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateRegistrationConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/registration", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateRegistrationConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateRegistrationConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateRegistrationConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/registration", map[string]any{"default_role": "member"}))
	w := httptest.NewRecorder()
	h.UpdateRegistrationConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateRegistrationConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateRegistrationConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
		getRegistrationConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/registration", map[string]any{"default_role": "member"}))
	w := httptest.NewRecorder()
	h.UpdateRegistrationConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateRegistrationConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateRegistrationConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/registration", map[string]any{"default_role": "member"})))
	w := httptest.NewRecorder()
	h.UpdateRegistrationConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateTokenConfig ─────────────────────────────────────────────────────────

func TestSecuritySettingHandler_UpdateTokenConfig_NoTenant(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withUser(jsonReq(t, http.MethodPut, "/security-settings/token", map[string]any{"clock_skew_leeway_seconds": 30}))
	w := httptest.NewRecorder()
	h.UpdateTokenConfig(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSecuritySettingHandler_UpdateTokenConfig_BadJSON(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(withUser(badJSONReq(t, http.MethodPut, "/security-settings/token")))
	w := httptest.NewRecorder()
	h.UpdateTokenConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateTokenConfig_ValidationError(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/token", map[string]any{}))
	w := httptest.NewRecorder()
	h.UpdateTokenConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateTokenConfig_ServiceError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateTokenConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return nil, errValidation
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/token", map[string]any{"clock_skew_leeway_seconds": 30}))
	w := httptest.NewRecorder()
	h.UpdateTokenConfig(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecuritySettingHandler_UpdateTokenConfig_GetConfigError(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateTokenConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
		getTokenConfigFn: func(tid int64) (map[string]any, error) {
			return nil, assert.AnError
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/token", map[string]any{"clock_skew_leeway_seconds": 30}))
	w := httptest.NewRecorder()
	h.UpdateTokenConfig(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSecuritySettingHandler_UpdateTokenConfig_Success(t *testing.T) {
	svc := &mockSecuritySettingService{
		updateTokenConfigFn: func(tid int64, cfg map[string]any, by int64, ip, ua string) (*SecuritySettingServiceDataResult, error) {
			return &SecuritySettingServiceDataResult{}, nil
		},
	}
	h := NewSecuritySettingHandler(svc)
	r := withSecurityCtx(withTenantAndUser(jsonReq(t, http.MethodPut, "/security-settings/token", map[string]any{"clock_skew_leeway_seconds": 30})))
	w := httptest.NewRecorder()
	h.UpdateTokenConfig(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Default cases in fetchConfigByType / updateConfigByType ──────────────────

func TestSecuritySettingHandler_FetchConfigByType_Default(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/security-settings/general", nil))
	_, err := h.fetchConfigByType(r.Context(), 1, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

func TestSecuritySettingHandler_UpdateConfigByType_Default(t *testing.T) {
	h := NewSecuritySettingHandler(&mockSecuritySettingService{})
	err := h.updateConfigByType(context.Background(), 1, "invalid", map[string]any{}, 10, "1.2.3.4", "agent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}
