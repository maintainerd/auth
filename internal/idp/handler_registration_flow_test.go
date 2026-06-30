package idp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRegistrationFlowHandler_GetAll(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/registration-flows", nil)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/registration-flows?sort_order=bad", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid client UUID returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/registration-flows?client_id=not-a-uuid", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with rows covers toRegistrationFlowResponseDtoList", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getAllFn: func(tid int64, name, id *string, status []string, clientUUID *uuid.UUID, pg, lim int, sb, so string) (*RegistrationFlowServiceListResult, error) {
				return &RegistrationFlowServiceListResult{
					Data: []RegistrationFlowServiceDataResult{
						{Name: "flow1"},
					},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/registration-flows?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("status filter and valid client_id covers parse success path", func(t *testing.T) {
		clientID := uuid.New()
		url := "/registration-flows?page=1&limit=10&status=active&client_id=" + clientID.String()
		r := jsonReq(t, http.MethodGet, url, nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getAllFn: func(tid int64, name, id *string, status []string, clientUUID *uuid.UUID, pg, lim int, sb, so string) (*RegistrationFlowServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/registration-flows?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRegistrationFlowHandler_Get(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", "bad")
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*RegistrationFlowServiceDataResult, error) {
				return nil, errNotFound
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{RegistrationFlowUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRegistrationFlowHandler_Create(t *testing.T) {
	clientUUID := uuid.New()
	validBody := map[string]any{
		"name":        "onboarding",
		"description": "Onboarding flow",
		"client_id":   clientUUID.String(),
	}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/registration-flows", validBody)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/registration-flows")
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/registration-flows", map[string]any{"name": ""})
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			createFn: func(tid int64, name, desc, status string, clientUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPost, "/registration-flows", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with explicit status covers status != nil branch", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			createFn: func(tid int64, name, desc, status string, cUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{Name: name, ClientUUID: cUUID, Status: status}, nil
			},
		}
		bodyWithStatus := map[string]any{
			"name":        "onboarding",
			"description": "Onboarding flow",
			"client_id":   clientUUID.String(),
			"status":      "inactive",
		}
		r := jsonReq(t, http.MethodPost, "/registration-flows", bodyWithStatus)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			createFn: func(tid int64, name, desc, status string, cUUID uuid.UUID) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{Name: name, ClientUUID: cUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/registration-flows", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestRegistrationFlowHandler_Update(t *testing.T) {
	flowUUID := uuid.New()
	validBody := map[string]any{"name": "updated", "description": "desc"}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", "bad")
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPut, "/")
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"name": ""})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc, status string) (*RegistrationFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with explicit status covers status != nil branch", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc, status string) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{RegistrationFlowUUID: id, Name: name, Status: status}, nil
			},
		}
		bodyWithStatus := map[string]any{"name": "updated", "description": "desc", "status": "inactive"}
		r := jsonReq(t, http.MethodPut, "/", bodyWithStatus)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc, status string) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{RegistrationFlowUUID: id, Name: name}, nil
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRegistrationFlowHandler_Delete(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			deleteFn: func(id uuid.UUID, tid int64) (*RegistrationFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			deleteFn: func(id uuid.UUID, tid int64) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{RegistrationFlowUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRegistrationFlowHandler_UpdateStatus(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPatch, "/")
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateStatusFn: func(id uuid.UUID, tid int64, status string) (*RegistrationFlowServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			updateStatusFn: func(id uuid.UUID, tid int64, status string) (*RegistrationFlowServiceDataResult, error) {
				return &RegistrationFlowServiceDataResult{RegistrationFlowUUID: id, Status: status}, nil
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).UpdateStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRegistrationFlowHandler_AssignRoles(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty registration flow UUID param returns 400", func(t *testing.T) {
		// Without setting chi param, chi.URLParam returns "" triggering the empty check
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid registration flow UUID format returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/")
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{}})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(id uuid.UUID, tid int64, roles []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with roles covers response mapping loop", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			assignRolesFn: func(id uuid.UUID, tid int64, roles []uuid.UUID) ([]RegistrationFlowRoleServiceDataResult, error) {
				return []RegistrationFlowRoleServiceDataResult{
					{RoleUUID: roleUUID, RoleName: "admin"},
				}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).AssignRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRegistrationFlowHandler_GetRoles(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty registration flow UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid registration flow UUID format returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?sort_order=bad", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getRolesFn: func(id uuid.UUID, tid int64, pg, lim int) (*RegistrationFlowRoleServiceListResult, error) {
				return nil, errValidation
			},
		}
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success with rows covers row mapping loop", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			getRolesFn: func(id uuid.UUID, tid int64, pg, lim int) (*RegistrationFlowRoleServiceListResult, error) {
				return &RegistrationFlowRoleServiceListResult{
					Data: []RegistrationFlowRoleServiceDataResult{
						{RoleName: "admin"},
					},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).GetRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRegistrationFlowHandler_RemoveRole(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing both UUID params returns 400", func(t *testing.T) {
		// Neither chi param set → both are "" → triggers the || empty check
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid registration flow UUID returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", "bad-uuid")
		r = withChiParam(r, "role_uuid", roleUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid role UUID returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		r = withChiParam(r, "role_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockRegistrationFlowService{
			removeRoleFn: func(id uuid.UUID, tid int64, rID uuid.UUID) error {
				return errValidation
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		r = withChiParam(r, "role_uuid", roleUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(svc).RemoveRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "registration_flow_uuid", flowUUID.String())
		r = withChiParam(r, "role_uuid", roleUUID.String())
		w := httptest.NewRecorder()
		NewRegistrationFlowHandler(&mockRegistrationFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
