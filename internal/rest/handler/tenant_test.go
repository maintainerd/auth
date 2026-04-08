package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func newTenantHandler(ts *mockTenantService, ms *mockTenantMemberService) *TenantHandler {
	if ts == nil {
		ts = &mockTenantService{}
	}
	if ms == nil {
		ms = &mockTenantMemberService{}
	}
	return NewTenantHandler(ts, ms)
}

func TestTenantHandler_Get(t *testing.T) {
	t.Run("validation error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10&sort_order=bad", nil)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockTenantService{getFn: func(service.TenantServiceGetFilter) (*service.TenantServiceGetResult, error) {
			return nil, errors.New("db error")
		}}
		r := httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Get(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with all filters and rows covers filter+loop branches", func(t *testing.T) {
		svc := &mockTenantService{getFn: func(service.TenantServiceGetFilter) (*service.TenantServiceGetResult, error) {
			return &service.TenantServiceGetResult{Data: []service.TenantServiceDataResult{{Name: "t1"}}}, nil
		}}
		r := httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10&status=active&is_default=true&is_system=false&is_public=true", nil)
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_GetByUUID(t *testing.T) {
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: "tenant1"}, nil
		}}
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_GetDefault(t *testing.T) {
	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockTenantService{getDefaultFn: func() (*service.TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetDefault(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockTenantService{getDefaultFn: func() (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: "default"}, nil
		}}
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetDefault(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_GetByIdentifier(t *testing.T) {
	t.Run("empty identifier returns 400", func(t *testing.T) {
		// no chi param set → chi.URLParam returns ""
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetByIdentifier(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockTenantService{getByIdentifierFn: func(string) (*service.TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "identifier", "my-tenant")
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByIdentifier(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockTenantService{getByIdentifierFn: func(string) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: "t"}, nil
		}}
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "identifier", "my-tenant")
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByIdentifier(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_Create(t *testing.T) {
	validBody := map[string]any{"name": "my-tenant", "display_name": "My Tenant", "description": "A long enough description", "status": "active"}

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/tenants")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/tenants", map[string]any{"name": ""})
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockTenantService{createFn: func(n, dn, desc, s string, isPublic, isDef bool) (*service.TenantServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := jsonReq(t, http.MethodPost, "/tenants", validBody)
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Create(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 201", func(t *testing.T) {
		svc := &mockTenantService{createFn: func(n, dn, desc, s string, isPublic, isDef bool) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: n}, nil
		}}
		r := jsonReq(t, http.MethodPost, "/tenants", validBody)
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestTenantHandler_Update(t *testing.T) {
	validBody := map[string]any{"name": "updated", "display_name": "Updated", "description": "A long enough description", "status": "active"}

	t.Run("no user returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", "bad"))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("IsUserInTenant error returns 500", func(t *testing.T) {
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) {
			return false, errors.New("db error")
		}}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("not a member returns 403", func(t *testing.T) {
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(badJSONReq(t, http.MethodPut, "/"), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": ""}), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		ts := &mockTenantService{updateFn: func(uuid.UUID, string, string, string, string, bool) (*service.TenantServiceDataResult, error) {
			return nil, errors.New("update error")
		}}
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{updateFn: func(id uuid.UUID, n, dn, desc, s string, isPublic bool) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: n}, nil
		}}
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_SetStatus(t *testing.T) {
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "tenant_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withChiParam(badJSONReq(t, http.MethodPatch, "/"), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		ts := &mockTenantService{setStatusByUUIDFn: func(uuid.UUID, string) (*service.TenantServiceDataResult, error) {
			return nil, errors.New("status error")
		}}
		r := withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{setStatusByUUIDFn: func(id uuid.UUID, s string) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Status: s}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_SetPublic(t *testing.T) {
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "tenant_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetPublic(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		ts := &mockTenantService{setActivePublicByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return nil, errors.New("error")
		}}
		r := withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetPublic(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{setActivePublicByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{IsPublic: true}, nil
		}}
		r := withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetPublic(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_SetDefault(t *testing.T) {
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "tenant_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetDefault(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		ts := &mockTenantService{setDefaultStatusByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return nil, errors.New("error")
		}}
		r := withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetDefault(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{setDefaultStatusByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{IsDefault: true}, nil
		}}
		r := withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetDefault(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_Delete(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", "bad"))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("IsUserInTenant error returns 500", func(t *testing.T) {
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) {
			return false, errors.New("db error")
		}}
		r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Delete(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("not a member returns 403", func(t *testing.T) {
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil }}
		r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Delete(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("GetByUUID error returns 404", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Delete(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("system tenant returns 403", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{IsSystem: true}, nil
		}}
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Delete(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("DeleteByUUID error returns 500", func(t *testing.T) {
		ts := &mockTenantService{
			getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
				return &service.TenantServiceDataResult{}, nil
			},
			deleteByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) { return nil, errors.New("delete error") },
		}
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Delete(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{
			getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
				return &service.TenantServiceDataResult{IsSystem: false}, nil
			},
			deleteByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
				return &service.TenantServiceDataResult{Name: "t1"}, nil
			},
		}
		ms := &mockTenantMemberService{isUserInTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_GetMembers(t *testing.T) {
	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10&sort_order=bad", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetByUUID error returns 404", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("ListByTenant error returns 500", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{}, nil
		}}
		ms := &mockTenantMemberService{listByTenantFn: func(int64) ([]service.TenantMemberServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).GetMembers(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with user member covers toTenantMemberResponseDTO User branch", func(t *testing.T) {
		userResult := &service.UserServiceDataResult{Username: "alice"}
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{}, nil
		}}
		ms := &mockTenantMemberService{listByTenantFn: func(int64) ([]service.TenantMemberServiceDataResult, error) {
			return []service.TenantMemberServiceDataResult{{Role: "admin", User: userResult}}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).GetMembers(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_AddMember(t *testing.T) {
	memberUUID := uuid.New()
	validBody := map[string]any{"user_id": memberUUID.String(), "role": "member"}

	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", validBody)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withChiParam(badJSONReq(t, http.MethodPost, "/"), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"user_id": memberUUID.String()}), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetByUUID error returns 404", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).AddMember(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("CreateByUserUUID error returns 400", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{}, nil
		}}
		ms := &mockTenantMemberService{createByUserUUIDFn: func(int64, uuid.UUID, string) (*service.TenantMemberServiceDataResult, error) {
			return nil, errValidation
		}}
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 201", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{}, nil
		}}
		ms := &mockTenantMemberService{createByUserUUIDFn: func(int64, uuid.UUID, string) (*service.TenantMemberServiceDataResult, error) {
			return &service.TenantMemberServiceDataResult{Role: "member"}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).AddMember(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestTenantHandler_UpdateMemberRole(t *testing.T) {
	memberUUID := uuid.New()

	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"role": "admin"})
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "admin"}), "tenant_member_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withChiParam(badJSONReq(t, http.MethodPut, "/"), "tenant_member_uuid", memberUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": ""}), "tenant_member_uuid", memberUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		ms := &mockTenantMemberService{updateRoleFn: func(uuid.UUID, string) (*service.TenantMemberServiceDataResult, error) {
			return nil, errValidation
		}}
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "owner"}), "tenant_member_uuid", memberUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ms := &mockTenantMemberService{updateRoleFn: func(id uuid.UUID, role string) (*service.TenantMemberServiceDataResult, error) {
			return &service.TenantMemberServiceDataResult{Role: role}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "member"}), "tenant_member_uuid", memberUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_RemoveMember(t *testing.T) {
	memberUUID := uuid.New()

	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_member_uuid", "bad")
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		ms := &mockTenantMemberService{deleteByUUIDFn: func(uuid.UUID) error { return errValidation }}
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_member_uuid", memberUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).RemoveMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_member_uuid", memberUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
