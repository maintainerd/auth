package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

// ---------------------------------------------------------------------------
// GetUserPools
// ---------------------------------------------------------------------------

func TestUserPoolHandler_GetUserPools(t *testing.T) {
	t.Run("no tenant in context", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := httptest.NewRequest(http.MethodGet, "/user-pools", nil)
		w := httptest.NewRecorder()
		h.GetUserPools(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			listFn: func(int64) ([]*UserPoolServiceDataResult, error) { return nil, errNotFound },
		})
		r := withTenant(httptest.NewRequest(http.MethodGet, "/user-pools", nil))
		w := httptest.NewRecorder()
		h.GetUserPools(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			listFn: func(int64) ([]*UserPoolServiceDataResult, error) {
				return []*UserPoolServiceDataResult{{UserPoolUUID: uuid.New(), Name: "a"}}, nil
			},
		})
		r := withTenant(httptest.NewRequest(http.MethodGet, "/user-pools", nil))
		w := httptest.NewRecorder()
		h.GetUserPools(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// GetUserPool
// ---------------------------------------------------------------------------

func TestUserPoolHandler_GetUserPool(t *testing.T) {
	t.Run("no tenant in context", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/user-pools/x", nil), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.GetUserPool(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenant(httptest.NewRequest(http.MethodGet, "/user-pools/x", nil)), "user_pool_uuid", "not-a-uuid")
		w := httptest.NewRecorder()
		h.GetUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			getByUUIDFn: func(uuid.UUID, int64) (*UserPoolServiceDataResult, error) { return nil, errNotFound },
		})
		r := withChiParam(withTenant(httptest.NewRequest(http.MethodGet, "/user-pools/x", nil)), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.GetUserPool(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		h := NewUserPoolHandler(&mockUserPoolService{
			getByUUIDFn: func(uuid.UUID, int64) (*UserPoolServiceDataResult, error) {
				return &UserPoolServiceDataResult{UserPoolUUID: id, Name: "pool"}, nil
			},
		})
		r := withChiParam(withTenant(httptest.NewRequest(http.MethodGet, "/user-pools/x", nil)), "user_pool_uuid", id.String())
		w := httptest.NewRecorder()
		h.GetUserPool(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// CreateUserPool
// ---------------------------------------------------------------------------

func TestUserPoolHandler_CreateUserPool(t *testing.T) {
	t.Run("no tenant in context", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := jsonReq(t, http.MethodPost, "/user-pools", UserPoolCreateRequestDTO{Name: "Customers"})
		w := httptest.NewRecorder()
		h.CreateUserPool(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("bad json", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/user-pools"))
		w := httptest.NewRecorder()
		h.CreateUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-pools", UserPoolCreateRequestDTO{Name: ""}))
		w := httptest.NewRecorder()
		h.CreateUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			createFn: func(int64, string, string, string, datatypes.JSON, *int64) (*UserPoolServiceDataResult, error) {
				return nil, errValidation
			},
		})
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-pools", UserPoolCreateRequestDTO{Name: "Customers"}))
		w := httptest.NewRecorder()
		h.CreateUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success forwards creator id", func(t *testing.T) {
		var gotCreator *int64
		h := NewUserPoolHandler(&mockUserPoolService{
			createFn: func(_ int64, name, _, _ string, _ datatypes.JSON, creator *int64) (*UserPoolServiceDataResult, error) {
				gotCreator = creator
				return &UserPoolServiceDataResult{Name: name}, nil
			},
		})
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/user-pools", UserPoolCreateRequestDTO{Name: "Customers"}))
		w := httptest.NewRecorder()
		h.CreateUserPool(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
		// creatorUserID is sourced from the authenticated user in context.
		assert.NotNil(t, gotCreator)
	})
}

// ---------------------------------------------------------------------------
// UpdateUserPool
// ---------------------------------------------------------------------------

func TestUserPoolHandler_UpdateUserPool(t *testing.T) {
	valid := UserPoolUpdateRequestDTO{Name: "Customers", Status: "active"}

	t.Run("no tenant in context", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(jsonReq(t, http.MethodPut, "/user-pools/x", valid), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.UpdateUserPool(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/user-pools/x", valid)), "user_pool_uuid", "bad")
		w := httptest.NewRecorder()
		h.UpdateUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenantAndUser(badJSONReq(t, http.MethodPut, "/user-pools/x")), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.UpdateUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/user-pools/x", UserPoolUpdateRequestDTO{Name: "Customers"})), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.UpdateUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			updateFn: func(uuid.UUID, int64, string, string, string, datatypes.JSON, *int64) (*UserPoolServiceDataResult, error) {
				return nil, errNotFound
			},
		})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/user-pools/x", valid)), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.UpdateUserPool(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			updateFn: func(_ uuid.UUID, _ int64, name, _, _ string, _ datatypes.JSON, _ *int64) (*UserPoolServiceDataResult, error) {
				return &UserPoolServiceDataResult{Name: name}, nil
			},
		})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPut, "/user-pools/x", valid)), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.UpdateUserPool(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// SetStatus
// ---------------------------------------------------------------------------

func TestUserPoolHandler_SetStatus(t *testing.T) {
	valid := UserPoolSetStatusRequestDTO{Status: "inactive"}

	t.Run("no tenant in context", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(jsonReq(t, http.MethodPatch, "/user-pools/x/status", valid), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPatch, "/user-pools/x/status", valid)), "user_pool_uuid", "bad")
		w := httptest.NewRecorder()
		h.SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenantAndUser(badJSONReq(t, http.MethodPatch, "/user-pools/x/status")), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPatch, "/user-pools/x/status", UserPoolSetStatusRequestDTO{Status: "bogus"})), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			setStatusFn: func(uuid.UUID, int64, string, *int64) (*UserPoolServiceDataResult, error) {
				return nil, errConflict
			},
		})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPatch, "/user-pools/x/status", valid)), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.SetStatus(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		var gotStatus string
		var gotUpdater *int64
		h := NewUserPoolHandler(&mockUserPoolService{
			setStatusFn: func(_ uuid.UUID, _ int64, status string, updater *int64) (*UserPoolServiceDataResult, error) {
				gotStatus, gotUpdater = status, updater
				return &UserPoolServiceDataResult{Status: status}, nil
			},
		})
		r := withChiParam(withTenantAndUser(jsonReq(t, http.MethodPatch, "/user-pools/x/status", valid)), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.SetStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "inactive", gotStatus)
		assert.NotNil(t, gotUpdater) // acting user forwarded for audit
	})
}

// ---------------------------------------------------------------------------
// DeleteUserPool
// ---------------------------------------------------------------------------

func TestUserPoolHandler_DeleteUserPool(t *testing.T) {
	t.Run("no tenant in context", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/user-pools/x", nil), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.DeleteUserPool(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{})
		r := withChiParam(withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/user-pools/x", nil)), "user_pool_uuid", "bad")
		w := httptest.NewRecorder()
		h.DeleteUserPool(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error (conflict)", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			deleteFn: func(uuid.UUID, int64) (*UserPoolServiceDataResult, error) {
				return nil, errConflict
			},
		})
		r := withChiParam(withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/user-pools/x", nil)), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.DeleteUserPool(w, r)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		h := NewUserPoolHandler(&mockUserPoolService{
			deleteFn: func(uuid.UUID, int64) (*UserPoolServiceDataResult, error) {
				return &UserPoolServiceDataResult{Name: "gone"}, nil
			},
		})
		r := withChiParam(withTenantAndUser(httptest.NewRequest(http.MethodDelete, "/user-pools/x", nil)), "user_pool_uuid", uuid.New().String())
		w := httptest.NewRecorder()
		h.DeleteUserPool(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
