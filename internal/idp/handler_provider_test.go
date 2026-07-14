package idp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// captureAuditLogger is a test ManagementAuditLogger that records every entry it
// is asked to log so tests can assert on the Changes payload.
type captureAuditLogger struct {
	entries []auditlog.LogEntry
}

func (c *captureAuditLogger) Log(_ context.Context, entry auditlog.LogEntry) error {
	c.entries = append(c.entries, entry)
	return nil
}

func TestIdentityProviderHandler_Get(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/idps", nil)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		// invalid sort_order triggers IdentityProviderFilterDTO.Validate failure
		r := withTenant(jsonReq(t, http.MethodGet, "/idps?sort_order=invalid", nil))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("with filters and rows", func(t *testing.T) {
		// Covers is_default/is_system bool parse, status/provider array branches,
		// loop body (rows[i] = toIdpListResponseDTO), and toIdpListResponseDTO itself.
		svc := &mockIdentityProviderService{
			getFn: func(f IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
				return &IdentityProviderServiceGetResult{
					Data: []IdentityProviderServiceDataResult{{Name: "idp1"}},
				}, nil
			},
		}
		r := withTenant(jsonReq(t, http.MethodGet,
			"/idps?page=1&limit=10&is_default=true&is_system=false&status=active&provider=google", nil))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
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
			getFn: func(f IdentityProviderServiceGetFilter) (*IdentityProviderServiceGetResult, error) {
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

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodGet, "/", nil), "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).GetByUUID(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

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
			getByUUIDFn: func(id uuid.UUID, tid int64) (*IdentityProviderServiceDataResult, error) {
				return nil, errNotFound
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
			getByUUIDFn: func(id uuid.UUID, tid int64) (*IdentityProviderServiceDataResult, error) {
				return &IdentityProviderServiceDataResult{IdentityProviderUUID: id}, nil
			},
		}
		r := jsonReq(t, http.MethodGet, "/", nil)
		r = withTenant(r)
		r = withChiParam(r, "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success with tenant covers toIdpDetailResponseDTO tenant branch", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*IdentityProviderServiceDataResult, error) {
				return &IdentityProviderServiceDataResult{
					IdentityProviderUUID: id,
					Tenant: &TenantServiceDataResult{
						TenantUUID: testTenantUUID,
						Name:       "main",
					},
				}, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success with config covers toIdpDetailResponseDTO config branch", func(t *testing.T) {
		cfg := datatypes.JSON(json.RawMessage(`{}`))
		svc := &mockIdentityProviderService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*IdentityProviderServiceDataResult, error) {
				return &IdentityProviderServiceDataResult{
					IdentityProviderUUID: id,
					Config:               &cfg,
				}, nil
			},
		}
		r := withTenant(withChiParam(jsonReq(t, http.MethodGet, "/", nil), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestIdentityProviderHandler_Create(t *testing.T) {
	validBody := map[string]any{
		"name":          "test-idp",
		"display_name":  "Test Identity Provider",
		"provider":      "maintainerd",
		"provider_type": "system",
		"status":        "active",
		"config":        map[string]any{},
	}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/idps", validBody)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodPost, "/idps", validBody))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/idps"))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/idps", map[string]any{"name": ""})
		r = withTenantAndUser(r)
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			createFn: func(in IdentityProviderCreateInput) (*IdentityProviderServiceDataResult, error) {
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
			createFn: func(in IdentityProviderCreateInput) (*IdentityProviderServiceDataResult, error) {
				return &IdentityProviderServiceDataResult{Name: in.Name}, nil
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
		"name":          "upd-idp",
		"display_name":  "Updated Provider Name",
		"provider":      "maintainerd",
		"provider_type": "system",
		"status":        "active",
		"config":        map[string]any{},
	}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", "bad")
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPut, "/"), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{}), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			updateFn: func(in IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error) {
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
			updateFn: func(in IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error) {
				return &IdentityProviderServiceDataResult{IdentityProviderUUID: in.IdpUUID}, nil
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

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "identity_provider_uuid", "bad"))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "invalid"}), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			setStatusByUUIDFn: func(id uuid.UUID, status string, tid int64, actor uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			setStatusByUUIDFn: func(id uuid.UUID, status string, tid int64, actor uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return &IdentityProviderServiceDataResult{IdentityProviderUUID: id}, nil
			},
		}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestIdentityProviderHandler_Delete(t *testing.T) {
	idpUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "identity_provider_uuid", idpUUID.String())
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("no user returns 401", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodDelete, "/", nil), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodDelete, "/", nil)
		r = withTenantAndUser(r)
		r = withChiParam(r, "identity_provider_uuid", "bad")
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(&mockIdentityProviderService{}).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "identity_provider_uuid", idpUUID.String()))
		w := httptest.NewRecorder()
		NewIdentityProviderHandler(svc).Delete(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := &mockIdentityProviderService{
			deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*IdentityProviderServiceDataResult, error) {
				return &IdentityProviderServiceDataResult{IdentityProviderUUID: id}, nil
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

// TestIdentityProviderHandler_Update_RedactsSecretInAuditLog asserts that the
// plaintext client secret from the update request never lands in the audit
// "Changes" payload, while still being handed to the service for persistence.
func TestIdentityProviderHandler_Update_RedactsSecretInAuditLog(t *testing.T) {
	idpUUID := uuid.New()
	const secret = "super-secret-value"

	var gotSecret string
	svc := &mockIdentityProviderService{
		updateFn: func(in IdentityProviderUpdateInput) (*IdentityProviderServiceDataResult, error) {
			gotSecret = in.ProviderClientSecret
			return &IdentityProviderServiceDataResult{IdentityProviderUUID: in.IdpUUID}, nil
		},
	}

	logger := &captureAuditLogger{}
	h := NewIdentityProviderHandler(svc)
	h.SetAuditLogger(logger)

	body := map[string]any{
		"name":                   "upd-idp",
		"display_name":           "Updated Provider Name",
		"provider":               "maintainerd",
		"provider_type":          "system",
		"status":                 "active",
		"config":                 map[string]any{},
		"provider_client_secret": secret,
	}
	r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/", body), "identity_provider_uuid", idpUUID.String()))
	w := httptest.NewRecorder()
	h.Update(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	// The real secret is still persisted (the service receives it verbatim).
	assert.Equal(t, secret, gotSecret)
	// But it must never appear in the recorded audit payload.
	require.Len(t, logger.entries, 1)
	assert.NotContains(t, logger.entries[0].Changes, secret)
	assert.Contains(t, logger.entries[0].Changes, redactedSecretPlaceholder)
}

func TestRedactUpdateSecret(t *testing.T) {
	t.Run("masks non-empty secret", func(t *testing.T) {
		out := redactUpdateSecret(IdentityProviderUpdateRequestDTO{ProviderClientSecret: "plaintext"})
		assert.Equal(t, redactedSecretPlaceholder, out.ProviderClientSecret)
	})

	t.Run("leaves empty secret empty", func(t *testing.T) {
		out := redactUpdateSecret(IdentityProviderUpdateRequestDTO{ProviderClientSecret: ""})
		assert.Equal(t, "", out.ProviderClientSecret)
	})
}
