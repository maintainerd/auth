package client

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
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
		getFn: func(ClientServiceGetFilter) (*ClientServiceGetResult, error) {
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
		getFn: func(ClientServiceGetFilter) (*ClientServiceGetResult, error) {
			return &ClientServiceGetResult{}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/clients?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientHandler_Get_ValidationError(t *testing.T) {
	// invalid status value triggers ClientFilterDTO.Validate failure
	h := NewClientHandler(&mockClientService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/clients?status=bad_status", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientHandler_Get_WithFilters(t *testing.T) {
	// Covers is_default, is_system, status array trim, client_type array trim, and result rows loop
	svc := &mockClientService{
		getFn: func(ClientServiceGetFilter) (*ClientServiceGetResult, error) {
			return &ClientServiceGetResult{
				Data: []ClientServiceDataResult{{Name: "c1"}},
			}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet,
		"/clients?page=1&limit=10&is_default=true&is_system=false&status=active&client_type=traditional", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientHandler_GetByUUID_WithRelations(t *testing.T) {
	// Covers the IdentityProvider, ClientURIs, and Permissions branches in toClientResponseDTO
	uriUUID := uuid.New()
	uris := []ClientURIServiceDataResult{{ClientURIUUID: uriUUID, URI: "https://example.com", Type: "redirect_uri"}}
	perms := []PermissionServiceDataResult{{Name: "read"}}
	svc := &mockClientService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{
				Name:             "c1",
				IdentityProvider: &IdentityProviderServiceDataResult{Name: "idp1"},
				ClientURIs:       &uris,
				Permissions:      &perms,
			}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
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
		getByUUIDFn: func(id uuid.UUID, tid int64) (*ClientServiceDataResult, error) {
			return nil, errNotFound
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
		getByUUIDFn: func(id uuid.UUID, tid int64) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{Name: "client1"}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestClientHandler_GetSecretByUUID lived here and asserted an unconditional 410
// Gone. See TestClientRoute_HasNoSecretReadRoute in routes_test.go for why that
// behaviour was wrong and what replaced it.

func TestClientHandler_GetConfigByUUID(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockClientService{getConfigByUUIDFn: func(id uuid.UUID, tid int64) (datatypes.JSON, error) {
			return nil, errNotFound
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{getConfigByUUIDFn: func(id uuid.UUID, tid int64) (datatypes.JSON, error) {
			return datatypes.JSON(`{}`), nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetConfigByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func validClientBody() map[string]any {
	return map[string]any{
		"name":                 "myclient",
		"display_name":         "My Client Display Name",
		"client_type":          "traditional",
		"domain":               "example.com",
		"config":               map[string]any{"key": "value"},
		"status":               "active",
		"identity_provider_id": testResourceUUID.String(),
	}
}

func TestClientHandler_Create(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(jsonReq(t, http.MethodPost, "/clients", validClientBody()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/clients"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/clients", map[string]any{"name": "x"}))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{createFn: func(tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, idpUUID string, brandingUUID *uuid.UUID, allowRegistration bool, _ *string, _ *string, _ *bool, _ *bool, actor ClientActor, _ *string) (*ClientCreateServiceResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/clients", validClientBody()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{createFn: func(tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, idpUUID string, brandingUUID *uuid.UUID, allowRegistration bool, _ *string, _ *string, _ *bool, _ *bool, actor ClientActor, _ *string) (*ClientCreateServiceResult, error) {
			return &ClientCreateServiceResult{Client: &ClientServiceDataResult{Name: n}}, nil
		}}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/clients", validClientBody()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestClientHandler_Update(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/clients/"+testResourceUUID.String(), validClientBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/clients/bad", validClientBody()), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPut, "/clients/"+testResourceUUID.String()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/clients/"+testResourceUUID.String(), map[string]any{"name": "x"}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{updateFn: func(id uuid.UUID, tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, brandingUUID *uuid.UUID, allowRegistration *bool, _ *string, _ *string, _ *bool, _ *bool, actor ClientActor, _ *string) (*ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/clients/"+testResourceUUID.String(), validClientBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{updateFn: func(id uuid.UUID, tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, brandingUUID *uuid.UUID, allowRegistration *bool, _ *string, _ *string, _ *bool, _ *bool, actor ClientActor, _ *string) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{Name: n}, nil
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/clients/"+testResourceUUID.String(), validClientBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_SetStatus(t *testing.T) {
	// The endpoint applies the status the caller asked for. It used to ignore the
	// body and flip whatever was currently in the DB, so a stale view could
	// invert the operator's choice — "Activate" silently deactivating a client.
	body := func(status string) *strings.Reader {
		return strings.NewReader(`{"status":"` + status + `"}`)
	}
	req := func(b io.Reader) *http.Request {
		return withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", b), "client_uuid", testResourceUUID.String()))
	}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", body("active")), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", body("active")), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing body returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).SetStatus(w, req(nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unknown status is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).SetStatus(w, req(body("banana")))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// The core of the fix: the requested status is what gets applied, regardless
	// of what the client's current status happens to be.
	for _, tc := range []struct{ current, requested string }{
		{"active", "active"},
		{"active", "inactive"},
		{"inactive", "active"},
		{"inactive", "inactive"},
	} {
		t.Run("applies requested "+tc.requested+" when currently "+tc.current, func(t *testing.T) {
			var applied string
			svc := &mockClientService{
				getByUUIDFn: func(uuid.UUID, int64) (*ClientServiceDataResult, error) {
					return &ClientServiceDataResult{Status: tc.current}, nil
				},
				setStatusByUUIDFn: func(_ uuid.UUID, _ int64, s string, _ uuid.UUID) (*ClientServiceDataResult, error) {
					applied = s
					return &ClientServiceDataResult{Status: s}, nil
				},
			}
			w := httptest.NewRecorder()
			NewClientHandler(svc).SetStatus(w, req(body(tc.requested)))
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tc.requested, applied, "must apply the requested status, not toggle the current one")
		})
	}

	t.Run("set status service error returns 500", func(t *testing.T) {
		svc := &mockClientService{
			setStatusByUUIDFn: func(uuid.UUID, int64, string, uuid.UUID) (*ClientServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		w := httptest.NewRecorder()
		NewClientHandler(svc).SetStatus(w, req(body("active")))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestClientHandler_Delete_NoTenant(t *testing.T) {
	h := NewClientHandler(&mockClientService{})
	r := withUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClientHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewClientHandler(&mockClientService{})
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/clients/bad", nil), "client_uuid", "bad"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientHandler_GetURIs(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetURIs(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetURIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockClientService{getByUUIDFn: func(id uuid.UUID, tid int64) (*ClientServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetURIs(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("success with uris", func(t *testing.T) {
		uris := []ClientURIServiceDataResult{{ClientURIUUID: uuid.New(), URI: "https://example.com", Type: "redirect_uri"}}
		svc := &mockClientService{getByUUIDFn: func(id uuid.UUID, tid int64) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{ClientURIs: &uris}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetURIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("success with nil uris", func(t *testing.T) {
		svc := &mockClientService{getByUUIDFn: func(id uuid.UUID, tid int64) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetURIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func validURIBody() map[string]any {
	return map[string]any{"uri": "https://example.com/cb", "type": "redirect_uri"}
}

func TestClientHandler_CreateURI(t *testing.T) {
	clientURI := ClientURIServiceDataResult{ClientURIUUID: uuid.New(), URI: "https://example.com/cb", Type: "redirect_uri"}
	uris := []ClientURIServiceDataResult{clientURI}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(jsonReq(t, http.MethodPost, "/", validURIBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).CreateURI(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validURIBody()), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).CreateURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPost, "/"), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).CreateURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"uri": "x"}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).CreateURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{createURIFn: func(id uuid.UUID, tid int64, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validURIBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).CreateURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{createURIFn: func(id uuid.UUID, tid int64, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{ClientURIs: &uris}, nil
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validURIBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).CreateURI(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestClientHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockClientService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*ClientServiceDataResult, error) {
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
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{Name: "c1"}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientHandler_UpdateURI(t *testing.T) {
	uriUUID := uuid.New()
	matchingURI := ClientURIServiceDataResult{ClientURIUUID: uriUUID, URI: "https://example.com/cb", Type: "redirect_uri"}
	uris := []ClientURIServiceDataResult{matchingURI}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateURI(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", "bad"), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid client_uri_uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", testResourceUUID.String()), "client_uri_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(badJSONReq(t, http.MethodPut, "/"), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"uri": "x"}), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{updateURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).UpdateURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("updated uri not found returns 500", func(t *testing.T) {
		// service returns result with nil ClientURIs → updatedURI stays nil
		svc := &mockClientService{updateURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{}, nil
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).UpdateURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{updateURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{ClientURIs: &uris}, nil
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).UpdateURI(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_DeleteURI(t *testing.T) {
	uriUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).DeleteURI(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", "bad"), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).DeleteURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid client_uri_uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_uri_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).DeleteURI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{deleteURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).DeleteURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{deleteURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{Name: "c1"}, nil
		}}
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).DeleteURI(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_GetAPIs(t *testing.T) {
	apiUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetAPIs(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{getClientAPIsFn: func(tid int64, id uuid.UUID) ([]ClientAPIServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetAPIs(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success with permissions", func(t *testing.T) {
		svc := &mockClientService{getClientAPIsFn: func(tid int64, id uuid.UUID) ([]ClientAPIServiceDataResult, error) {
			return []ClientAPIServiceDataResult{{
				ClientAPIUUID: uuid.New(),
				Api:           APIServiceDataResult{APIUUID: apiUUID, Name: "api1"},
				Permissions:   []PermissionServiceDataResult{{Name: "read"}},
			}}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetAPIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_AddAPIs(t *testing.T) {
	apiUUID := uuid.New()

	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_ids": []string{apiUUID.String()}}), "client_uuid", "bad")
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPost, "/"), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_ids": []string{apiUUID.String()}}), "client_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{addClientAPIsFn: func(tid int64, id uuid.UUID, apis []uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_ids": []string{apiUUID.String()}}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).AddAPIs(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	// A grant change must be attributable, so a tenant alone is not enough.
	t.Run("no user returns 401", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_ids": []string{apiUUID.String()}}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	// The DTO rules existed but were never run, so an empty list reached the service.
	t.Run("empty api list is rejected", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_ids": []string{}}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_ids": []string{apiUUID.String()}}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_RemoveAPI(t *testing.T) {
	apiUUID := uuid.New()

	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", "bad"), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid api uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", "bad")
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPI(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPI(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{removeClientAPIFn: func(tid int64, id, api uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RemoveAPI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPI(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_GetAPIPermissions(t *testing.T) {
	apiUUID := uuid.New()

	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", "bad"), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid api uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", "bad")
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{getClientAPIPermsFn: func(tid int64, id, api uuid.UUID) ([]PermissionServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{getClientAPIPermsFn: func(tid int64, id, api uuid.UUID) ([]PermissionServiceDataResult, error) {
			return []PermissionServiceDataResult{{Name: "read"}}, nil
		}}
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_AddAPIPermissions(t *testing.T) {
	apiUUID := uuid.New()
	permUUID := uuid.New()

	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_ids": []string{permUUID.String()}}), "client_uuid", "bad"), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid api uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_ids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", "bad")
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(badJSONReq(t, http.MethodPost, "/"), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_ids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{addClientAPIPermsFn: func(tid int64, id, api uuid.UUID, perms []uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_ids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_ids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_RotateSecret(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RotateSecret(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RotateSecret(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{rotateSecretFn: func(uuid.UUID, int64, uuid.UUID, int) (string, error) {
			return "", errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RotateSecret(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{rotateSecretFn: func(uuid.UUID, int64, uuid.UUID, int) (string, error) {
			return "new-secret-value", nil
		}}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPost, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RotateSecret(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_RemoveAPIPermission(t *testing.T) {
	apiUUID := uuid.New()
	permUUID := uuid.New()

	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", "bad"), "api_uuid", apiUUID.String()), "permission_uuid", permUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid api uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", "bad"), "permission_uuid", permUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid permission uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()), "permission_uuid", "bad")
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()), "permission_uuid", permUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{removeClientAPIPermFn: func(tid int64, id, api, perm uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()), "permission_uuid", permUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()), "permission_uuid", permUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func validConnectionBody() map[string]any {
	return map[string]any{
		"identity_provider_id": uuid.New().String(),
		"is_default":           false,
		"enabled":              true,
		"display_order":        0,
	}
}

func TestClientHandler_GetConnections(t *testing.T) {
	conns := []ClientIdentityProviderServiceDataResult{{ClientIdentityProviderUUID: uuid.New(), Enabled: true}}

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetConnections(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetConnections(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{getConnectionsFn: func(id uuid.UUID, tid int64) ([]ClientIdentityProviderServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetConnections(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{getConnectionsFn: func(id uuid.UUID, tid int64) ([]ClientIdentityProviderServiceDataResult, error) {
			return conns, nil
		}}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetConnections(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_AddConnection(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(jsonReq(t, http.MethodPost, "/", validConnectionBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddConnection(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validConnectionBody()), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(badJSONReq(t, http.MethodPost, "/"), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid identity provider uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"identity_provider_id": "not-a-uuid"}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{addConnectionFn: func(id uuid.UUID, tid int64, idp uuid.UUID, isDefault, enabled bool, order int, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validConnectionBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).AddConnection(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{addConnectionFn: func(id uuid.UUID, tid int64, idp uuid.UUID, isDefault, enabled bool, order int, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{Name: "c1"}, nil
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validConnectionBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).AddConnection(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestClientHandler_UpdateConnection(t *testing.T) {
	connUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validConnectionBody()), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateConnection(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validConnectionBody()), "client_uuid", "bad"), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid connection uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validConnectionBody()), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(badJSONReq(t, http.MethodPut, "/"), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"display_order": -1}), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).UpdateConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{updateConnectionFn: func(id uuid.UUID, tid int64, conn uuid.UUID, isDefault, enabled *bool, order *int, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validConnectionBody()), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).UpdateConnection(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{updateConnectionFn: func(id uuid.UUID, tid int64, conn uuid.UUID, isDefault, enabled *bool, order *int, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{Name: "c1"}, nil
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validConnectionBody()), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).UpdateConnection(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_RemoveConnection(t *testing.T) {
	connUUID := uuid.New()

	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveConnection(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid client uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", "bad"), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid connection uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveConnection(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{removeConnectionFn: func(id uuid.UUID, tid int64, conn uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RemoveConnection(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{removeConnectionFn: func(id uuid.UUID, tid int64, conn uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
			return &ClientServiceDataResult{Name: "c1"}, nil
		}}
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_identity_provider_uuid", connUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RemoveConnection(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
