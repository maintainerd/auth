package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestEmailConfigHandler_Get_NoTenant(t *testing.T) {
	h := NewEmailConfigHandler(&mockEmailConfigService{})
	w := httptest.NewRecorder()
	h.Get(w, httptest.NewRequest(http.MethodGet, "/email-config", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailConfigHandler_Get_ServiceError(t *testing.T) {
	svc := &mockEmailConfigService{
		getFn: func(_ int64) (*service.EmailConfigServiceDataResult, error) { return nil, assert.AnError },
	}
	h := NewEmailConfigHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withTenant(httptest.NewRequest(http.MethodGet, "/email-config", nil)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEmailConfigHandler_Get_NotFound(t *testing.T) {
	svc := &mockEmailConfigService{
		getFn: func(_ int64) (*service.EmailConfigServiceDataResult, error) { return nil, errNotFound },
	}
	h := NewEmailConfigHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withTenant(httptest.NewRequest(http.MethodGet, "/email-config", nil)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEmailConfigHandler_Get_Success(t *testing.T) {
	svc := &mockEmailConfigService{
		getFn: func(_ int64) (*service.EmailConfigServiceDataResult, error) {
			return &service.EmailConfigServiceDataResult{EmailConfigUUID: uuid.New(), Provider: "smtp"}, nil
		},
	}
	h := NewEmailConfigHandler(svc)
	w := httptest.NewRecorder()
	h.Get(w, withTenant(httptest.NewRequest(http.MethodGet, "/email-config", nil)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEmailConfigHandler_Update_NoTenant(t *testing.T) {
	h := NewEmailConfigHandler(&mockEmailConfigService{})
	w := httptest.NewRecorder()
	h.Update(w, httptest.NewRequest(http.MethodPut, "/email-config", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestEmailConfigHandler_Update_BadJSON(t *testing.T) {
	h := NewEmailConfigHandler(&mockEmailConfigService{})
	w := httptest.NewRecorder()
	h.Update(w, withTenant(badJSONReq(t, http.MethodPut, "/email-config")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailConfigHandler_Update_ValidationError(t *testing.T) {
	h := NewEmailConfigHandler(&mockEmailConfigService{})
	// Missing required fields
	body := map[string]any{"host": "smtp.example.com"}
	w := httptest.NewRecorder()
	h.Update(w, withTenant(jsonReq(t, http.MethodPut, "/email-config", body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEmailConfigHandler_Update_ServiceError(t *testing.T) {
	svc := &mockEmailConfigService{
		updateFn: func(_ int64, _, _ string, _ int, _, _, _, _, _, _ string, _ *bool) (*service.EmailConfigServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewEmailConfigHandler(svc)
	body := map[string]any{
		"provider":     "smtp",
		"from_address": "noreply@example.com",
	}
	w := httptest.NewRecorder()
	h.Update(w, withTenant(jsonReq(t, http.MethodPut, "/email-config", body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEmailConfigHandler_Update_Success(t *testing.T) {
	svc := &mockEmailConfigService{
		updateFn: func(_ int64, _, _ string, _ int, _, _, _, _, _, _ string, _ *bool) (*service.EmailConfigServiceDataResult, error) {
			return &service.EmailConfigServiceDataResult{EmailConfigUUID: uuid.New(), Provider: "smtp"}, nil
		},
	}
	h := NewEmailConfigHandler(svc)
	body := map[string]any{
		"provider":     "smtp",
		"from_address": "noreply@example.com",
	}
	w := httptest.NewRecorder()
	h.Update(w, withTenant(jsonReq(t, http.MethodPut, "/email-config", body)))
	assert.Equal(t, http.StatusOK, w.Code)
}
