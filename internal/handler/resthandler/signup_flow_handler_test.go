package resthandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestSignupFlowHandler_GetAll(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/signup-flows", nil)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/signup-flows?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).GetAll(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockSignupFlowService{
			getAllFn: func(tid int64, name, id *string, status []string, clientUUID *uuid.UUID, pg, lim int, sb, so string) (*service.SignupFlowServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/signup-flows?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).GetAll(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSignupFlowHandler_Get(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", "bad")
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := &mockSignupFlowService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*service.SignupFlowServiceDataResult, error) {
				return nil, errors.New("not found")
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockSignupFlowService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*service.SignupFlowServiceDataResult, error) {
				return &service.SignupFlowServiceDataResult{SignupFlowUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSignupFlowHandler_Create(t *testing.T) {
	clientUUID := uuid.New()
	validBody := map[string]any{
		"name":        "onboarding",
		"description": "Onboarding flow",
		"client_id":   clientUUID.String(),
	}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/signup-flows", validBody)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/signup-flows", map[string]any{"name": ""})
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockSignupFlowService{
			createFn: func(tid int64, name, desc string, cfg map[string]any, status string, clientUUID uuid.UUID) (*service.SignupFlowServiceDataResult, error) {
				return nil, errors.New("create error")
			},
		}
		r := jsonReq(t, http.MethodPost, "/signup-flows", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockSignupFlowService{
			createFn: func(tid int64, name, desc string, cfg map[string]any, status string, cUUID uuid.UUID) (*service.SignupFlowServiceDataResult, error) {
				return &service.SignupFlowServiceDataResult{Name: name, ClientUUID: cUUID}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/signup-flows", validBody)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestSignupFlowHandler_Update(t *testing.T) {
	flowUUID := uuid.New()
	validBody := map[string]any{"name": "updated", "description": "desc"}

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", "bad")
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		svc := &mockSignupFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc string, cfg map[string]any, status string) (*service.SignupFlowServiceDataResult, error) {
				return nil, errors.New("update error")
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockSignupFlowService{
			updateFn: func(id uuid.UUID, tid int64, name, desc string, cfg map[string]any, status string) (*service.SignupFlowServiceDataResult, error) {
				return &service.SignupFlowServiceDataResult{SignupFlowUUID: id, Name: name}, nil
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSignupFlowHandler_Delete(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockSignupFlowService{
			deleteFn: func(id uuid.UUID, tid int64) (*service.SignupFlowServiceDataResult, error) {
				return &service.SignupFlowServiceDataResult{SignupFlowUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSignupFlowHandler_UpdateStatus(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"})
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).UpdateStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockSignupFlowService{
			updateStatusFn: func(id uuid.UUID, tid int64, status string) (*service.SignupFlowServiceDataResult, error) {
				return &service.SignupFlowServiceDataResult{SignupFlowUUID: id, Status: status}, nil
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(svc).UpdateStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSignupFlowHandler_AssignRoles(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{roleUUID.String()}})
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"role_uuids": []string{}})
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).AssignRoles(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSignupFlowHandler_GetRoles(t *testing.T) {
	flowUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).GetRoles(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSignupFlowHandler_RemoveRole(t *testing.T) {
	flowUUID := uuid.New()
	roleUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "signup_flow_uuid", flowUUID.String())
		r = withChiParam(r, "role_uuid", roleUUID.String())
		w := httptest.NewRecorder()
		NewSignupFlowHandler(&mockSignupFlowService{}).RemoveRole(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
