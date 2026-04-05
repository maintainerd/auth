package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestTenantHandler_Get_ServiceError(t *testing.T) {
	svc := &mockTenantService{
		getFn: func(service.TenantServiceGetFilter) (*service.TenantServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantHandler(svc, &mockTenantMemberService{})
	r := httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantHandler_Get_Success(t *testing.T) {
	svc := &mockTenantService{
		getFn: func(service.TenantServiceGetFilter) (*service.TenantServiceGetResult, error) {
			return &service.TenantServiceGetResult{}, nil
		},
	}
	h := NewTenantHandler(svc, &mockTenantMemberService{})
	r := httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewTenantHandler(&mockTenantService{}, &mockTenantMemberService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/tenants/bad", nil), "tenant_uuid", "bad")
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockTenantService{
		getByUUIDFn: func(id uuid.UUID) (*service.TenantServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantHandler(svc, &mockTenantMemberService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/tenants/"+testResourceUUID.String(), nil), "tenant_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenantHandler_GetByUUID_Success(t *testing.T) {
	svc := &mockTenantService{
		getByUUIDFn: func(id uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: "tenant1"}, nil
		},
	}
	h := NewTenantHandler(svc, &mockTenantMemberService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/tenants/"+testResourceUUID.String(), nil), "tenant_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantHandler_Create_BadJSON(t *testing.T) {
	h := NewTenantHandler(&mockTenantService{}, &mockTenantMemberService{})
	r := httptest.NewRequest(http.MethodPost, "/tenants", nil)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantHandler_Create_ServiceError(t *testing.T) {
	svc := &mockTenantService{
		createFn: func(n, dn, desc, s string, isPublic, isDef bool) (*service.TenantServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewTenantHandler(svc, &mockTenantMemberService{})
	r := jsonReq(t, http.MethodPost, "/tenants", map[string]interface{}{
		"name": "my-tenant", "display_name": "My Tenant", "description": "Tenant description", "status": "active",
	})
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTenantHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewTenantHandler(&mockTenantService{}, &mockTenantMemberService{})
	r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/tenants/bad", nil), "tenant_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantHandler_Delete_Success(t *testing.T) {
	svc := &mockTenantService{
		getByUUIDFn: func(id uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: "t1", IsSystem: false}, nil
		},
		deleteByUUIDFn: func(id uuid.UUID) (*service.TenantServiceDataResult, error) {
			return &service.TenantServiceDataResult{Name: "t1"}, nil
		},
	}
	memberSvc := &mockTenantMemberService{
		isUserInTenantFn: func(userID int64, tenantUUID uuid.UUID) (bool, error) { return true, nil },
	}
	h := NewTenantHandler(svc, memberSvc)
	r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/tenants/"+testResourceUUID.String(), nil), "tenant_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
