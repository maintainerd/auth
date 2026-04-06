package rest

import (
	"bytes"
	"errors"
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

func TestPermissionHandler_Get_NoTenant(t *testing.T) {
	h := NewPermissionHandler(&mockPermissionService{})
	r := httptest.NewRequest(http.MethodGet, "/permissions", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPermissionHandler_Get_ValidationError(t *testing.T) {
	h := NewPermissionHandler(&mockPermissionService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/permissions?sort_order=invalid", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Get_WithFiltersAndRows(t *testing.T) {
	// Covers is_default/is_system bool parse branches, row loop, and toPermissionResponseDto API branch.
	apiUUID := uuid.New()
	svc := &mockPermissionService{
		getFn: func(service.PermissionServiceGetFilter) (*service.PermissionServiceGetResult, error) {
			return &service.PermissionServiceGetResult{
				Data: []service.PermissionServiceDataResult{{
					Name: "perm1",
					API:  &service.APIServiceDataResult{APIUUID: apiUUID, Name: "api1"},
				}},
			}, nil
		},
	}
	r := withTenant(httptest.NewRequest(http.MethodGet, "/permissions?page=1&limit=10&is_default=true&is_system=false", nil))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPermissionHandler_Get_ServiceError(t *testing.T) {
	svc := &mockPermissionService{
		getFn: func(service.PermissionServiceGetFilter) (*service.PermissionServiceGetResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPermissionHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/permissions?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPermissionHandler_Get_Success(t *testing.T) {
	svc := &mockPermissionService{
		getFn: func(service.PermissionServiceGetFilter) (*service.PermissionServiceGetResult, error) {
			return &service.PermissionServiceGetResult{}, nil
		},
	}
	h := NewPermissionHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/permissions?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestPermissionHandler_GetByUUID_NoTenant(t *testing.T) {
	h := NewPermissionHandler(&mockPermissionService{})
	r := httptest.NewRequest(http.MethodGet, "/permissions/"+testResourceUUID.String(), nil)
	r = withChiParam(r, "permission_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPermissionHandler_GetByUUID_InvalidUUID(t *testing.T) {
	h := NewPermissionHandler(&mockPermissionService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/permissions/bad-uuid", nil))
	r = withChiParam(r, "permission_uuid", "bad-uuid")
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_GetByUUID_Success(t *testing.T) {
	// Also covers toPermissionResponseDto with API != nil branch (lines 291-303).
	apiUUID := uuid.New()
	svc := &mockPermissionService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.PermissionServiceDataResult, error) {
			return &service.PermissionServiceDataResult{
				Name: "perm1",
				API:  &service.APIServiceDataResult{APIUUID: apiUUID, Name: "api1"},
			}, nil
		},
	}
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPermissionHandler_GetByUUID_NotFound(t *testing.T) {
	svc := &mockPermissionService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.PermissionServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPermissionHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/permissions/"+testResourceUUID.String(), nil))
	r = withChiParam(r, "permission_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestPermissionHandler_Create_NoTenant(t *testing.T) {
	h := NewPermissionHandler(&mockPermissionService{})
	r := jsonReq(t, http.MethodPost, "/permissions", map[string]string{"name": "perm1"})
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPermissionHandler_Create_InvalidBody(t *testing.T) {
	h := NewPermissionHandler(&mockPermissionService{})
	r := withTenant(httptest.NewRequest(http.MethodPost, "/permissions", bytes.NewBufferString(`{bad}`)))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Create_ValidationError(t *testing.T) {
	h := NewPermissionHandler(&mockPermissionService{})
	r := withTenant(jsonReq(t, http.MethodPost, "/permissions", map[string]any{}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Create_ServiceError(t *testing.T) {
	svc := &mockPermissionService{
		createFn: func(tid int64, n, desc, status string, isSystem bool, apiUUID string) (*service.PermissionServiceDataResult, error) {
			return nil, assert.AnError
		},
	}
	h := NewPermissionHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/permissions", map[string]string{
		"name": "perm1", "description": "A test description", "status": "active", "api_id": testResourceUUID.String(),
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPermissionHandler_Create_Success(t *testing.T) {
	svc := &mockPermissionService{
		createFn: func(tid int64, n, desc, status string, isSystem bool, apiUUID string) (*service.PermissionServiceDataResult, error) {
			return &service.PermissionServiceDataResult{Name: n}, nil
		},
	}
	h := NewPermissionHandler(svc)
	r := withTenant(jsonReq(t, http.MethodPost, "/permissions", map[string]string{
		"name": "perm1", "description": "A test description", "status": "active", "api_id": testResourceUUID.String(),
	}))
	w := httptest.NewRecorder()
	h.Create(w, r)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestPermissionHandler_Update_NoTenant(t *testing.T) {
	r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"}), "permission_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).Update(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPermissionHandler_Update_InvalidUUID(t *testing.T) {
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"}), "permission_uuid", "bad"))
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Update_BadJSON(t *testing.T) {
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPut, "/"), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Update_ValidationError(t *testing.T) {
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{}), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).Update(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Update_ServiceError(t *testing.T) {
	svc := &mockPermissionService{
		updateFn: func(id uuid.UUID, tid int64, n, desc, status string) (*service.PermissionServiceDataResult, error) {
			return nil, errors.New("db error")
		},
	}
	body := map[string]any{"name": "perm1", "description": "A valid description", "status": "active"}
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", body), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).Update(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPermissionHandler_Update_Success(t *testing.T) {
	svc := &mockPermissionService{
		updateFn: func(id uuid.UUID, tid int64, n, desc, status string) (*service.PermissionServiceDataResult, error) {
			return &service.PermissionServiceDataResult{Name: n}, nil
		},
	}
	body := map[string]any{"name": "perm1", "description": "A valid description", "status": "active"}
	r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", body), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).Update(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// SetStatus
// ---------------------------------------------------------------------------

func TestPermissionHandler_SetStatus_NoTenant(t *testing.T) {
	r := withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "permission_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).SetStatus(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPermissionHandler_SetStatus_InvalidUUID(t *testing.T) {
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "permission_uuid", "bad"))
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_SetStatus_BadJSON(t *testing.T) {
	r := withTenant(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_SetStatus_ValidationError(t *testing.T) {
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"}), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).SetStatus(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_SetStatus_ServiceError(t *testing.T) {
	svc := &mockPermissionService{
		setStatusFn: func(id uuid.UUID, tid int64, s string) (*service.PermissionServiceDataResult, error) {
			return nil, errors.New("db error")
		},
	}
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).SetStatus(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPermissionHandler_SetStatus_Success(t *testing.T) {
	svc := &mockPermissionService{
		setStatusFn: func(id uuid.UUID, tid int64, s string) (*service.PermissionServiceDataResult, error) {
			return &service.PermissionServiceDataResult{Name: "perm1"}, nil
		},
	}
	r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).SetStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestPermissionHandler_Delete_NoTenant(t *testing.T) {
	r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "permission_uuid", testResourceUUID.String())
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPermissionHandler_Delete_InvalidUUID(t *testing.T) {
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "permission_uuid", "bad"))
	w := httptest.NewRecorder()
	NewPermissionHandler(&mockPermissionService{}).Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPermissionHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockPermissionService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.PermissionServiceDataResult, error) {
			return nil, errors.New("db error")
		},
	}
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).Delete(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPermissionHandler_Delete_Success(t *testing.T) {
	svc := &mockPermissionService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64) (*service.PermissionServiceDataResult, error) {
			return &service.PermissionServiceDataResult{Name: "perm1"}, nil
		},
	}
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "permission_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	NewPermissionHandler(svc).Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}
