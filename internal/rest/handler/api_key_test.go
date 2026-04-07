package handler

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
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/api-keys", nil)
		r = withUser(r) // user but no tenant
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		// invalid status value triggers APIKeyGetRequestDTO.Validate failure
		r := jsonReq(t, http.MethodGet, "/api-keys?status=invalid_status", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("with optional filters", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getFn: func(f service.APIKeyServiceGetFilter, u uuid.UUID) (*service.APIKeyServiceGetResult, error) {
				return &service.APIKeyServiceGetResult{Data: []service.APIKeyServiceDataResult{{Name: "k1"}}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/api-keys?name=foo&description=bar&status=active", nil)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

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

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

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
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/api-keys", map[string]any{"name": "mykey"})
		r = withUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/api-keys")
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		// empty name fails validation
		r := jsonReq(t, http.MethodPost, "/api-keys", map[string]any{"name": ""})
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("custom status is passed to service", func(t *testing.T) {
		svc := &mockAPIKeyService{
			createFn: func(tid int64, n, desc string, cfg datatypes.JSON, exp *time.Time, rl *int, s string) (*service.APIKeyServiceDataResult, string, error) {
				return &service.APIKeyServiceDataResult{APIKeyUUID: uuid.New(), Name: n, Status: s}, "plainkey", nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/api-keys", map[string]any{"name": "mykey", "status": "inactive"})
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
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

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"})
		r = withUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"name": "n"})
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPut, "/")
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		// invalid status value triggers APIKeyUpdateRequestDTO.Validate failure
		invalidStatus := "bad_status"
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"status": invalidStatus})
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
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

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPatch, "/")
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid_status"})
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string) (*service.APIKeyServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenant(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
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

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			deleteFn: func(id uuid.UUID, tid int64, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
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
	apiUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		// invalid sort_order triggers PaginationRequestDTO.Validate failure
		r := jsonReq(t, http.MethodGet, "/?sort_order=invalid", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getAPIKeyAPIsFn: func(id uuid.UUID, pg, lim int, sb, so string) (*service.APIKeyAPIServicePaginatedResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetAPIs(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with rows", func(t *testing.T) {
		// Covers the loop body in GetAPIs that maps rows to APIResponseDTO
		svc := &mockAPIKeyService{
			getAPIKeyAPIsFn: func(id uuid.UUID, pg, lim int, sb, so string) (*service.APIKeyAPIServicePaginatedResult, error) {
				return &service.APIKeyAPIServicePaginatedResult{
					Data: []service.APIKeyAPIServiceDataResult{
						{Api: service.APIServiceDataResult{APIUUID: apiUUID, Name: "api1"}},
					},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetAPIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success empty", func(t *testing.T) {
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

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{apiUUID.String()}})
		r = withChiParam(r, "api_key_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/")
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		// empty api_uuids fails validation
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{}})
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			addAPIKeyAPIsFn: func(id uuid.UUID, apis []uuid.UUID) error {
				return errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{apiUUID.String()}})
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).AddAPIs(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

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

	t.Run("invalid api_key_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", "bad")
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).RemoveAPI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid api_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).RemoveAPI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			removeAPIKeyAPIFn: func(id, api uuid.UUID) error {
				return errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).RemoveAPI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

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

	t.Run("invalid api_key_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", "bad")
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid api_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getAPIKeyAPIPermsFn: func(id, api uuid.UUID) ([]service.PermissionServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockAPIKeyService{
			getAPIKeyAPIPermsFn: func(id, api uuid.UUID) ([]service.PermissionServiceDataResult, error) {
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

	t.Run("invalid api_key_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}})
		r = withChiParam(r, "api_key_uuid", "bad")
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid api_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}})
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := badJSONReq(t, http.MethodPost, "/")
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{}})
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			addAPIKeyAPIPermsFn: func(id, api uuid.UUID, perms []uuid.UUID) error {
				return errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}})
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

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

	t.Run("invalid api_key_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", "bad")
		r = withChiParam(r, "api_uuid", apiUUID.String())
		r = withChiParam(r, "permission_uuid", permUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid api_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", "bad")
		r = withChiParam(r, "permission_uuid", permUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid permission_uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		r = withChiParam(r, "permission_uuid", "bad")
		w := httptest.NewRecorder()
		NewAPIKeyHandler(&mockAPIKeyService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockAPIKeyService{
			removeAPIKeyAPIPermFn: func(id, api, perm uuid.UUID) error {
				return errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withChiParam(r, "api_key_uuid", keyUUID.String())
		r = withChiParam(r, "api_uuid", apiUUID.String())
		r = withChiParam(r, "permission_uuid", permUUID.String())
		w := httptest.NewRecorder()
		NewAPIKeyHandler(svc).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

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
