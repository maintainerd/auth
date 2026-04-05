package resthandler

import (
	"bytes"
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
