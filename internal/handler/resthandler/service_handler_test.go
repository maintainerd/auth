package resthandler

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
			return nil, assert.AnError
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
