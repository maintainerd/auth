package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestUserHandler_GetUsers_NoTenant(t *testing.T) {
	h := NewUserHandler(&mockUserService{})
	r := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	h.GetUsers(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_GetUsers_ServiceError(t *testing.T) {
	svc := &mockUserService{
		getFn: func(service.UserServiceGetFilter) (*service.UserServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetUsers(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserHandler_GetUsers_Success(t *testing.T) {
	svc := &mockUserService{
		getFn: func(service.UserServiceGetFilter) (*service.UserServiceGetResult, error) {
			return &service.UserServiceGetResult{}, nil
		},
	}
	h := NewUserHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.GetUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUserByUUID_NoTenant(t *testing.T) {
	h := NewUserHandler(&mockUserService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/users/"+testResourceUUID.String(), nil), "user_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetUser(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_GetUserByUUID_InvalidUUID(t *testing.T) {
	h := NewUserHandler(&mockUserService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/users/bad", nil), "user_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_GetUserByUUID_NotFound(t *testing.T) {
	svc := &mockUserService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.UserServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/users/"+testResourceUUID.String(), nil), "user_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetUser(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserHandler_CreateUser_NoTenant(t *testing.T) {
	h := NewUserHandler(&mockUserService{})
	r := jsonReq(t, http.MethodPost, "/users", map[string]string{"username": "u"})
	w := httptest.NewRecorder()
	h.CreateUser(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_CreateUser_ServiceError(t *testing.T) {
	svc := &mockUserService{
		createFn: func(u, fn string, e, ph *string, pw, s string, meta datatypes.JSON, tUUID string, creator uuid.UUID) (*service.UserServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/users", map[string]any{
		"username": "user1", "fullname": "User One", "password": "P@ssw0rd1!", "status": "active", "tenant_id": testTenantUUID.String(),
	}))
	w := httptest.NewRecorder()
	h.CreateUser(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserHandler_DeleteUser_InvalidUUID(t *testing.T) {
	h := NewUserHandler(&mockUserService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/users/bad", nil), "user_uuid", "bad"))
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_DeleteUser_ServiceError(t *testing.T) {
	svc := &mockUserService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, deleter uuid.UUID) (*service.UserServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserHandler(svc)
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/users/"+testResourceUUID.String(), nil), "user_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserHandler_DeleteUser_Success(t *testing.T) {
	svc := &mockUserService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, deleter uuid.UUID) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		},
	}
	h := NewUserHandler(svc)
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/users/"+testResourceUUID.String(), nil), "user_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
