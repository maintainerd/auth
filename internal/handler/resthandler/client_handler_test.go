package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestClientHandler_Get_NoTenant(t *testing.T) {
	h := NewClientHandler(&mockClientService{})
	r := httptest.NewRequest(http.MethodGet, "/clients", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClientHandler_Get_ServiceError(t *testing.T) {
	svc := &mockClientService{
		getFn: func(service.ClientServiceGetFilter) (*service.ClientServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/clients?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestClientHandler_Get_Success(t *testing.T) {
	svc := &mockClientService{
		getFn: func(service.ClientServiceGetFilter) (*service.ClientServiceGetResult, error) {
			return &service.ClientServiceGetResult{}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/clients?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientHandler_GetByUUID_NoTenant(t *testing.T) {
	h := NewClientHandler(&mockClientService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClientHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewClientHandler(&mockClientService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/clients/bad", nil), "client_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockClientService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestClientHandler_GetByUUID_Success(t *testing.T) {
	svc := &mockClientService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{Name: "client1"}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewClientHandler(&mockClientService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/clients/bad", nil), "client_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockClientService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewClientHandler(svc)
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestClientHandler_Delete_Success(t *testing.T) {
	svc := &mockClientService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{Name: "c1"}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
