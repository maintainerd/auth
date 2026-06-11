package idp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthFlowHandler_GetAll(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/auth-flows", nil)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/auth-flows?sort_order=bad", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid client UUID returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/auth-flows?client_id=not-a-uuid", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with rows covers toAuthFlowResponseDtoList", func(t *testing.T) {
		svc := &mockAuthFlowService{
			getAllFn: func(tid int64, name, id *string, status []string, clientUUID *uuid.UUID, pg, lim int, sb, so string) (*AuthFlowServiceListResult, error) {
				return &AuthFlowServiceListResult{
					Data: []AuthFlowServiceDataResult{
						{Name: "flow1"},
					},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/auth-flows?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("status filter and valid client_id covers parse success path", func(t *testing.T) {
		clientID := uuid.New()
		url := "/auth-flows?page=1&limit=10&status=active&client_id=" + clientID.String()
		r := jsonReq(t, http.MethodGet, url, nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAuthFlowService{
			getAllFn: func(tid int64, name, id *string, status []string, clientUUID *uuid.UUID, pg, lim int, sb, so string) (*AuthFlowServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/auth-flows?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAuthFlowHandler_Get(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", "bad")
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockAuthFlowService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*AuthFlowServiceDataResult, error) {
				return nil, errNotFound
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAuthFlowService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*AuthFlowServiceDataResult, error) {
				return &AuthFlowServiceDataResult{AuthFlowUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthFlowHandler_Create(t *testing.T) {
	clientUUID := uuid.New()
	validBody := map[string]any{
		"name":        "onboarding",
		"description": "Onboarding flow",
		"client_id":   clientUUID.String(),
	}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/auth-flows", validBody)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/auth-flows")
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/auth-flows", map[string]any{"name": ""})
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockAuthFlowService{
			createFn: func(tid int64, name, desc, status string, clientUUID uuid.UUID) (*AuthFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPost, "/auth-flows", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with explicit status covers status != nil branch", func(t *testing.T) {
		svc := &mockAuthFlowService{
			createFn: func(tid int64, name, desc, status string, cUUID uuid.UUID) (*AuthFlowServiceDataResult, error) {
				return &AuthFlowServiceDataResult{Name: name, ClientUUID: cUUID, Status: status}, nil
			},
		}
		bodyWithStatus := map[string]any{
			"name":        "onboarding",
			"description": "Onboarding flow",
			"client_id":   clientUUID.String(),
			"status":      "inactive",
		}
		r := jsonReq(t, http.MethodPost, "/auth-flows", bodyWithStatus)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAuthFlowService{
			createFn: func(tid int64, name, desc, status string, cUUID uuid.UUID) (*AuthFlowServiceDataResult, error) {
				return &AuthFlowServiceDataResult{Name: name, ClientUUID: cUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/auth-flows", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestAuthFlowHandler_Update(t *testing.T) {
	flowUUID := uuid.New()
	validBody := map[string]any{"name": "updated", "description": "desc"}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", "bad")
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPut, "/")
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"name": ""})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockAuthFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc, status string) (*AuthFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with explicit status covers status != nil branch", func(t *testing.T) {
		svc := &mockAuthFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc, status string) (*AuthFlowServiceDataResult, error) {
				return &AuthFlowServiceDataResult{AuthFlowUUID: id, Name: name, Status: status}, nil
			},
		}
		bodyWithStatus := map[string]any{"name": "updated", "description": "desc", "status": "inactive"}
		r := jsonReq(t, http.MethodPut, "/", bodyWithStatus)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAuthFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc, status string) (*AuthFlowServiceDataResult, error) {
				return &AuthFlowServiceDataResult{AuthFlowUUID: id, Name: name}, nil
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthFlowHandler_Delete(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockAuthFlowService{
			deleteFn: func(id uuid.UUID, tid int64) (*AuthFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAuthFlowService{
			deleteFn: func(id uuid.UUID, tid int64) (*AuthFlowServiceDataResult, error) {
				return &AuthFlowServiceDataResult{AuthFlowUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthFlowHandler_UpdateStatus(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPatch, "/")
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockAuthFlowService{
			updateStatusFn: func(id uuid.UUID, tid int64, status string) (*AuthFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAuthFlowService{
			updateStatusFn: func(id uuid.UUID, tid int64, status string) (*AuthFlowServiceDataResult, error) {
				return &AuthFlowServiceDataResult{AuthFlowUUID: id, Status: status}, nil
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).UpdateStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthFlowHandler_AssignRoles(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty signup flow UUID param returns 400", func(t *testing.T) {
		// Without setting chi param, chi.URLParam returns "" triggering the empty check
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid signup flow UUID format returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/")
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{}})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockAuthFlowService{
			assignRolesFn: func(id uuid.UUID, tid int64, roles []uuid.UUID) ([]AuthFlowRoleServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with roles covers response mapping loop", func(t *testing.T) {
		svc := &mockAuthFlowService{
			assignRolesFn: func(id uuid.UUID, tid int64, roles []uuid.UUID) ([]AuthFlowRoleServiceDataResult, error) {
				return []AuthFlowRoleServiceDataResult{
					{RoleUUID: roleUUID, RoleName: "admin"},
				}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthFlowHandler_GetRoles(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty signup flow UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid signup flow UUID format returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?sort_order=bad", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockAuthFlowService{
			getRolesFn: func(id uuid.UUID, tid int64, pg, lim int) (*AuthFlowRoleServiceListResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with rows covers row mapping loop", func(t *testing.T) {
		svc := &mockAuthFlowService{
			getRolesFn: func(id uuid.UUID, tid int64, pg, lim int) (*AuthFlowRoleServiceListResult, error) {
				return &AuthFlowRoleServiceListResult{
					Data: []AuthFlowRoleServiceDataResult{
						{RoleName: "admin"},
					},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).GetRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthFlowHandler_RemoveRole(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing both UUID params returns 400", func(t *testing.T) {
		// Neither chi param set → both are "" → triggers the || empty check
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid signup flow UUID returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", "bad-uuid")
		r = withChiParam(r, "role_uuid", roleUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid role UUID returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		r = withChiParam(r, "role_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockAuthFlowService{
			removeRoleFn: func(id uuid.UUID, tid int64, rID uuid.UUID) error {
				return errValidation
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		r = withChiParam(r, "role_uuid", roleUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "auth_flow_uuid", flowUUID.String())
		r = withChiParam(r, "role_uuid", roleUUID.String())
		w := httptest.NewRecorder()
		NewAuthFlowHandler(&mockAuthFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
