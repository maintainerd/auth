package rest

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

func TestClientHandler_Get_NoTenant(t *testing.T) {
	h := NewClientHandler(&mockClientService{})
	r := httptest.NewRequest(http.MethodGet, "/clients", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestClientHandler_Get_ServiceError(t *testing.T) {
	svc := &mockClientService{
		getFn: func(service.ClientServiceGetFilter) (*service.ClientServiceGetResult, error) {
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
		getFn: func(service.ClientServiceGetFilter) (*service.ClientServiceGetResult, error) {
			return &service.ClientServiceGetResult{}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/clients?page=1&limit=10", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientHandler_Get_ValidationError(t *testing.T) {
	// invalid status value triggers ClientFilterDto.Validate failure
	h := NewClientHandler(&mockClientService{})
	r := withTenant(httptest.NewRequest(http.MethodGet, "/clients?status=bad_status", nil))
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientHandler_Get_WithFilters(t *testing.T) {
	// Covers is_default, is_system, status array trim, client_type array trim, and result rows loop
	svc := &mockClientService{
		getFn: func(service.ClientServiceGetFilter) (*service.ClientServiceGetResult, error) {
			return &service.ClientServiceGetResult{
				Data: []service.ClientServiceDataResult{{Name: "c1"}},
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
	// Covers the IdentityProvider, ClientURIs, and Permissions branches in toClientResponseDto
	uriUUID := uuid.New()
	uris := []service.ClientURIServiceDataResult{{ClientURIUUID: uriUUID, URI: "https://example.com", Type: "redirect-uri"}}
	perms := []service.PermissionServiceDataResult{{Name: "read"}}
	svc := &mockClientService{
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{
				Name:             "c1",
				IdentityProvider: &service.IdentityProviderServiceDataResult{Name: "idp1"},
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
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return nil, assert.AnError
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
		getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{Name: "client1"}, nil
		},
	}
	h := NewClientHandler(svc)
	r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/clients/"+testResourceUUID.String(), nil), "client_uuid", testResourceUUID.String()))
	w := httptest.NewRecorder()
	h.GetByUUID(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientHandler_GetSecretByUUID(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetSecretByUUID(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).GetSecretByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockClientService{getSecretByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientSecretServiceDataResult, error) {
			return nil, errors.New("not found")
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetSecretByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{getSecretByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientSecretServiceDataResult, error) {
			secret := "s3cr3t"
			return &service.ClientSecretServiceDataResult{ClientID: "cid", ClientSecret: &secret}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetSecretByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

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
			return nil, errors.New("not found")
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
		svc := &mockClientService{createFn: func(tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDefault bool, idpUUID string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(jsonReq(t, http.MethodPost, "/clients", validClientBody()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).Create(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{createFn: func(tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDefault bool, idpUUID string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{Name: n}, nil
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
		svc := &mockClientService{updateFn: func(id uuid.UUID, tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDefault bool, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/clients/"+testResourceUUID.String(), validClientBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{updateFn: func(id uuid.UUID, tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDefault bool, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{Name: n}, nil
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPut, "/clients/"+testResourceUUID.String(), validClientBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestClientHandler_SetStatus(t *testing.T) {
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "client_uuid", "bad"))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("get by uuid error returns 404", func(t *testing.T) {
		svc := &mockClientService{getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return nil, errors.New("not found")
		}}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("active client toggled to inactive", func(t *testing.T) {
		svc := &mockClientService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
				return &service.ClientServiceDataResult{Status: "active"}, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
				return &service.ClientServiceDataResult{Status: s}, nil
			},
		}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).SetStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("set status service error returns 500", func(t *testing.T) {
		svc := &mockClientService{
			getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
				return &service.ClientServiceDataResult{Status: "inactive"}, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, tid int64, s string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodPatch, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).SetStatus(w, r)
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
		svc := &mockClientService{getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return nil, errors.New("not found")
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetURIs(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
	t.Run("success with uris", func(t *testing.T) {
		uris := []service.ClientURIServiceDataResult{{ClientURIUUID: uuid.New(), URI: "https://example.com", Type: "redirect-uri"}}
		svc := &mockClientService{getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{ClientURIs: &uris}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetURIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
	t.Run("success with nil uris", func(t *testing.T) {
		svc := &mockClientService{getByUUIDFn: func(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{}, nil
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetURIs(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func validURIBody() map[string]any {
	return map[string]any{"uri": "https://example.com/cb", "type": "redirect-uri"}
}

func TestClientHandler_CreateURI(t *testing.T) {
	clientURI := service.ClientURIServiceDataResult{ClientURIUUID: uuid.New(), URI: "https://example.com/cb", Type: "redirect-uri"}
	uris := []service.ClientURIServiceDataResult{clientURI}

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
		svc := &mockClientService{createURIFn: func(id uuid.UUID, tid int64, uri, uriType string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validURIBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).CreateURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{createURIFn: func(id uuid.UUID, tid int64, uri, uriType string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{ClientURIs: &uris}, nil
		}}
		r := withTenantAndUser(withChiParam(jsonReq(t, http.MethodPost, "/", validURIBody()), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).CreateURI(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestClientHandler_Delete_ServiceError(t *testing.T) {
	svc := &mockClientService{
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
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
		deleteByUUIDFn: func(id uuid.UUID, tid int64, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{Name: "c1"}, nil
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
	matchingURI := service.ClientURIServiceDataResult{ClientURIUUID: uriUUID, URI: "https://example.com/cb", Type: "redirect-uri"}
	uris := []service.ClientURIServiceDataResult{matchingURI}

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
		svc := &mockClientService{updateURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).UpdateURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("updated uri not found returns 500", func(t *testing.T) {
		// service returns result with nil ClientURIs → updatedURI stays nil
		svc := &mockClientService{updateURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{}, nil
		}}
		r := withTenantAndUser(withChiParam(withChiParam(jsonReq(t, http.MethodPut, "/", validURIBody()), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).UpdateURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{updateURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{ClientURIs: &uris}, nil
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
		svc := &mockClientService{deleteURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "client_uri_uuid", uriUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).DeleteURI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{deleteURIFn: func(id uuid.UUID, tid int64, uriID uuid.UUID, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
			return &service.ClientServiceDataResult{Name: "c1"}, nil
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
		svc := &mockClientService{getClientApisFn: func(tid int64, id uuid.UUID) ([]service.ClientApiServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetAPIs(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success with permissions", func(t *testing.T) {
		svc := &mockClientService{getClientApisFn: func(tid int64, id uuid.UUID) ([]service.ClientApiServiceDataResult, error) {
			return []service.ClientApiServiceDataResult{{
				ClientApiUUID: uuid.New(),
				Api:           service.APIServiceDataResult{APIUUID: apiUUID, Name: "api1"},
				Permissions:   []service.PermissionServiceDataResult{{Name: "read"}},
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
		r := withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{apiUUID.String()}}), "client_uuid", "bad")
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(badJSONReq(t, http.MethodPost, "/"), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{apiUUID.String()}}), "client_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIs(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{addClientApisFn: func(tid int64, id uuid.UUID, apis []uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{apiUUID.String()}}), "client_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).AddAPIs(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenant(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"api_uuids": []string{apiUUID.String()}}), "client_uuid", testResourceUUID.String()))
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
		svc := &mockClientService{removeClientApiFn: func(tid int64, id, api uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RemoveAPI(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
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
		svc := &mockClientService{getClientApiPermsFn: func(tid int64, id, api uuid.UUID) ([]service.PermissionServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenant(withChiParam(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).GetAPIPermissions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		svc := &mockClientService{getClientApiPermsFn: func(tid int64, id, api uuid.UUID) ([]service.PermissionServiceDataResult, error) {
			return []service.PermissionServiceDataResult{{Name: "read"}}, nil
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
		r := withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}}), "client_uuid", "bad"), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("invalid api uuid returns 400", func(t *testing.T) {
		r := withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", "bad")
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("bad json returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(withChiParam(badJSONReq(t, http.MethodPost, "/"), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	t.Run("no tenant returns 401", func(t *testing.T) {
		r := withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String())
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
	t.Run("service error returns 500", func(t *testing.T) {
		svc := &mockClientService{addClientApiPermsFn: func(tid int64, id, api uuid.UUID, perms []uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenant(withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).AddAPIPermissions(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenant(withChiParam(withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"permission_uuids": []string{permUUID.String()}}), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).AddAPIPermissions(w, r)
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
		svc := &mockClientService{removeClientApiPermFn: func(tid int64, id, api, perm uuid.UUID) error {
			return errors.New("db error")
		}}
		r := withTenant(withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()), "permission_uuid", permUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(svc).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
	t.Run("success", func(t *testing.T) {
		r := withTenant(withChiParam(withChiParam(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "client_uuid", testResourceUUID.String()), "api_uuid", apiUUID.String()), "permission_uuid", permUUID.String()))
		w := httptest.NewRecorder()
		NewClientHandler(&mockClientService{}).RemoveAPIPermission(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
