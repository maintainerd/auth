package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newTenantHandler(ts *mockTenantService, ms *mockTenantMemberService) *TenantHandler {
	if ts == nil {
		ts = &mockTenantService{}
	}
	if ms == nil {
		ms = &mockTenantMemberService{}
	}
	return NewTenantHandler(ts, ms, nil, nil)
}

func TestTenantHandler_Get(t *testing.T) {
	t.Run("validation error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10&sort_order=bad", nil)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Get(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// systemGet returns a system tenant whose ID matches withTenant's auth tenant
	// so the actor counts as a system-tenant member (sees all tenants, unscoped).
	systemGet := func(getFn func(TenantServiceGetFilter) (*TenantServiceGetResult, error)) *mockTenantService {
		return &mockTenantService{
			getFn:       getFn,
			getSystemFn: func() (*TenantServiceDataResult, error) { return &TenantServiceDataResult{TenantID: tenantID}, nil },
		}
	}

	t.Run("no auth tenant returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10", nil)
		w := httptest.NewRecorder()
		newTenantHandler(systemGet(nil), nil).Get(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := systemGet(func(TenantServiceGetFilter) (*TenantServiceGetResult, error) {
			return nil, errors.New("db error")
		})
		r := withTenant(httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10", nil))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Get(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with no rows covers empty result branch", func(t *testing.T) {
		svc := systemGet(func(TenantServiceGetFilter) (*TenantServiceGetResult, error) {
			return &TenantServiceGetResult{Data: nil, Total: 0, Page: 1, Limit: 10}, nil
		})
		r := withTenant(httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10", nil))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success with all filters and rows covers filter+loop branches", func(t *testing.T) {
		svc := systemGet(func(TenantServiceGetFilter) (*TenantServiceGetResult, error) {
			return &TenantServiceGetResult{Data: []TenantServiceDataResult{{Name: "t1"}}}, nil
		})
		r := withTenant(httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10&status=active&is_default=true&is_system=false&is_public=true", nil))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("regular tenant user is scoped to own tenant", func(t *testing.T) {
		var gotFilter TenantServiceGetFilter
		svc := &mockTenantService{
			getFn: func(f TenantServiceGetFilter) (*TenantServiceGetResult, error) {
				gotFilter = f
				return &TenantServiceGetResult{}, nil
			},
			// System tenant differs from the actor's tenant → actor is NOT a
			// system member → listing must be scoped to their own tenant.
			getSystemFn: func() (*TenantServiceDataResult, error) {
				return &TenantServiceDataResult{TenantID: tenantID + 999}, nil
			},
		}
		r := withTenant(httptest.NewRequest(http.MethodGet, "/tenants?page=1&limit=10", nil))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Get(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, []int64{tenantID}, gotFilter.TenantIDs)
	})
}

func TestTenantHandler_GetByUUID(t *testing.T) {
	systemSvc := func(byUUID func(uuid.UUID) (*TenantServiceDataResult, error)) *mockTenantService {
		return &mockTenantService{
			getByUUIDFn: byUUID,
			getSystemFn: func() (*TenantServiceDataResult, error) { return &TenantServiceDataResult{TenantID: tenantID}, nil },
		}
	}

	t.Run("no auth tenant returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", "bad"))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := systemSvc(func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("db error")
		})
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		svc := systemSvc(func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		})
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := systemSvc(func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "tenant1"}, nil
		})
		r := withTenant(withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByUUID(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_GetDefault(t *testing.T) {
	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetDefault(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "system"}, nil
		}}
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetDefault(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// mockSurfaceClientReader is an inline SurfaceClientReader for bootstrap tests.
type mockSurfaceClientReader struct {
	fn func(tenantName, surface string) (*SurfaceClient, error)
}

func (m *mockSurfaceClientReader) GetSurfaceClient(_ context.Context, tenantName, surface string) (*SurfaceClient, error) {
	if m.fn != nil {
		return m.fn(tenantName, surface)
	}
	return nil, nil
}

// mockSurfaceConnectionsReader is an inline SurfaceConnectionsReader for
// bootstrap tests.
type mockSurfaceConnectionsReader struct {
	fn func(clientIdentifier string) (SurfaceLoginMethods, error)
}

func (m *mockSurfaceConnectionsReader) ListSurfaceConnections(_ context.Context, clientIdentifier string) (SurfaceLoginMethods, error) {
	if m.fn != nil {
		return m.fn(clientIdentifier)
	}
	return SurfaceLoginMethods{}, nil
}

type mockClientBrandingReader struct {
	fn func(tenantID int64, clientIdentifier string) (*branding.BrandingServiceDataResult, error)
}

func (m *mockClientBrandingReader) GetPublicClientBranding(_ context.Context, tenantID int64, clientIdentifier string) (*branding.BrandingServiceDataResult, error) {
	if m.fn != nil {
		return m.fn(tenantID, clientIdentifier)
	}
	return nil, nil
}

type mockBootstrapBrandingService struct {
	branding.BrandingService
	fn func(tenantID int64) (*branding.BrandingServiceDataResult, error)
}

func (m *mockBootstrapBrandingService) GetPublic(_ context.Context, tenantID int64) (*branding.BrandingServiceDataResult, error) {
	if m.fn != nil {
		return m.fn(tenantID)
	}
	return nil, nil
}

func TestTenantHandler_GetBootstrap(t *testing.T) {
	setBootstrapBases := func(t *testing.T) {
		t.Helper()
		origIdentity := config.AppFrontendIdentityHostname
		origConsole := config.AppFrontendConsoleHostname
		config.AppFrontendIdentityHostname = "https://auth.maintainerd.local"
		config.AppFrontendConsoleHostname = "https://console.auth.maintainerd.local"
		t.Cleanup(func() {
			config.AppFrontendIdentityHostname = origIdentity
			config.AppFrontendConsoleHostname = origConsole
		})
	}

	decode := func(t *testing.T, w *httptest.ResponseRecorder) TenantBootstrapResponseDTO {
		t.Helper()
		var body struct {
			Data TenantBootstrapResponseDTO `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body.Data
	}

	t.Run("unknown domain returns 404", func(t *testing.T) {
		setBootstrapBases(t)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=evil.example.com", nil)
		newTenantHandler(&mockTenantService{}, nil).GetDefault(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	// The hosted login page renders its provider buttons straight from this
	// payload, so the bootstrap must carry the surface client's connections.
	t.Run("includes the surface client's federated connections", func(t *testing.T) {
		setBootstrapBases(t)
		var askedFor string
		svc := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "system", Status: "active", IsSystem: true}, nil
		}}
		h := newTenantHandler(svc, nil)
		h.SetSurfaceClientReader(&mockSurfaceClientReader{fn: func(_, _ string) (*SurfaceClient, error) {
			return &SurfaceClient{ClientID: "cid-identity", Name: "auth-identity", ClientType: "spa"}, nil
		}})
		h.SetSurfaceConnectionsReader(&mockSurfaceConnectionsReader{fn: func(clientIdentifier string) (SurfaceLoginMethods, error) {
			askedFor = clientIdentifier
			return SurfaceLoginMethods{Connections: []SurfaceConnection{{
				Identifier:   "idp-cognito",
				DisplayName:  "AWS Cognito",
				Provider:     "cognito",
				ProviderType: "enterprise",
				DisplayOrder: 1,
			}}}, nil
		}})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=auth.maintainerd.local", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		// Looked up by the resolved surface client, not the query string.
		assert.Equal(t, "cid-identity", askedFor)
		got := decode(t, w)
		require.Len(t, got.Connections, 1)
		assert.Equal(t, "idp-cognito", got.Connections[0].Identifier)
		assert.Equal(t, "AWS Cognito", got.Connections[0].DisplayName)
		assert.Equal(t, 1, got.Connections[0].DisplayOrder)
	})

	t.Run("connections degrade to an empty array, never null", func(t *testing.T) {
		setBootstrapBases(t)
		svc := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "system", Status: "active", IsSystem: true}, nil
		}}
		h := newTenantHandler(svc, nil)
		h.SetSurfaceClientReader(&mockSurfaceClientReader{fn: func(_, _ string) (*SurfaceClient, error) {
			return &SurfaceClient{ClientID: "cid-identity", Name: "auth-identity"}, nil
		}})
		// A failing reader must not fail the whole bootstrap — the login page
		// still needs its tenant, branding and password policy.
		h.SetSurfaceConnectionsReader(&mockSurfaceConnectionsReader{fn: func(string) (SurfaceLoginMethods, error) {
			return SurfaceLoginMethods{}, errors.New("connections unavailable")
		}})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=auth.maintainerd.local", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		assert.Empty(t, decode(t, w).Connections)
		assert.Contains(t, w.Body.String(), `"connections":[]`)
	})

	t.Run("system identity host resolves system tenant with identity surface", func(t *testing.T) {
		setBootstrapBases(t)
		svc := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "system", DisplayName: "System", Status: "active", IsSystem: true}, nil
		}}
		h := newTenantHandler(svc, nil)
		h.SetSurfaceClientReader(&mockSurfaceClientReader{fn: func(_, surface string) (*SurfaceClient, error) {
			return &SurfaceClient{ClientID: "cid-identity", Name: "auth-identity", ClientType: "spa"}, nil
		}})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=auth.maintainerd.local", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		got := decode(t, w)
		assert.Equal(t, "identity", got.Surface)
		assert.True(t, got.Tenant.IsSystem)
		assert.Equal(t, "system", got.Tenant.Name)
		assert.Equal(t, "https://auth.maintainerd.local", got.IdentityURL)
		assert.Equal(t, "https://console.auth.maintainerd.local", got.ConsoleURL)
		require.NotNil(t, got.Client)
		assert.Equal(t, "cid-identity", got.Client.ClientID)
		assert.Equal(t, "auth-identity", got.Client.Name)
	})

	t.Run("tenant console subdomain resolves by name with console surface", func(t *testing.T) {
		setBootstrapBases(t)
		var gotSurface string
		svc := &mockTenantService{getByNameFn: func(name string) (*TenantServiceDataResult, error) {
			assert.Equal(t, "acme", name)
			return &TenantServiceDataResult{Name: "acme", Status: "active"}, nil
		}}
		h := newTenantHandler(svc, nil)
		h.SetSurfaceClientReader(&mockSurfaceClientReader{fn: func(_, surface string) (*SurfaceClient, error) {
			gotSurface = surface
			return &SurfaceClient{ClientID: "cid-console", Name: "auth-console", ClientType: "spa"}, nil
		}})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=acme.console.auth.maintainerd.local", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		got := decode(t, w)
		assert.Equal(t, "console", got.Surface)
		assert.Equal(t, "console", gotSurface)
		assert.False(t, got.Tenant.IsSystem)
		assert.Equal(t, "acme", got.Tenant.Name)
		// Per-tenant subdomain URLs.
		assert.Equal(t, "https://acme.auth.maintainerd.local", got.IdentityURL)
		assert.Equal(t, "https://acme.console.auth.maintainerd.local", got.ConsoleURL)
		require.NotNil(t, got.Client)
		assert.Equal(t, "cid-console", got.Client.ClientID)
	})

	t.Run("includes active public branding metadata for theming", func(t *testing.T) {
		setBootstrapBases(t)
		svc := &mockTenantService{getByNameFn: func(name string) (*TenantServiceDataResult, error) {
			assert.Equal(t, "acme", name)
			return &TenantServiceDataResult{TenantID: 42, Name: "acme", Status: "active"}, nil
		}}
		brandingSvc := &mockBootstrapBrandingService{fn: func(tenantID int64) (*branding.BrandingServiceDataResult, error) {
			assert.Equal(t, int64(42), tenantID)
			return &branding.BrandingServiceDataResult{
				CompanyName:           "Acme IAM",
				LogoLabel:             "Acme",
				LogoDetail:            "Acme Console",
				ShowLogoLabel:         true,
				IdentityLogoLabel:     "Acme Public",
				IdentityShowLogoLabel: true,
				LogoURL:               "https://cdn.example.test/acme.svg",
				Metadata:              datatypes.JSON([]byte(`{"colors":{"topPanelBackground":"#101820"},"components":{"input":{"borderRadius":"12px"}}}`)),
			}, nil
		}}
		h := NewTenantHandler(svc, nil, brandingSvc, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=acme.console.auth.maintainerd.local", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		got := decode(t, w)
		require.NotNil(t, got.Branding)
		assert.Equal(t, "Acme IAM", got.Branding.CompanyName)
		assert.Equal(t, "Acme", got.Branding.LogoLabel)
		assert.Equal(t, "Acme Console", got.Branding.LogoDetail)
		assert.True(t, got.Branding.ShowLogoLabel)
		assert.Equal(t, "Acme Public", got.Branding.IdentityLogoLabel)
		assert.True(t, got.Branding.IdentityShowLogoLabel)
		assert.Equal(t, "https://cdn.example.test/acme.svg", got.Branding.LogoURL)

		var metadata map[string]any
		require.NoError(t, json.Unmarshal(got.Branding.Metadata, &metadata))
		colors := metadata["colors"].(map[string]any)
		assert.Equal(t, "#101820", colors["topPanelBackground"])
		components := metadata["components"].(map[string]any)
		input := components["input"].(map[string]any)
		assert.Equal(t, "12px", input["borderRadius"])
	})

	t.Run("client_id selects attached client branding", func(t *testing.T) {
		setBootstrapBases(t)
		svc := &mockTenantService{getByNameFn: func(name string) (*TenantServiceDataResult, error) {
			assert.Equal(t, "acme", name)
			return &TenantServiceDataResult{TenantID: 42, Name: "acme", Status: "active"}, nil
		}}
		brandingSvc := &mockBootstrapBrandingService{fn: func(int64) (*branding.BrandingServiceDataResult, error) {
			t.Fatal("tenant fallback branding should not be used when client branding resolves")
			return nil, nil
		}}
		h := NewTenantHandler(svc, nil, brandingSvc, nil)
		h.SetClientBrandingReader(&mockClientBrandingReader{fn: func(tenantID int64, clientIdentifier string) (*branding.BrandingServiceDataResult, error) {
			assert.Equal(t, int64(42), tenantID)
			assert.Equal(t, "client-abc", clientIdentifier)
			return &branding.BrandingServiceDataResult{
				CompanyName:           "Client App",
				LogoLabel:             "Client",
				LogoDetail:            "Client Console",
				ShowLogoLabel:         false,
				IdentityLogoLabel:     "Client Public",
				IdentityShowLogoLabel: true,
				Metadata:              datatypes.JSON([]byte(`{"colors":{"authPageBackground":"#050505"}}`)),
			}, nil
		}})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=acme.auth.maintainerd.local&client_id=client-abc", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		got := decode(t, w)
		require.NotNil(t, got.Branding)
		assert.Equal(t, "Client App", got.Branding.CompanyName)
		assert.Equal(t, "Client", got.Branding.LogoLabel)
		assert.Equal(t, "Client Console", got.Branding.LogoDetail)
		assert.False(t, got.Branding.ShowLogoLabel)
		assert.Equal(t, "Client Public", got.Branding.IdentityLogoLabel)
		assert.True(t, got.Branding.IdentityShowLogoLabel)

		var metadata map[string]any
		require.NoError(t, json.Unmarshal(got.Branding.Metadata, &metadata))
		colors := metadata["colors"].(map[string]any)
		assert.Equal(t, "#050505", colors["authPageBackground"])
	})

	t.Run("client_id without attached branding falls back to active tenant branding", func(t *testing.T) {
		setBootstrapBases(t)
		svc := &mockTenantService{getByNameFn: func(name string) (*TenantServiceDataResult, error) {
			assert.Equal(t, "acme", name)
			return &TenantServiceDataResult{TenantID: 42, Name: "acme", Status: "active"}, nil
		}}
		brandingSvc := &mockBootstrapBrandingService{fn: func(tenantID int64) (*branding.BrandingServiceDataResult, error) {
			assert.Equal(t, int64(42), tenantID)
			return &branding.BrandingServiceDataResult{CompanyName: "Tenant Theme", LogoLabel: "Tenant", ShowLogoLabel: true}, nil
		}}
		h := NewTenantHandler(svc, nil, brandingSvc, nil)
		h.SetClientBrandingReader(&mockClientBrandingReader{fn: func(tenantID int64, clientIdentifier string) (*branding.BrandingServiceDataResult, error) {
			assert.Equal(t, int64(42), tenantID)
			assert.Equal(t, "client-abc", clientIdentifier)
			return nil, nil
		}})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=acme.auth.maintainerd.local&client_id=client-abc", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		got := decode(t, w)
		require.NotNil(t, got.Branding)
		assert.Equal(t, "Tenant Theme", got.Branding.CompanyName)
		assert.Equal(t, "Tenant", got.Branding.LogoLabel)
		assert.True(t, got.Branding.ShowLogoLabel)
	})

	t.Run("missing surface client still returns 200 without client field", func(t *testing.T) {
		setBootstrapBases(t)
		svc := &mockTenantService{getByNameFn: func(string) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "acme", Status: "active"}, nil
		}}
		h := newTenantHandler(svc, nil)
		h.SetSurfaceClientReader(&mockSurfaceClientReader{fn: func(_, _ string) (*SurfaceClient, error) {
			return nil, errNotFound
		}})
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=acme.auth.maintainerd.local", nil)
		h.GetDefault(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		assert.Nil(t, decode(t, w).Client)
	})

	t.Run("tenant not found returns 404", func(t *testing.T) {
		setBootstrapBases(t)
		svc := &mockTenantService{getByNameFn: func(string) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/?domain=acme.auth.maintainerd.local", nil)
		newTenantHandler(svc, nil).GetDefault(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTenantHandler_GetByName(t *testing.T) {
	t.Run("empty name returns 400", func(t *testing.T) {
		// no chi param set → chi.URLParam returns ""
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetByName(w, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 404", func(t *testing.T) {
		svc := &mockTenantService{getByNameFn: func(string) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "name", "my-tenant")
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByName(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockTenantService{getByNameFn: func(string) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "t"}, nil
		}}
		r := withChiParam(httptest.NewRequest(http.MethodGet, "/", nil), "name", "my-tenant")
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).GetByName(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_Create(t *testing.T) {
	validBody := map[string]any{"name": "my-tenant", "display_name": "My Tenant", "description": "A long enough description", "status": "active"}

	// systemSvc returns a system tenant whose ID matches the auth tenant injected
	// by withTenant, so the system-tenant create gate passes.
	systemSvc := func() *mockTenantService {
		return &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: tenantID}, nil
		}}
	}

	t.Run("no auth tenant returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/tenants", validBody)
		w := httptest.NewRecorder()
		newTenantHandler(systemSvc(), nil).Create(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("non-system tenant returns 403", func(t *testing.T) {
		svc := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: tenantID + 999}, nil
		}}
		r := withTenant(jsonReq(t, http.MethodPost, "/tenants", validBody))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Create(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withTenant(badJSONReq(t, http.MethodPost, "/tenants"))
		w := httptest.NewRecorder()
		newTenantHandler(systemSvc(), nil).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withTenant(jsonReq(t, http.MethodPost, "/tenants", map[string]any{"name": ""}))
		w := httptest.NewRecorder()
		newTenantHandler(systemSvc(), nil).Create(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := systemSvc()
		svc.createFn = func(n, dn, desc, s string) (*TenantServiceDataResult, error) {
			return nil, errors.New("db error")
		}
		r := withTenant(jsonReq(t, http.MethodPost, "/tenants", validBody))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Create(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 201", func(t *testing.T) {
		svc := systemSvc()
		svc.createFn = func(n, dn, desc, s string) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: n}, nil
		}
		r := withTenant(jsonReq(t, http.MethodPost, "/tenants", validBody))
		w := httptest.NewRecorder()
		newTenantHandler(svc, nil).Create(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestTenantHandler_Update(t *testing.T) {
	validBody := map[string]any{"name": "updated", "display_name": "Updated", "description": "A long enough description", "status": "active"}

	t.Run("no user returns 401", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", validBody)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Update(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", "bad"))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("IsUserInTenant error returns 500", func(t *testing.T) {
		ms := &mockTenantMemberService{canManageTenantFn: func(int64, uuid.UUID) (bool, error) {
			return false, errors.New("db error")
		}}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("not a member returns 403", func(t *testing.T) {
		ms := &mockTenantMemberService{canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		ms := &mockTenantMemberService{canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(badJSONReq(t, http.MethodPut, "/"), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		ms := &mockTenantMemberService{canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"name": ""}), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, ms).Update(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		ts := &mockTenantService{updateFn: func(uuid.UUID, string, string, string, string) (*TenantServiceDataResult, error) {
			return nil, errors.New("update error")
		}}
		ms := &mockTenantMemberService{canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Update(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{updateFn: func(id uuid.UUID, n, dn, desc, s string) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: n}, nil
		}}
		ms := &mockTenantMemberService{canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return true, nil }}
		r := withUser(withChiParam(jsonReq(t, http.MethodPut, "/", validBody), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).Update(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_SetStatus(t *testing.T) {
	t.Run("no user returns 401", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "tenant_uuid", "bad"))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withUser(withChiParam(badJSONReq(t, http.MethodPatch, "/"), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		r := withUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "deleted"}), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		ts := &mockTenantService{setStatusByUUIDFn: func(uuid.UUID, string) (*TenantServiceDataResult, error) {
			return nil, errors.New("status error")
		}}
		r := withUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{setStatusByUUIDFn: func(id uuid.UUID, s string) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Status: s}, nil
		}}
		r := withUser(withChiParam(jsonReq(t, http.MethodPatch, "/", map[string]any{"status": "active"}), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).SetStatus(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_Delete(t *testing.T) {
	systemSvc := func() *mockTenantService {
		return &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: tenantID}, nil
		}}
	}

	t.Run("no user returns 401", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String())
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Delete(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", "bad"))
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).Delete(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetSystem error returns 500", func(t *testing.T) {
		ts := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return nil, errors.New("db error")
		}}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).Delete(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("non-system tenant returns 403", func(t *testing.T) {
		ts := &mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: tenantID + 999}, nil
		}}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).Delete(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("GetByUUID error returns 404", func(t *testing.T) {
		ts := systemSvc()
		ts.getByUUIDFn = func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).Delete(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("system tenant returns 403", func(t *testing.T) {
		ts := systemSvc()
		ts.getByUUIDFn = func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{IsSystem: true}, nil
		}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).Delete(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("DeleteByUUID error returns 500", func(t *testing.T) {
		ts := systemSvc()
		ts.getByUUIDFn = func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{}, nil
		}
		ts.deleteByUUIDFn = func(uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("delete error") }
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).Delete(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := systemSvc()
		ts.getByUUIDFn = func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{IsSystem: false}, nil
		}
		ts.deleteByUUIDFn = func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{Name: "t1"}, nil
		}
		r := withTenantAndUser(withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String()))
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).Delete(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_GetMembers(t *testing.T) {
	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil)
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", "bad")
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10&sort_order=bad", nil), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetByUUID error returns 404", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).GetMembers(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("ListByTenant error returns 500", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		ms := &mockTenantMemberService{listByTenantFn: func(TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error) {
			return nil, errors.New("db error")
		}}
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).GetMembers(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success with no rows covers empty result branch", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		ms := &mockTenantMemberService{listByTenantFn: func(TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error) {
			return &TenantMemberServiceListResult{
				Data:  nil,
				Total: 0,
				Page:  1,
				Limit: 10,
			}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).GetMembers(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("success with user member covers toTenantMemberResponseDTO User branch", func(t *testing.T) {
		userResult := &MemberUser{Username: "alice"}
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		ms := &mockTenantMemberService{listByTenantFn: func(TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error) {
			return &TenantMemberServiceListResult{
				Data:  []TenantMemberServiceDataResult{{Role: "owner", User: userResult}},
				Total: 1,
				Page:  1,
				Limit: 10,
			}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodGet, "/?page=1&limit=10", nil), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).GetMembers(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_AddMember(t *testing.T) {
	memberUUID := uuid.New()
	validBody := map[string]any{"user_id": memberUUID.String(), "role": "member"}

	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPost, "/", validBody)
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", "bad")
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		r := withChiParam(badJSONReq(t, http.MethodPost, "/"), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPost, "/", map[string]any{"user_id": memberUUID.String()}), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetByUUID error returns 404", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).AddMember(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("CreateByUserUUID error returns 400", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{}, nil
		}}
		ms := &mockTenantMemberService{createByUserUUIDFn: func(int64, uuid.UUID, string) (*TenantMemberServiceDataResult, error) {
			return nil, errValidation
		}}
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).AddMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 201", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{}, nil
		}}
		ms := &mockTenantMemberService{createByUserUUIDFn: func(int64, uuid.UUID, string) (*TenantMemberServiceDataResult, error) {
			return &TenantMemberServiceDataResult{Role: "member"}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodPost, "/", validBody), "tenant_uuid", testResourceUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).AddMember(w, r)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestTenantHandler_UpdateMemberRole(t *testing.T) {
	memberUUID := uuid.New()

	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := jsonReq(t, http.MethodPut, "/", map[string]any{"role": "admin"})
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid member UUID but missing tenant UUID returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "admin"}), "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("tenant lookup error returns 404", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "admin"}), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "admin"}), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", "bad")
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid tenant UUID format returns 400 before member UUID parsing", func(t *testing.T) {
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "admin"}), "tenant_member_uuid", "bad")
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad JSON returns 400", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		r := withChiParam(badJSONReq(t, http.MethodPut, "/"), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": ""}), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		ms := &mockTenantMemberService{updateRoleFn: func(int64, uuid.UUID, string) (*TenantMemberServiceDataResult, error) {
			return nil, errValidation
		}}
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "owner"}), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		ms := &mockTenantMemberService{updateRoleFn: func(_ int64, id uuid.UUID, role string) (*TenantMemberServiceDataResult, error) {
			return &TenantMemberServiceDataResult{Role: role}, nil
		}}
		r := withChiParam(jsonReq(t, http.MethodPut, "/", map[string]any{"role": "member"}), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).UpdateMemberRole(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTenantHandler_RemoveMember(t *testing.T) {
	memberUUID := uuid.New()

	t.Run("empty UUID param returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/", nil)
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid member UUID but missing tenant UUID returns 400", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("tenant lookup error returns 404", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errNotFound
		}}
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid UUID format returns 400", func(t *testing.T) {
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", "bad")
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(nil, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns 400", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		ms := &mockTenantMemberService{deleteByUUIDFn: func(int64, uuid.UUID) error { return errValidation }}
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, ms).RemoveMember(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		ts := &mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		r := withChiParam(httptest.NewRequest(http.MethodDelete, "/", nil), "tenant_uuid", testResourceUUID.String())
		r = withChiParam(r, "tenant_member_uuid", memberUUID.String())
		r = withUser(r)
		w := httptest.NewRecorder()
		newTenantHandler(ts, nil).RemoveMember(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
