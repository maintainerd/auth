package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestSMSConfigHandler_Get_NoTenant(t *testing.T) {
	h := NewSMSConfigHandler(&mockSMSConfigService{})
	w := httptest.NewRecorder()
	h.Get(w, httptest.NewRequest(http.MethodGet, "/sms-config", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSMSConfigHandler_Get_ServiceError(t *testing.T) {
	svc := &mockSMSConfigService{
		getFn: func(_ int64) (*service.SMSConfigServiceDataResult, error) { return nil, assert.AnError },
	}
	h := NewSMSConfigHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withTenant(httptest.NewRequest(http.MethodGet, "/sms-config", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSMSConfigHandler_Get_NotFound(t *testing.T) {
	svc := &mockSMSConfigService{
		getFn: func(_ int64) (*service.SMSConfigServiceDataResult, error) { return nil, errNotFound },
	}
	h := NewSMSConfigHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withTenant(httptest.NewRequest(http.MethodGet, "/sms-config", nil)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSMSConfigHandler_Get_Success(t *testing.T) {
	svc := &mockSMSConfigService{
		getFn: func(_ int64) (*service.SMSConfigServiceDataResult, error) {
			return &service.SMSConfigServiceDataResult{SMSConfigUUID: uuid.New(), Provider: "twilio"}, nil
		},
	}
	h := NewSMSConfigHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withTenant(httptest.NewRequest(http.MethodGet, "/sms-config", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSMSConfigHandler_Update_NoTenant(t *testing.T) {
	h := NewSMSConfigHandler(&mockSMSConfigService{})
	w := httptest.NewRecorder()
	h.Update(w, httptest.NewRequest(http.MethodPut, "/sms-config", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSMSConfigHandler_Update_BadJSON(t *testing.T) {
	h := NewSMSConfigHandler(&mockSMSConfigService{})
	w := httptest.NewRecorder()
	h.Update(w, withTenant(badJSONReq(t, http.MethodPut, "/sms-config")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSMSConfigHandler_Update_ValidationError(t *testing.T) {
	h := NewSMSConfigHandler(&mockSMSConfigService{})
	body := map[string]any{"account_sid": "AC123"}
	w := httptest.NewRecorder()
	h.Update(w, withTenant(jsonReq(t, http.MethodPut, "/sms-config", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSMSConfigHandler_Update_ServiceError(t *testing.T) {
	svc := &mockSMSConfigService{
		updateFn: func(_ int64, _, _, _, _, _ string, _ *bool) (*service.SMSConfigServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewSMSConfigHandler(svc)
	body := map[string]any{"provider": "twilio"}
	w := httptest.NewRecorder()
	h.Update(w, withTenant(jsonReq(t, http.MethodPut, "/sms-config", body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSMSConfigHandler_Update_Success(t *testing.T) {
	svc := &mockSMSConfigService{
		updateFn: func(_ int64, _, _, _, _, _ string, _ *bool) (*service.SMSConfigServiceDataResult, error) {
			return &service.SMSConfigServiceDataResult{SMSConfigUUID: uuid.New(), Provider: "twilio"}, nil
		},
	}
	h := NewSMSConfigHandler(svc)
	body := map[string]any{"provider": "twilio"}
	w := httptest.NewRecorder()
	h.Update(w, withTenant(jsonReq(t, http.MethodPut, "/sms-config", body)))
	assert.Equal(t, http.StatusOK, w.Code)
}
