package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestRoleHandler_Get_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := httptest.NewRequest(http.MethodGet, "/roles", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_Get_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		getFn: func(service.RoleServiceGetFilter) (*service.RoleServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/roles?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_Get_Success(t *testing.T) {
	svc := &mockRoleService{
		getFn: func(service.RoleServiceGetFilter) (*service.RoleServiceGetResult, error) {
			return &service.RoleServiceGetResult{}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/roles?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleHandler_GetByUUID_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/roles/bad", nil), "role_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockRoleService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.RoleServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRoleHandler_GetByUUID_Success(t *testing.T) {
	svc := &mockRoleService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.RoleServiceDataResult, error) {
			return &service.RoleServiceDataResult{Name: "admin"}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleHandler_Create_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := jsonReq(t, http.MethodPost, "/roles", map[string]string{"name": "r"})
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_Create_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		createFn: func(n, desc string, isDef, isSys bool, s, tUUID string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/roles", map[string]string{
		"name": "role1", "description": "A test description", "status": "active",
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/roles/bad", nil), "role_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_Delete_Success(t *testing.T) {
	svc := &mockRoleService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return &service.RoleServiceDataResult{Name: "role1"}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
