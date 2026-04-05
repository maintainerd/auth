package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestUserSettingHandler_CreateOrUpdate_BadJSON(t *testing.T) {
	h := NewUserSettingHandler(&mockUserSettingService{})
	r := httptest.NewRequest(http.MethodPost, "/user-settings", nil)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserSettingHandler_CreateOrUpdate_Success(t *testing.T) {
	svc := &mockUserSettingService{
		createOrUpdateFn: func(
			userUUID uuid.UUID,
			timezone, preferredLanguage, locale *string,
			socialLinks map[string]any,
			preferredContactMethod *string,
			marketingEmailConsent, smsConsent, pushConsent *bool,
			profileVisibility *string,
			dataProcessingConsent *bool,
			termsAcceptedAt, privacyPolicyAcceptedAt *time.Time,
			emergencyName, emergencyPhone, emergencyEmail, emergencyRelation *string,
		) (*service.UserSettingServiceDataResult, error) {
			return &service.UserSettingServiceDataResult{}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-settings", map[string]interface{}{
		"timezone": "UTC",
	}))
	w := httptest.NewRecorder()
	h.CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserSettingHandler_Get_NotFound(t *testing.T) {
	svc := &mockUserSettingService{
		getByUserUUIDFn: func(id uuid.UUID) (*service.UserSettingServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserSettingHandler_Get_Success(t *testing.T) {
	svc := &mockUserSettingService{
		getByUserUUIDFn: func(id uuid.UUID) (*service.UserSettingServiceDataResult, error) {
			return &service.UserSettingServiceDataResult{}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
