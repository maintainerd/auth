package rest

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

func TestAPIHandler_Get_ValidationError(t *testing.T) {
	// page=0 and limit=0 fail PaginationRequestDto.Validate (min 1)
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/apis?page=0&limit=0", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestAPIHandler_Get_WithFilters(t *testing.T) {
	svc := &mockAPIService{
		getFn: func(f service.APIServiceGetFilter) (*service.APIServiceGetResult, error) {
			return &service.APIServiceGetResult{
				Data: []service.APIServiceDataResult{{Name: "api1"}},
			}, nil
		},
	}
	h := NewAPIHandler(svc)
	// covers status comma-split, is_system bool parsing, and the result rows loop
	r := withTenant(httptest.NewRequest(http.MethodGet, "/apis?page=1&limit=10&status=active,inactive&is_system=true", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIHandler_Get_InvalidServiceUUID(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/apis?page=1&limit=10&service_id=not-a-uuid", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Get_ServiceIDLookupError(t *testing.T) {
	svc := &mockAPIService{
		getServiceIDByUUIDFn: func(id uuid.UUID) (int64, error) {
			return 0, assert.AnError
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/apis?page=1&limit=10&service_id="+testResourceUUID.String(), nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Get_WithServiceIDFilter(t *testing.T) {
	// Covers the successful GetServiceIDByUUID path where serviceID is resolved and passed to Get.
	svc := &mockAPIService{
		getServiceIDByUUIDFn: func(id uuid.UUID) (int64, error) { return 42, nil },
		getFn: func(f service.APIServiceGetFilter) (*service.APIServiceGetResult, error) {
			return &service.APIServiceGetResult{}, nil
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/apis?page=1&limit=10&service_id="+testResourceUUID.String(), nil))
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
// GetByUUID — with Service populated (covers toAPIResponseDto Service branch)
// ---------------------------------------------------------------------------

func TestAPIHandler_GetByUUID_WithService(t *testing.T) {
	svcData := &service.APIServiceDataResult{
		Name:    "api1",
		Service: &service.ServiceServiceDataResult{Name: "svc1"},
	}
	svc := &mockAPIService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.APIServiceDataResult, error) {
			return svcData, nil
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

func TestAPIHandler_Create_BadJSON(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(badJSONReq(t, http.MethodPost, "/apis"))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Create_ValidationError(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	// name too short triggers validation failure
	r := withTenant(jsonReq(t, http.MethodPost, "/apis", map[string]string{"name": "a"}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
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
// Update
// ---------------------------------------------------------------------------

func TestAPIHandler_Update_NoTenant(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withChiParam(jsonReq(t, http.MethodPut, "/apis/"+testResourceUUID.String(), map[string]string{"name": "n"}), "api_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIHandler_Update_InvalidUUID(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/apis/bad", map[string]string{"name": "n"}), "api_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Update_BadJSON(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPut, "/apis/"+testResourceUUID.String()), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Update_ValidationError(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/apis/"+testResourceUUID.String(), map[string]string{"name": "a"}), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_Update_ServiceError(t *testing.T) {
	svc := &mockAPIService{
		updateFn: func(id uuid.UUID, tid int64, n, dn, desc, tp, s, svcUUID string) (*service.APIServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/apis/"+testResourceUUID.String(), map[string]string{
		"name": "api1", "display_name": "API One", "description": "A test API for testing",
		"api_type": "rest", "status": "active", "service_id": testResourceUUID.String(),
	}), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_Update_Success(t *testing.T) {
	svc := &mockAPIService{
		updateFn: func(id uuid.UUID, tid int64, n, dn, desc, tp, s, svcUUID string) (*service.APIServiceDataResult, error) {
			return &service.APIServiceDataResult{Name: n}, nil
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/apis/"+testResourceUUID.String(), map[string]string{
		"name": "api1", "display_name": "API One", "description": "A test API for testing",
		"api_type": "rest", "status": "active", "service_id": testResourceUUID.String(),
	}), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// SetStatus
// ---------------------------------------------------------------------------

func TestAPIHandler_SetStatus_NoTenant(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withChiParam(jsonReq(t, http.MethodPatch, "/apis/"+testResourceUUID.String()+"/status", map[string]string{"status": "active"}), "api_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIHandler_SetStatus_InvalidUUID(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/apis/bad/status", map[string]string{"status": "active"}), "api_uuid", "bad"))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_SetStatus_BadJSON(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPatch, "/apis/"+testResourceUUID.String()+"/status"), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_SetStatus_ValidationError(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/apis/"+testResourceUUID.String()+"/status", map[string]string{"status": "invalid"}), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIHandler_SetStatus_ServiceError(t *testing.T) {
	svc := &mockAPIService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, status string) (*service.APIServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/apis/"+testResourceUUID.String()+"/status", map[string]string{"status": "active"}), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIHandler_SetStatus_Success(t *testing.T) {
	svc := &mockAPIService{
		setStatusByUUIDFn: func(id uuid.UUID, tid int64, status string) (*service.APIServiceDataResult, error) {
			return &service.APIServiceDataResult{Name: "api1", Status: status}, nil
		},
	}
	h := NewAPIHandler(svc)
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/apis/"+testResourceUUID.String()+"/status", map[string]string{"status": "active"}), "api_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.SetStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAPIHandler_Delete_NoTenant(t *testing.T) {
	h := NewAPIHandler(&mockAPIService{})
	r := withChiParam(httptest.NewRequest(http.MethodDelete, "/apis/"+testResourceUUID.String(), nil), "api_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
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
