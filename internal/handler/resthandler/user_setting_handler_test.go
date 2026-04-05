package resthandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
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

func TestUserSettingHandler_CreateOrUpdate_ValidationError(t *testing.T) {
	// timezone too long triggers validation error
	h := NewUserSettingHandler(&mockUserSettingService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-settings", map[string]any{
		"timezone": "this-timezone-string-is-way-too-long-to-pass-validation-rules",
	}))
	w := httptest.NewRecorder()
	h.CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserSettingHandler_CreateOrUpdate_WithSocialLinks(t *testing.T) {
	// covers the SocialLinks map conversion loop (lines 36-41)
	svc := &mockUserSettingService{
		createOrUpdateFn: func(userUUID uuid.UUID, tz, lang, locale *string, sl map[string]any, pcm *string, mec, sms, push *bool, pv *string, dpc *bool, ta, ppa *time.Time, ecn, ecp, ece, ecr *string) (*service.UserSettingServiceDataResult, error) {
			return &service.UserSettingServiceDataResult{}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-settings", map[string]any{
		"social_links": map[string]string{"twitter": "https://twitter.com/user"},
	}))
	w := httptest.NewRecorder()
	h.CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserSettingHandler_CreateOrUpdate_ServiceError(t *testing.T) {
	svc := &mockUserSettingService{
		createOrUpdateFn: func(userUUID uuid.UUID, tz, lang, locale *string, sl map[string]any, pcm *string, mec, sms, push *bool, pv *string, dpc *bool, ta, ppa *time.Time, ecn, ecp, ece, ecr *string) (*service.UserSettingServiceDataResult, error) {
			return nil, errors.New("save error")
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-settings", map[string]any{"timezone": "UTC"}))
	w := httptest.NewRecorder()
	h.CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserSettingHandler_Delete_NotFound(t *testing.T) {
	h := NewUserSettingHandler(&mockUserSettingService{})
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserSettingHandler_Delete_ServiceError(t *testing.T) {
	settingUUID := uuid.New()
	svc := &mockUserSettingService{
		getByUserUUIDFn: func(uuid.UUID) (*service.UserSettingServiceDataResult, error) {
			return &service.UserSettingServiceDataResult{UserSettingUUID: settingUUID}, nil
		},
		deleteByUUIDFn: func(uuid.UUID) (*service.UserSettingServiceDataResult, error) {
			return nil, errors.New("delete error")
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserSettingHandler_Delete_Success(t *testing.T) {
	settingUUID := uuid.New()
	svc := &mockUserSettingService{
		getByUserUUIDFn: func(uuid.UUID) (*service.UserSettingServiceDataResult, error) {
			return &service.UserSettingServiceDataResult{UserSettingUUID: settingUUID}, nil
		},
		deleteByUUIDFn: func(uuid.UUID) (*service.UserSettingServiceDataResult, error) {
			return &service.UserSettingServiceDataResult{UserSettingUUID: settingUUID}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserSettingHandler_toUserSettingResponseDto_InvalidSocialLinksJSON(t *testing.T) {
	// SocialLinks with invalid JSON bytes covers the unmarshal-error else branch (line 103)
	svc := &mockUserSettingService{
		getByUserUUIDFn: func(uuid.UUID) (*service.UserSettingServiceDataResult, error) {
			return &service.UserSettingServiceDataResult{
				SocialLinks: datatypes.JSON([]byte("not-valid-json")),
			}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
