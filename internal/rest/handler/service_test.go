package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestServiceHandler_Get_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := httptest.NewRequest(http.MethodGet, "/services", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceHandler_Get_ServiceError(t *testing.T) {
	svc := &mockServiceService{
		getFn: func(service.ServiceServiceGetFilter) (*service.ServiceServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/services?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceHandler_Get_Success(t *testing.T) {
	svc := &mockServiceService{
		getFn: func(service.ServiceServiceGetFilter) (*service.ServiceServiceGetResult, error) {
			return &service.ServiceServiceGetResult{}, nil
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/services?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServiceHandler_GetByUUID_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/services/"+testResourceUUID.String(), nil), "service_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/services/bad", nil), "service_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockServiceService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ServiceServiceDataResult, error) {
			return nil, errNotFound
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/services/"+testResourceUUID.String(), nil), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServiceHandler_GetByUUID_Success(t *testing.T) {
	svc := &mockServiceService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ServiceServiceDataResult, error) {
			return &service.ServiceServiceDataResult{Name: "svc"}, nil
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/services/"+testResourceUUID.String(), nil), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServiceHandler_Create_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := jsonReq(t, http.MethodPost, "/services", map[string]string{"name": "s"})
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceHandler_Create_ServiceError(t *testing.T) {
	svc := &mockServiceService{
		createFn: func(n, dn, desc, v string, isSys bool, s string, tid int64) (*service.ServiceServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/services", map[string]interface{}{
		"name": "svc1", "display_name": "Svc1", "description": "Service description", "version": "v1", "status": "active",
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/services/bad", nil), "service_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockServiceService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.ServiceServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/services/"+testResourceUUID.String(), nil), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceHandler_Delete_Success(t *testing.T) {
	svc := &mockServiceService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.ServiceServiceDataResult, error) {
			return &service.ServiceServiceDataResult{Name: "svc1"}, nil
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/services/"+testResourceUUID.String(), nil), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestServiceHandler_Get_ValidationError(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/services?sort_order=bad", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_Get_WithFiltersAndRows(t *testing.T) {
	svc := &mockServiceService{
		getFn: func(service.ServiceServiceGetFilter) (*service.ServiceServiceGetResult, error) {
			return &service.ServiceServiceGetResult{
				Data: []service.ServiceServiceDataResult{{Name: "svc1"}},
			}, nil
		},
	}
	h := NewServiceHandler(svc)
	// is_system + status filter branches; result with rows covers the loop body.
	r := withTenant(httptest.NewRequest(http.MethodGet,
		"/services?page=1&limit=10&is_system=true&status=active", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestServiceHandler_Create_BadJSON(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(badJSONReq(t, http.MethodPost, "/services"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_Create_ValidationError(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(jsonReq(t, http.MethodPost, "/services", map[string]any{}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_Create_Success(t *testing.T) {
	svc := &mockServiceService{
		createFn: func(n, dn, desc, v string, isSys bool, s string, tid int64) (*service.ServiceServiceDataResult, error) {
			return &service.ServiceServiceDataResult{Name: n}, nil
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/services", map[string]any{
		"name": "svc1", "display_name": "Svc1", "description": "Service description",
		"version": "v1", "status": "active",
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestServiceHandler_Delete_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withChiParam(httptest.NewRequest(http.MethodDelete, "/services/"+testResourceUUID.String(), nil), "service_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestServiceHandler_Update_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withChiParam(httptest.NewRequest(http.MethodPut, "/services/"+testResourceUUID.String(), nil), "service_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceHandler_Update_InvalidUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPut, "/services/bad", nil), "service_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_Update_BadJSON(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPut, "/services/"+testResourceUUID.String()), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_Update_ValidationError(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/services/"+testResourceUUID.String(), map[string]any{}), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_Update_ServiceError(t *testing.T) {
	svc := &mockServiceService{
		updateFn: func(id uuid.UUID, tid int64, n, dn, desc, v string, isSys bool, s string) (*service.ServiceServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/services/"+testResourceUUID.String(), map[string]any{
		"name": "svc1", "display_name": "Svc1", "description": "Service description",
		"version": "v1", "status": "active",
	}), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceHandler_Update_Success(t *testing.T) {
	svc := &mockServiceService{
		updateFn: func(id uuid.UUID, tid int64, n, dn, desc, v string, isSys bool, s string) (*service.ServiceServiceDataResult, error) {
			return &service.ServiceServiceDataResult{Name: n}, nil
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/services/"+testResourceUUID.String(), map[string]any{
		"name": "svc1", "display_name": "Svc1", "description": "Service description",
		"version": "v1", "status": "active",
	}), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── SetStatus ─────────────────────────────────────────────────────────────────

func TestServiceHandler_SetStatus_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withChiParam(httptest.NewRequest(http.MethodPatch, "/services/"+testResourceUUID.String()+"/status", nil), "service_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceHandler_SetStatus_InvalidUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPatch, "/services/bad/status", nil), "service_uuid", "bad"))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_SetStatus_BadJSON(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPatch, "/services/"+testResourceUUID.String()+"/status"), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_SetStatus_ValidationError(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/services/"+testResourceUUID.String()+"/status", map[string]any{}), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_SetStatus_ServiceError(t *testing.T) {
	svc := &mockServiceService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string) (*service.ServiceServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/services/"+testResourceUUID.String()+"/status", map[string]any{"status": "active"}), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceHandler_SetStatus_Success(t *testing.T) {
	svc := &mockServiceService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string) (*service.ServiceServiceDataResult, error) {
			return &service.ServiceServiceDataResult{}, nil
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/services/"+testResourceUUID.String()+"/status", map[string]any{"status": "inactive"}), "service_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── AssignPolicy ──────────────────────────────────────────────────────────────

func TestServiceHandler_AssignPolicy_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withChiParam(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.AssignPolicy(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceHandler_AssignPolicy_InvalidServiceUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "service_uuid", "bad"))
	w := httptest.NewRecorder()
	h.AssignPolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_AssignPolicy_InvalidPolicyUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", "bad"))
	w := httptest.NewRecorder()
	h.AssignPolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_AssignPolicy_ServiceError(t *testing.T) {
	svc := &mockServiceService{
		assignPolicyFn: func(svcID, polID uuid.UUID, tid int64) error {
			return assert.AnError
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.AssignPolicy(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceHandler_AssignPolicy_Success(t *testing.T) {
	svc := &mockServiceService{
		assignPolicyFn: func(svcID, polID uuid.UUID, tid int64) error { return nil },
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.AssignPolicy(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── RemovePolicy ──────────────────────────────────────────────────────────────

func TestServiceHandler_RemovePolicy_NoTenant(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.RemovePolicy(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestServiceHandler_RemovePolicy_InvalidServiceUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "service_uuid", "bad"))
	w := httptest.NewRecorder()
	h.RemovePolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_RemovePolicy_InvalidPolicyUUID(t *testing.T) {
	h := NewServiceHandler(&mockServiceService{})
	r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", "bad"))
	w := httptest.NewRecorder()
	h.RemovePolicy(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceHandler_RemovePolicy_ServiceError(t *testing.T) {
	svc := &mockServiceService{
		removePolicyFn: func(svcID, polID uuid.UUID, tid int64) error {
			return assert.AnError
		},
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.RemovePolicy(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestServiceHandler_RemovePolicy_Success(t *testing.T) {
	svc := &mockServiceService{
		removePolicyFn: func(svcID, polID uuid.UUID, tid int64) error { return nil },
	}
	h := NewServiceHandler(svc)
	r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "service_uuid", testResourceUUID.String()), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.RemovePolicy(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
