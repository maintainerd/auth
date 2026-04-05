package resthandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestAPIKeyHandler_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getFn: func(f service.APIKeyServiceGetFilter, u uuid.UUID) (*service.APIKeyServiceGetResult, error) {
				return &service.APIKeyServiceGetResult{Data: []service.APIKeyServiceDataResult{{Name: "k1"}}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/api-keys", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getFn: func(f service.APIKeyServiceGetFilter, u uuid.UUID) (*service.APIKeyServiceGetResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/api-keys", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAPIKeyHandler_GetByUUID(t *testing.T) {
	keyUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getByUUIDFn: func(id uuid.UUID, tid int64, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
				return &service.APIKeyServiceDataResult{APIKeyUUID: id, Name: "key1"}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", "bad-uuid")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getByUUIDFn: func(id uuid.UUID, tid int64, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
				return nil, errors.New("not found")
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAPIKeyHandler_GetConfigByUUID(t *testing.T) {
	keyUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getConfigByUUIDFn: func(id uuid.UUID, tid int64) (datatypes.JSON, error) {
				return datatypes.JSON(`{"rate":100}`), nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getConfigByUUIDFn: func(id uuid.UUID, tid int64) (datatypes.JSON, error) {
				return nil, errors.New("not found")
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAPIKeyHandler_Create(t *testing.T) {
	t.Run("validation error returns 400", func(t *testing.T) {
		// empty name fails validation
		r := jsonReq(t, http.MethodPost, "/api-keys", map[string]any{"name": ""})
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			createFn: func(tid int64, n, desc string, cfg datatypes.JSON, exp *time.Time, rl *int, s string) (*service.APIKeyServiceDataResult, string, error) {
				return nil, "", errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodPost, "/api-keys", map[string]any{"name": "mykey"})
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			createFn: func(tid int64, n, desc string, cfg datatypes.JSON, exp *time.Time, rl *int, s string) (*service.APIKeyServiceDataResult, string, error) {
				return &service.APIKeyServiceDataResult{APIKeyUUID: uuid.New(), Name: n}, "plainkey", nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/api-keys", map[string]any{"name": "mykey"})
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestAPIKeyHandler_Update(t *testing.T) {
	keyUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"})
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			updateFn: func(id uuid.UUID, tid int64, n, desc *string, cfg datatypes.JSON, exp *time.Time, rl *int, s *string, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"})
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			updateFn: func(id uuid.UUID, tid int64, n, desc *string, cfg datatypes.JSON, exp *time.Time, rl *int, s *string, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
				return &service.APIKeyServiceDataResult{APIKeyUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"})
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_SetStatus(t *testing.T) {
	keyUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", nil)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string) (*service.APIKeyServiceDataResult, error) {
				return &service.APIKeyServiceDataResult{APIKeyUUID: id, Status: s}, nil
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_Delete(t *testing.T) {
	keyUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			deleteFn: func(id uuid.UUID, tid int64, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
				return &service.APIKeyServiceDataResult{APIKeyUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_GetAPIs(t *testing.T) {
	keyUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetAPIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_AddAPIs(t *testing.T) {
	keyUUID := uuid.New()
	apiUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{apiUUID.String()}})
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_RemoveAPI(t *testing.T) {
	keyUUID := uuid.New()
	apiUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).RemoveAPI(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_GetAPIPermissions(t *testing.T) {
	keyUUID := uuid.New()
	apiUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getAPIKeyApiPermsFn: func(id, api uuid.UUID) ([]service.PermissionServiceDataResult, error) {
				return []service.PermissionServiceDataResult{{Name: "read"}}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_AddAPIPermissions(t *testing.T) {
	keyUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}})
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAPIKeyHandler_RemoveAPIPermission(t *testing.T) {
	keyUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		r = withChiParam(r, "permission_uuid", permUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
