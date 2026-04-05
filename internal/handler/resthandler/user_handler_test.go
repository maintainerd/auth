package resthandler

import (
	"errors"
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

func TestUserHandler_DeleteUser_NoTenant(t *testing.T) {
	h := NewUserHandler(&mockUserService{})
	r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "user_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
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

// ---------------------------------------------------------------------------
// GetUsers – missing branches
// ---------------------------------------------------------------------------

func TestUserHandler_GetUsers_ValidationError(t *testing.T) {
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users?page=1&limit=10&sort_order=bad", nil))
	w := httptest.NewRecorder()
	NewUserHandler(&mockUserService{}).GetUsers(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_GetUsers_WithFiltersAndRows(t *testing.T) {
	roleID := uuid.New()
	svc := &mockUserService{
		getFn: func(service.UserServiceGetFilter) (*service.UserServiceGetResult, error) {
			ten := &service.TenantServiceDataResult{Name: "t1"}
			return &service.UserServiceGetResult{
				Data:  []service.UserServiceDataResult{{Username: "u1", Tenant: ten}},
				Total: 1, Page: 1, Limit: 10, TotalPages: 1,
			}, nil
		},
	}
	url := "/users?page=1&limit=10&status=active&role_id=" + roleID.String()
	r := withTenant(httptest.NewRequest(http.MethodGet, url, nil))
	w := httptest.NewRecorder()
	NewUserHandler(svc).GetUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetUser – success with Tenant covers toUserResponseDto Tenant branch
// ---------------------------------------------------------------------------

func TestUserHandler_GetUser_Success(t *testing.T) {
	ten := &service.TenantServiceDataResult{Name: "t1"}
	svc := &mockUserService{
		getByUUIDFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{Username: "u1", Tenant: ten}, nil
		},
	}
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "user_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewUserHandler(svc).GetUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// CreateUser – missing branches
// ---------------------------------------------------------------------------

func TestUserHandler_CreateUser_BadJSON(t *testing.T) {
	r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/users"))
	w := httptest.NewRecorder()
	NewUserHandler(&mockUserService{}).CreateUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_CreateUser_ValidationError(t *testing.T) {
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/users", map[string]any{"username": ""}))
	w := httptest.NewRecorder()
	NewUserHandler(&mockUserService{}).CreateUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_CreateUser_Success(t *testing.T) {
	svc := &mockUserService{
		createFn: func(u, fn string, e, ph *string, pw, s string, meta datatypes.JSON, tUUID string, creator uuid.UUID) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{Username: u}, nil
		},
	}
	body := map[string]any{"username": "user1", "fullname": "User One", "password": "P@ssw0rd1!", "status": "active", "tenant_id": testTenantUUID.String()}
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/users", body))
	w := httptest.NewRecorder()
	NewUserHandler(svc).CreateUser(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateUser
// ---------------------------------------------------------------------------

func TestUserHandler_UpdateUser(t *testing.T) {
	validBody := map[string]any{"username": "user1", "fullname": "User One", "status": "active"}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).UpdateUser(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).UpdateUser(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPut, "/"), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).UpdateUser(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"username": ""}), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).UpdateUser(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{updateFn: func(uuid.UUID, int64, string, string, *string, *string, string, datatypes.JSON, uuid.UUID) (*service.UserServiceDataResult, error) {
			return nil, errors.New("update error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).UpdateUser(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockUserService{updateFn: func(id uuid.UUID, tid int64, u, fn string, e, ph *string, s string, meta datatypes.JSON, updater uuid.UUID) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{Username: u}, nil
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).UpdateUser(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// SetUserStatus
// ---------------------------------------------------------------------------

func TestUserHandler_SetUserStatus(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"}), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).SetUserStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{setStatusFn: func(uuid.UUID, int64, string, uuid.UUID) (*service.UserServiceDataResult, error) {
			return nil, errors.New("status error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).SetUserStatus(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockUserService{setStatusFn: func(id uuid.UUID, tid int64, s string, updater uuid.UUID) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{Status: s}, nil
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).SetUserStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// VerifyEmail / VerifyPhone / CompleteAccount
// ---------------------------------------------------------------------------

func TestUserHandler_VerifyEmail(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).VerifyEmail(w, httptest.NewRequest(http.MethodPost, "/", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).VerifyEmail(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{verifyEmailFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return nil, errors.New("verify error")
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).VerifyEmail(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockUserService{verifyEmailFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).VerifyEmail(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserHandler_VerifyPhone(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).VerifyPhone(w, httptest.NewRequest(http.MethodPost, "/", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).VerifyPhone(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{verifyPhoneFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return nil, errors.New("verify error")
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).VerifyPhone(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockUserService{verifyPhoneFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).VerifyPhone(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserHandler_CompleteAccount(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).CompleteAccount(w, httptest.NewRequest(http.MethodPost, "/", nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).CompleteAccount(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{completeAccountFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return nil, errors.New("complete error")
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).CompleteAccount(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockUserService{completeAccountFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).CompleteAccount(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// AssignRoles / RemoveRole
// ---------------------------------------------------------------------------

func TestUserHandler_AssignRoles(t *testing.T) {
	roleID := uuid.New()
	validBody := map[string]any{"role_ids": []string{roleID.String()}}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", validBody)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(badJSONReq(t, http.MethodPost, "/"), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"role_ids": []string{}}), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{assignUserRolesFn: func(uuid.UUID, []uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return nil, errors.New("assign error")
		}}
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockUserService{assignUserRolesFn: func(uuid.UUID, []uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		}}
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestUserHandler_RemoveRole(t *testing.T) {
	roleID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid user UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid role UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "user_uuid", testResourceUUID.String()), "role_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockUserService{removeUserRoleFn: func(uuid.UUID, uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return nil, errors.New("remove error")
		}}
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "user_uuid", testResourceUUID.String()), "role_uuid", roleID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockUserService{removeUserRoleFn: func(uuid.UUID, uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		}}
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "user_uuid", testResourceUUID.String()), "role_uuid", roleID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// GetUserRoles – covers containsIgnoreCase, sortRoles, pagination branches
// ---------------------------------------------------------------------------

// twoRoles returns two roles for sort comparator coverage (need ≥ 2 elements).
func twoRoles() []service.RoleServiceDataResult {
	return []service.RoleServiceDataResult{
		{Name: "admin-role", Description: "Admin description", Status: "active"},
		{Name: "member-role", Description: "Member description", Status: "inactive"},
	}
}

func userRolesReq(t *testing.T, extraQuery string, userUUID uuid.UUID) *http.Request {
	t.Helper()
	url := "/?page=1&limit=10" + extraQuery
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, url, nil), "user_uuid", userUUID.String()))
	return r
}

func userRolesSvc(roles []service.RoleServiceDataResult) *mockUserService {
	return &mockUserService{
		getByUUIDFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		},
		getUserRolesFn: func(uuid.UUID) ([]service.RoleServiceDataResult, error) {
			return roles, nil
		},
	}
}

func TestUserHandler_GetUserRoles(t *testing.T) {
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).GetUserRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1&limit=10&sort_order=bad", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).GetUserRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil), "user_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).GetUserRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("user not found returns 404", func(t *testing.T) {
		svc := &mockUserService{getByUUIDFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return nil, errors.New("not found")
		}}
		w := httptest.NewRecorder()
		NewUserHandler(svc).GetUserRoles(w, userRolesReq(t, "", testResourceUUID))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetUserRoles error returns 500", func(t *testing.T) {
		svc := &mockUserService{
			getByUUIDFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
				return &service.UserServiceDataResult{}, nil
			},
			getUserRolesFn: func(uuid.UUID) ([]service.RoleServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		w := httptest.NewRecorder()
		NewUserHandler(svc).GetUserRoles(w, userRolesReq(t, "", testResourceUUID))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with filters covers containsIgnoreCase and pagination end>len branch", func(t *testing.T) {
		// name+description+status filters; one role matches, one doesn't
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w,
			userRolesReq(t, "&name=admin&description=admin&status=active&sort_by=name&sort_order=asc", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=description covers description case", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w, userRolesReq(t, "&sort_by=description", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=status covers status case", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w, userRolesReq(t, "&sort_by=status", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=created_at covers created_at case", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w, userRolesReq(t, "&sort_by=created_at", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=updated_at covers updated_at case", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w, userRolesReq(t, "&sort_by=updated_at", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("description-only filter hits description continue branch", func(t *testing.T) {
		// "member-role" has Description "Member description" which does NOT contain "Admin",
		// so it hits the continue at line 644; "admin-role" passes and is included.
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w,
			userRolesReq(t, "&description=Admin", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("status-only filter hits status continue branch", func(t *testing.T) {
		// "member-role" has Status "inactive" which != "active", so it hits the continue
		// at line 648; "admin-role" passes and is included.
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w,
			userRolesReq(t, "&status=active", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=name with two roles covers name sort comparator", func(t *testing.T) {
		// No filter so both roles remain; sort.Slice calls comparator on 2 elements → covers line 833.
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w,
			userRolesReq(t, "&sort_by=name&sort_order=asc", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=other+desc covers default case and desc branch", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w, userRolesReq(t, "&sort_by=other&sort_order=desc", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("large page covers offset>len branch", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1000&limit=10", nil), "user_uuid", testResourceUUID.String()))
		NewUserHandler(userRolesSvc(twoRoles())).GetUserRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// GetUserIdentities – covers sortIdentities, Client != nil branch, pagination
// ---------------------------------------------------------------------------

func twoIdentities(withClient bool) []service.UserIdentityServiceDataResult {
	var client *service.ClientServiceDataResult
	if withClient {
		client = &service.ClientServiceDataResult{Name: "web-app"}
	}
	return []service.UserIdentityServiceDataResult{
		{Provider: "google", Sub: "sub-1", Client: client},
		{Provider: "github", Sub: "sub-2"},
	}
}

func userIdentitiesReq(t *testing.T, extraQuery string, userUUID uuid.UUID) *http.Request {
	t.Helper()
	url := "/?page=1&limit=10" + extraQuery
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, url, nil), "user_uuid", userUUID.String()))
	return r
}

func userIdentitiesSvc(identities []service.UserIdentityServiceDataResult) *mockUserService {
	return &mockUserService{
		getByUUIDFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return &service.UserServiceDataResult{}, nil
		},
		getUserIdentsFn: func(uuid.UUID) ([]service.UserIdentityServiceDataResult, error) {
			return identities, nil
		},
	}
}

func TestUserHandler_GetUserIdentities(t *testing.T) {
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil), "user_uuid", "bad"))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).GetUserIdentities(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1&limit=10&sort_order=bad", nil), "user_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).GetUserIdentities(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1&limit=10", nil), "user_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewUserHandler(&mockUserService{}).GetUserIdentities(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("user not found returns 404", func(t *testing.T) {
		svc := &mockUserService{getByUUIDFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
			return nil, errors.New("not found")
		}}
		w := httptest.NewRecorder()
		NewUserHandler(svc).GetUserIdentities(w, userIdentitiesReq(t, "", testResourceUUID))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetUserIdentities error returns 500", func(t *testing.T) {
		svc := &mockUserService{
			getByUUIDFn: func(uuid.UUID, int64) (*service.UserServiceDataResult, error) {
				return &service.UserServiceDataResult{}, nil
			},
			getUserIdentsFn: func(uuid.UUID) ([]service.UserIdentityServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		w := httptest.NewRecorder()
		NewUserHandler(svc).GetUserIdentities(w, userIdentitiesReq(t, "", testResourceUUID))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with Client!=nil covers Client branch", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userIdentitiesSvc(twoIdentities(true))).GetUserIdentities(w,
			userIdentitiesReq(t, "&provider=google", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=provider with two identities covers provider sort comparator", func(t *testing.T) {
		// No provider filter so both identities remain; sort.Slice calls comparator → covers line 858.
		w := httptest.NewRecorder()
		NewUserHandler(userIdentitiesSvc(twoIdentities(false))).GetUserIdentities(w,
			userIdentitiesReq(t, "&sort_by=provider&sort_order=asc", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=sub covers sub case", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userIdentitiesSvc(twoIdentities(false))).GetUserIdentities(w,
			userIdentitiesReq(t, "&sort_by=sub", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=created_at covers created_at case", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userIdentitiesSvc(twoIdentities(false))).GetUserIdentities(w,
			userIdentitiesReq(t, "&sort_by=created_at", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=updated_at covers updated_at case", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userIdentitiesSvc(twoIdentities(false))).GetUserIdentities(w,
			userIdentitiesReq(t, "&sort_by=updated_at", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("sort_by=other+desc covers default case and desc branch", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewUserHandler(userIdentitiesSvc(twoIdentities(false))).GetUserIdentities(w,
			userIdentitiesReq(t, "&sort_by=other&sort_order=desc", testResourceUUID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("large page covers offset>len branch", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/?page=1000&limit=10", nil), "user_uuid", testResourceUUID.String()))
		NewUserHandler(userIdentitiesSvc(twoIdentities(false))).GetUserIdentities(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
