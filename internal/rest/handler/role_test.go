package handler

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

// ── Get ───────────────────────────────────────────────────────────────────────

func TestRoleHandler_Get_ValidationError(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/roles?sort_order=bad", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Get_WithFiltersAndRows(t *testing.T) {
	perms := []service.PermissionServiceDataResult{{Name: "read"}}
	svc := &mockRoleService{
		getFn: func(service.RoleServiceGetFilter) (*service.RoleServiceGetResult, error) {
			return &service.RoleServiceGetResult{
				Data: []service.RoleServiceDataResult{{Name: "admin", Permissions: &perms}},
			}, nil
		},
	}
	h := NewRoleHandler(svc)
	// is_default=true, is_system=false, status=active cover all optional filter branches.
	// Result with Permissions != nil covers toRoleResponseDTO Permissions branch.
	r := withTenant(httptest.NewRequest(http.MethodGet,
		"/roles?page=1&limit=10&is_default=true&is_system=false&status=active", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestRoleHandler_Create_NoUser(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(jsonReq(t, http.MethodPost, "/roles", map[string]string{"name": "r"}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_Create_BadJSON(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/roles"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Create_ValidationError(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/roles", map[string]any{}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Create_Success(t *testing.T) {
	svc := &mockRoleService{
		createFn: func(n, desc string, isDef, isSys bool, s, tUUID string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return &service.RoleServiceDataResult{Name: n}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/roles", map[string]string{
		"name": "role1", "description": "A test description", "status": "active",
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestRoleHandler_Delete_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withChiParam(httptest.NewRequest(http.MethodDelete, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_Delete_NoUser(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestRoleHandler_Update_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withChiParam(httptest.NewRequest(http.MethodPut, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_Update_NoUser(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPut, "/roles/"+testResourceUUID.String(), nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_Update_InvalidUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPut, "/roles/bad", nil), "role_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Update_BadJSON(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPut, "/roles/"+testResourceUUID.String()), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Update_ValidationError(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/roles/"+testResourceUUID.String(), map[string]any{}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_Update_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		updateFn: func(id uuid.UUID, tid int64, n, desc string, isDef, isSys bool, s string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/roles/"+testResourceUUID.String(), map[string]string{
		"name": "role1", "description": "A test description", "status": "active",
	}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_Update_Success(t *testing.T) {
	svc := &mockRoleService{
		updateFn: func(id uuid.UUID, tid int64, n, desc string, isDef, isSys bool, s string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return &service.RoleServiceDataResult{Name: n}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/roles/"+testResourceUUID.String(), map[string]string{
		"name": "role1", "description": "A test description", "status": "active",
	}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── SetStatus ─────────────────────────────────────────────────────────────────

func TestRoleHandler_SetStatus_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withChiParam(httptest.NewRequest(http.MethodPatch, "/roles/"+testResourceUUID.String()+"/status", nil), "role_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_SetStatus_NoUser(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPatch, "/roles/"+testResourceUUID.String()+"/status", nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_SetStatus_InvalidUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/roles/bad/status", nil), "role_uuid", "bad"))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_SetStatus_BadJSON(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPatch, "/roles/"+testResourceUUID.String()+"/status"), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_SetStatus_InvalidStatus(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/roles/"+testResourceUUID.String()+"/status", map[string]string{"status": "invalid"}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_SetStatus_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/roles/"+testResourceUUID.String()+"/status", map[string]string{"status": "active"}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_SetStatus_Success(t *testing.T) {
	svc := &mockRoleService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return &service.RoleServiceDataResult{}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/roles/"+testResourceUUID.String()+"/status", map[string]string{"status": "inactive"}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GetPermissions ────────────────────────────────────────────────────────────

func TestRoleHandler_GetPermissions_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/roles/"+testResourceUUID.String()+"/permissions", nil), "role_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetPermissions(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_GetPermissions_InvalidUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/roles/bad/permissions", nil), "role_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetPermissions(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_GetPermissions_ValidationError(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/roles/"+testResourceUUID.String()+"/permissions?sort_order=bad", nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetPermissions(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_GetPermissions_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		getRolePermissionsFn: func(service.RoleServiceGetPermissionsFilter) (*service.RoleServiceGetPermissionsResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/roles/"+testResourceUUID.String()+"/permissions?page=1&limit=10", nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetPermissions(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_GetPermissions_Success(t *testing.T) {
	apiResult := &service.APIServiceDataResult{Name: "users-api"}
	svc := &mockRoleService{
		getRolePermissionsFn: func(service.RoleServiceGetPermissionsFilter) (*service.RoleServiceGetPermissionsResult, error) {
			return &service.RoleServiceGetPermissionsResult{
				Data: []service.PermissionServiceDataResult{
					{Name: "read", API: apiResult}, // API != nil covers line 407-419
					{Name: "write"},                // API == nil covers status filter + row loop
				},
			}, nil
		},
	}
	h := NewRoleHandler(svc)
	// status filter covers the status != "" branch (lines 357-359)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/roles/"+testResourceUUID.String()+"/permissions?page=1&limit=10&status=active", nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetPermissions(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── AddPermissions ────────────────────────────────────────────────────────────

func TestRoleHandler_AddPermissions_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withChiParam(httptest.NewRequest(http.MethodPost, "/roles/"+testResourceUUID.String()+"/permissions", nil), "role_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.AddPermissions(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_AddPermissions_NoUser(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/roles/"+testResourceUUID.String()+"/permissions", nil), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.AddPermissions(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_AddPermissions_InvalidUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPost, "/roles/bad/permissions", nil), "role_uuid", "bad"))
	w := httptest.NewRecorder()
	h.AddPermissions(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_AddPermissions_BadJSON(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPost, "/roles/"+testResourceUUID.String()+"/permissions"), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.AddPermissions(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_AddPermissions_ValidationError(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	// Empty permissions slice fails RoleAddPermissionsRequestDTO.Validate()
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/roles/"+testResourceUUID.String()+"/permissions", map[string]any{"permissions": []uuid.UUID{}}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.AddPermissions(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_AddPermissions_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		addRolePermsFn: func(id uuid.UUID, tid int64, perms []uuid.UUID, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/roles/"+testResourceUUID.String()+"/permissions", map[string]any{"permissions": []uuid.UUID{testResourceUUID}}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.AddPermissions(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_AddPermissions_Success(t *testing.T) {
	svc := &mockRoleService{
		addRolePermsFn: func(id uuid.UUID, tid int64, perms []uuid.UUID, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return &service.RoleServiceDataResult{}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/roles/"+testResourceUUID.String()+"/permissions", map[string]any{"permissions": []uuid.UUID{testResourceUUID}}), "role_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.AddPermissions(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── RemovePermission ──────────────────────────────────────────────────────────

func TestRoleHandler_RemovePermission_NoTenant(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "role_uuid", testResourceUUID.String()), "permission_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.RemovePermission(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_RemovePermission_NoUser(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "role_uuid", testResourceUUID.String()), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.RemovePermission(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleHandler_RemovePermission_InvalidRoleUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "role_uuid", "bad"))
	w := httptest.NewRecorder()
	h.RemovePermission(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_RemovePermission_InvalidPermissionUUID(t *testing.T) {
	h := NewRoleHandler(&mockRoleService{})
	r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "role_uuid", testResourceUUID.String()), "permission_uuid", "bad"))
	w := httptest.NewRecorder()
	h.RemovePermission(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleHandler_RemovePermission_ServiceError(t *testing.T) {
	svc := &mockRoleService{
		removeRolePermsFn: func(id uuid.UUID, tid int64, perm uuid.UUID, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "role_uuid", testResourceUUID.String()), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.RemovePermission(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRoleHandler_RemovePermission_Success(t *testing.T) {
	svc := &mockRoleService{
		removeRolePermsFn: func(id uuid.UUID, tid int64, perm uuid.UUID, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
			return &service.RoleServiceDataResult{}, nil
		},
	}
	h := NewRoleHandler(svc)
	r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "role_uuid", testResourceUUID.String()), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.RemovePermission(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
