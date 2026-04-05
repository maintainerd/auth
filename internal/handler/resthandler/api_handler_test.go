package resthandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestAPIHandler_Get_NoTenant(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIHandler_Get_ServiceError(t *testing.T) {
	svc := &mockAPIService{
		getFn: func(service.APIServiceGetFilter) (*service.APIServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/apis?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_Get_Success(t *testing.T) {
	svc := &mockAPIService{
		getFn: func(service.APIServiceGetFilter) (*service.APIServiceGetResult, error) {
			return &service.APIServiceGetResult{}, nil
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/apis?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestAPIHandler_GetByUUID_NoTenant(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withChiParam(httptest.NewRequest(http.MethodGet, "/apis/"+testResourceUUID.String(), nil), "api_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/apis/bad", nil), "api_uuid", "bad"))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockAPIService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.APIServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/apis/"+testResourceUUID.String(), nil), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIHandler_GetByUUID_Success(t *testing.T) {
	svc := &mockAPIService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.APIServiceDataResult, error) {
			return &service.APIServiceDataResult{Name: "test-api"}, nil
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/apis/"+testResourceUUID.String(), nil), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAPIHandler_Create_NoTenant(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := jsonReq(t, http.MethodPost, "/apis", map[string]string{"name": "api1"})
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIHandler_Create_ServiceError(t *testing.T) {
	svc := &mockAPIService{
		createFn: func(tid int64, n, dn, desc, tp, s string, isSys bool, svcUUID string) (*service.APIServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/apis", map[string]string{
		"name": "api1", "display_name": "API One", "description": "A test API for testing",
		"api_type": "rest", "status": "active", "service_id": testResourceUUID.String(),
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_Create_Success(t *testing.T) {
	svc := &mockAPIService{
		createFn: func(tid int64, n, dn, desc, tp, s string, isSys bool, svcUUID string) (*service.APIServiceDataResult, error) {
			return &service.APIServiceDataResult{Name: n}, nil
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/apis", map[string]string{
		"name": "api1", "display_name": "API One", "description": "A test API for testing",
		"api_type": "rest", "status": "active", "service_id": testResourceUUID.String(),
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// Update / SetStatus / Delete
// ---------------------------------------------------------------------------

func TestAPIHandler_Update_InvalidUUID(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/apis/bad", map[string]string{"name": "n"}), "api_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_SetStatus_InvalidUUID(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/apis/bad/status", map[string]string{"status": "active"}), "api_uuid", "bad"))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/apis/bad", nil), "api_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockAPIService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.APIServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/apis/"+testResourceUUID.String(), nil), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_Delete_Success(t *testing.T) {
	svc := &mockAPIService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.APIServiceDataResult, error) {
			return &service.APIServiceDataResult{Name: "api1"}, nil
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/apis/"+testResourceUUID.String(), nil), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
