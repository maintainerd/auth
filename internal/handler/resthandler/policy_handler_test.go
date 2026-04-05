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
