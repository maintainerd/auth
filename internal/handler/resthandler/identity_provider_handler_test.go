package resthandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestIdentityProviderHandler_Get(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/idps", nil)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/idps?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			getFn: func(f service.IdentityProviderServiceGetFilter) (*service.IdentityProviderServiceGetResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodGet, "/idps?page=1&limit=10", nil)
		r = withTenant(r)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestIdentityProviderHandler_GetByUUID(t *testing.T) {
	idpUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "identity_provider_uuid", "bad")
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*service.IdentityProviderServiceDataResult, error) {
				return nil, errors.New("not found")
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*service.IdentityProviderServiceDataResult, error) {
				return &service.IdentityProviderServiceDataResult{IdentityProviderUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestIdentityProviderHandler_Create(t *testing.T) {
	validBody := map[string]any{
		"name":          "test-idp",
		"display_name":  "Test Identity Provider",
		"provider":      "internal",
		"provider_type": "identity",
		"status":        "active",
		"config":        map[string]any{},
		"tenant_id":     testTenantUUID.String(),
	}

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/idps", map[string]any{"name": ""})
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			createFn: func(name, display, provider, providerType string, config datatypes.JSON, status, tUUID string, tid int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodPost, "/idps", validBody)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			createFn: func(name, display, provider, providerType string, config datatypes.JSON, status, tUUID string, tid int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
				return &service.IdentityProviderServiceDataResult{Name: name}, nil
			},
		}
		r := jsonReq(t, http.MethodPost, "/idps", validBody)
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestIdentityProviderHandler_Update(t *testing.T) {
	idpUUID := uuid.New()
	validBody := map[string]any{
		"name":          "upd",
		"display_name":  "Updated Provider",
		"provider":      "internal",
		"provider_type": "identity",
		"status":        "active",
		"config":        map[string]any{},
		"tenant_id":     testTenantUUID.String(),
	}

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", "bad")
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			updateFn: func(id uuid.UUID, name, display, provider, providerType string, config datatypes.JSON, status string, tid int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			updateFn: func(id uuid.UUID, name, display, provider, providerType string, config datatypes.JSON, status string, tid int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
				return &service.IdentityProviderServiceDataResult{IdentityProviderUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestIdentityProviderHandler_SetStatus(t *testing.T) {
	idpUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			setStatusByUUIDFn: func(id uuid.UUID, status string, tid int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
				return &service.IdentityProviderServiceDataResult{IdentityProviderUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"})
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"})
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestIdentityProviderHandler_Delete(t *testing.T) {
	idpUUID := uuid.New()

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", "bad")
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
				return &service.IdentityProviderServiceDataResult{IdentityProviderUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
