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

func validPolicyBody() map[string]any {
	return map[string]any{
		"name":    "pol1",
		"version": "v1",
		"status":  "active",
		"document": map[string]any{
			"version": "v1",
			"statement": []map[string]any{
				{"effect": "allow", "action": []string{"read"}, "resource": []string{"*"}},
			},
		},
	}
}

func TestPolicyHandler_Get_NoTenant(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := httptest.NewRequest(http.MethodGet, "/policies", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHandler_Get_ServiceError(t *testing.T) {
	svc := &mockPolicyService{
		getFn: func(service.PolicyServiceGetFilter) (*service.PolicyServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPolicyHandler_Get_Success(t *testing.T) {
	svc := &mockPolicyService{
		getFn: func(service.PolicyServiceGetFilter) (*service.PolicyServiceGetResult, error) {
			return &service.PolicyServiceGetResult{}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPolicyHandler_GetByUUID_NoTenant(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/policies/"+testResourceUUID.String(), nil), "policy_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/policies/bad", nil), "policy_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockPolicyService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.PolicyServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/policies/"+testResourceUUID.String(), nil), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPolicyHandler_GetByUUID_Success(t *testing.T) {
	svc := &mockPolicyService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.PolicyServiceDataResult, error) {
			return &service.PolicyServiceDataResult{Name: "pol"}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/policies/"+testResourceUUID.String(), nil), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPolicyHandler_Create_NoTenant(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := jsonReq(t, http.MethodPost, "/policies", map[string]string{"name": "p"})
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHandler_Create_ServiceError(t *testing.T) {
	svc := &mockPolicyService{
		createFn: func(tid int64, name string, desc *string, doc datatypes.JSON, ver, status string, isSys bool) (*service.PolicyServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/policies", map[string]any{
		"name":    "pol1",
		"version": "v1",
		"status":  "active",
		"document": map[string]any{
			"version": "v1",
			"statement": []map[string]any{
				{"effect": "allow", "action": []string{"read"}, "resource": []string{"*"}},
			},
		},
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPolicyHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/policies/bad", nil), "policy_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockPolicyService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.PolicyServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/policies/"+testResourceUUID.String(), nil), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPolicyHandler_GetServicesByPolicyUUID_NoTenant(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/policies/"+testResourceUUID.String()+"/services", nil), "policy_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetServicesByPolicyUUID(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── Get ─────────────────────────────────────────────────────────────────────

func TestPolicyHandler_Get_ValidationError(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies?sort_order=bad", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_Get_WithFilters(t *testing.T) {
	svc := &mockPolicyService{
		getFn: func(f service.PolicyServiceGetFilter) (*service.PolicyServiceGetResult, error) {
			return &service.PolicyServiceGetResult{
				Data: []service.PolicyServiceDataResult{{Name: "pol1"}},
			}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet,
		"/policies?name=n&description=d&version=v1&status=active&is_system=true&service_id="+testResourceUUID.String(), nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPolicyHandler_Get_MultiStatus(t *testing.T) {
	svc := &mockPolicyService{
		getFn: func(f service.PolicyServiceGetFilter) (*service.PolicyServiceGetResult, error) {
			return &service.PolicyServiceGetResult{}, nil
		},
	}
	h := NewPolicyHandler(svc)
	// first status param is empty → falls into the else-if multi-status branch
	r := withTenant(httptest.NewRequest(http.MethodGet, "/policies?status=&status=active&status=inactive", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestPolicyHandler_Create_BadJSON(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(badJSONReq(t, http.MethodPost, "/policies"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_Create_ValidationError(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(jsonReq(t, http.MethodPost, "/policies", map[string]any{}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_Create_Success(t *testing.T) {
	svc := &mockPolicyService{
		createFn: func(tid int64, name string, desc *string, doc datatypes.JSON, ver, status string, isSys bool) (*service.PolicyServiceDataResult, error) {
			return &service.PolicyServiceDataResult{Name: name}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/policies", validPolicyBody()))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ── Update ───────────────────────────────────────────────────────────────────

func TestPolicyHandler_Update_NoTenant(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withChiParam(httptest.NewRequest(http.MethodPut, "/policies/"+testResourceUUID.String(), nil), "policy_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHandler_Update_InvalidUUID(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPut, "/policies/bad", nil), "policy_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_Update_BadJSON(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPut, "/policies/"+testResourceUUID.String()), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_Update_ValidationError(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/policies/"+testResourceUUID.String(), map[string]any{}), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_Update_ServiceError(t *testing.T) {
	svc := &mockPolicyService{
		updateFn: func(id uuid.UUID, tid int64, name string, desc *string, doc datatypes.JSON, ver, status string) (*service.PolicyServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/policies/"+testResourceUUID.String(), validPolicyBody()), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPolicyHandler_Update_Success(t *testing.T) {
	svc := &mockPolicyService{
		updateFn: func(id uuid.UUID, tid int64, name string, desc *string, doc datatypes.JSON, ver, status string) (*service.PolicyServiceDataResult, error) {
			return &service.PolicyServiceDataResult{Name: name}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/policies/"+testResourceUUID.String(), validPolicyBody()), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── UpdateStatus ─────────────────────────────────────────────────────────────

func TestPolicyHandler_UpdateStatus_NoTenant(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withChiParam(httptest.NewRequest(http.MethodPatch, "/policies/"+testResourceUUID.String()+"/status", nil), "policy_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHandler_UpdateStatus_InvalidUUID(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodPatch, "/policies/bad/status", nil), "policy_uuid", "bad"))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_UpdateStatus_BadJSON(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPatch, "/policies/"+testResourceUUID.String()+"/status"), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_UpdateStatus_ValidationError(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/policies/"+testResourceUUID.String()+"/status", map[string]any{"status": "invalid_status"}), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_UpdateStatus_ServiceError(t *testing.T) {
	svc := &mockPolicyService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, status string) (*service.PolicyServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/policies/"+testResourceUUID.String()+"/status", map[string]any{"status": "active"}), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPolicyHandler_UpdateStatus_Success(t *testing.T) {
	svc := &mockPolicyService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, status string) (*service.PolicyServiceDataResult, error) {
			return &service.PolicyServiceDataResult{}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/policies/"+testResourceUUID.String()+"/status", map[string]any{"status": "active"}), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Delete ───────────────────────────────────────────────────────────────────

func TestPolicyHandler_Delete_NoTenant(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withChiParam(httptest.NewRequest(http.MethodDelete, "/policies/"+testResourceUUID.String(), nil), "policy_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPolicyHandler_Delete_Success(t *testing.T) {
	svc := &mockPolicyService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.PolicyServiceDataResult, error) {
			return &service.PolicyServiceDataResult{}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/policies/"+testResourceUUID.String(), nil), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GetServicesByPolicyUUID ───────────────────────────────────────────────────

func TestPolicyHandler_GetServicesByPolicyUUID_InvalidUUID(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/policies/bad/services", nil), "policy_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetServicesByPolicyUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_GetServicesByPolicyUUID_ValidationError(t *testing.T) {
	h := NewPolicyHandler(&mockPolicyService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/policies/"+testResourceUUID.String()+"/services?sort_order=bad", nil), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetServicesByPolicyUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPolicyHandler_GetServicesByPolicyUUID_ServiceError(t *testing.T) {
	svc := &mockPolicyService{
		getServicesByPolicyUUIDFn: func(id uuid.UUID, tid int64, f service.PolicyServiceServicesFilter) (*service.PolicyServiceServicesResult, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/policies/"+testResourceUUID.String()+"/services", nil), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetServicesByPolicyUUID(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPolicyHandler_GetServicesByPolicyUUID_Success(t *testing.T) {
	svc := &mockPolicyService{
		getServicesByPolicyUUIDFn: func(id uuid.UUID, tid int64, f service.PolicyServiceServicesFilter) (*service.PolicyServiceServicesResult, error) {
			return &service.PolicyServiceServicesResult{
				Data: []service.PolicyServiceServiceDataResult{{ServiceUUID: testResourceUUID, Name: "svc1"}},
			}, nil
		},
	}
	h := NewPolicyHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/policies/"+testResourceUUID.String()+"/services?name=svc&display_name=d&description=desc", nil), "policy_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetServicesByPolicyUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
