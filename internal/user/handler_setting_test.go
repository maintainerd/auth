package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
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
		createOrUpdateFn: func(userUUID uuid.UUID, timezone, preferredLanguage, locale *string) (*UserSettingServiceDataResult, error) {
			return &UserSettingServiceDataResult{}, nil
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
		getByUserUUIDFn: func(id uuid.UUID) (*UserSettingServiceDataResult, error) {
			return nil, errNotFound
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
		getByUserUUIDFn: func(id uuid.UUID) (*UserSettingServiceDataResult, error) {
			return &UserSettingServiceDataResult{}, nil
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

func TestUserSettingHandler_CreateOrUpdate_WithEmptyBody(t *testing.T) {
	svc := &mockUserSettingService{
		createOrUpdateFn: func(userUUID uuid.UUID, tz, lang, locale *string) (*UserSettingServiceDataResult, error) {
			return &UserSettingServiceDataResult{}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-settings", map[string]any{}))
	w := httptest.NewRecorder()
	h.CreateOrUpdate(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserSettingHandler_CreateOrUpdate_ServiceError(t *testing.T) {
	svc := &mockUserSettingService{
		createOrUpdateFn: func(userUUID uuid.UUID, tz, lang, locale *string) (*UserSettingServiceDataResult, error) {
			return nil, errValidation
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
		getByUserUUIDFn: func(uuid.UUID) (*UserSettingServiceDataResult, error) {
			return &UserSettingServiceDataResult{UserSettingUUID: settingUUID}, nil
		},
		deleteByUUIDFn: func(uuid.UUID, int64) (*UserSettingServiceDataResult, error) {
			return nil, errValidation
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
		getByUserUUIDFn: func(uuid.UUID) (*UserSettingServiceDataResult, error) {
			return &UserSettingServiceDataResult{UserSettingUUID: settingUUID}, nil
		},
		deleteByUUIDFn: func(uuid.UUID, int64) (*UserSettingServiceDataResult, error) {
			return &UserSettingServiceDataResult{UserSettingUUID: settingUUID}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserSettingHandler_Get_EmptySetting(t *testing.T) {
	svc := &mockUserSettingService{
		getByUserUUIDFn: func(uuid.UUID) (*UserSettingServiceDataResult, error) {
			return &UserSettingServiceDataResult{}, nil
		},
	}
	h := NewUserSettingHandler(svc)
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/user-settings", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
