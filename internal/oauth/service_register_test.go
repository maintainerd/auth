package oauth

import (
	"context"
	"errors"
	"testing"

	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOAuthRegisterSvc(
	db *gorm.DB,
	clientRepo *mockClientRepo,
	clientURIRepo *mockClientURIRepo,
	tenantRepo *mockTenantRepo,
	authEventSvc *mockAuthEventService,
) OAuthRegisterService {
	return &oauthRegisterService{
		db:               db,
		clientRepo:       clientRepo,
		clientURIRepo:    clientURIRepo,
		tenantRepo:       tenantRepo,
		authEventService: authEventSvc,
	}
}

type mockTenantRepo struct {
	findSystemFn func() (*Tenant, error)
}

func (m *mockTenantRepo) WithTx(_ *gorm.DB) TenantRepository { return m }
func (m *mockTenantRepo) FindSystem() (*Tenant, error) {
	if m.findSystemFn != nil {
		return m.findSystemFn()
	}
	return &Tenant{TenantID: 1}, nil
}
func (m *mockTenantRepo) Create(e *Tenant) (*Tenant, error)              { return e, nil }
func (m *mockTenantRepo) CreateOrUpdate(e *Tenant) (*Tenant, error)      { return e, nil }
func (m *mockTenantRepo) FindAll(_ ...string) ([]Tenant, error)          { return nil, nil }
func (m *mockTenantRepo) FindByUUID(_ any, _ ...string) (*Tenant, error) { return nil, nil }
func (m *mockTenantRepo) FindByUUIDs(_ []string, _ ...string) ([]Tenant, error) {
	return nil, nil
}
func (m *mockTenantRepo) FindByID(_ any, _ ...string) (*Tenant, error) { return nil, nil }
func (m *mockTenantRepo) UpdateByUUID(_, _ any) (*Tenant, error)       { return nil, nil }
func (m *mockTenantRepo) UpdateByID(_, _ any) (*Tenant, error)         { return nil, nil }
func (m *mockTenantRepo) DeleteByUUID(_ any) error                     { return nil }
func (m *mockTenantRepo) DeleteByID(_ any) error                       { return nil }
func (m *mockTenantRepo) Paginate(_ map[string]any, _, _ int, _ ...string) (*PaginationResult[Tenant], error) {
	return nil, nil
}

// ── TestOAuthRegisterService_Register ───────────────────────────────────────

func TestOAuthRegisterService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("duplicate redirect_uri (create error)", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(_ *Client) (*Client, error) {
					return nil, errors.New("pq: duplicate key value violates unique constraint")
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Test Client",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("success", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Test Client",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.ClientID)
		assert.Equal(t, "Test Client", result.ClientName)
		assert.Empty(t, result.ClientSecret)
	})

	t.Run("success with client secret", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Confidential Client",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "client_secret_basic",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.ClientID)
		assert.NotEmpty(t, result.ClientSecret)
		assert.Equal(t, "Confidential Client", result.ClientName)
	})

	t.Run("client_name validated already by handler", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					assert.Equal(t, "Registered Client", c.DisplayName)
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, "Registered Client", result.ClientName)
	})

	// INVERTED. This used to assert that a failing tenantRepo.FindSystem() produced
	// a server_error — i.e. that registration resolved every client into the SYSTEM
	// tenant. It no longer consults the system tenant at all: the tenant comes from
	// the authenticated caller, so a broken FindSystem is irrelevant and
	// registration succeeds into the caller's own tenant.
	t.Run("registration ignores the system tenant and uses the caller's", func(t *testing.T) {
		var savedClient *Client
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					savedClient = c
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return nil, errors.New("tenant not found")
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Test",
			RedirectURIs: []string{"https://example.com/callback"},
		}, 7)
		require.Nil(t, oerr)
		require.NotNil(t, savedClient)
		assert.Equal(t, int64(7), savedClient.TenantID)
	})

	t.Run("no caller tenant is refused", func(t *testing.T) {
		created := false
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					created = true
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Test",
			RedirectURIs: []string{"https://example.com/callback"},
		}, 0)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.False(t, created)
	})

	t.Run("no redirect_uris", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
			&mockClientURIRepo{},
			&mockTenantRepo{},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName: "Test",
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "at least one redirect_uri is required")
	})

	t.Run("multiple redirect_uris", func(t *testing.T) {
		type capturedURI struct {
			uri      string
			clientID int64
		}
		var capturedURIs []capturedURI

		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{
				createFn: func(uri *ClientURI) (*ClientURI, error) {
					capturedURIs = append(capturedURIs, capturedURI{uri: uri.URI, clientID: uri.ClientID})
					return uri, nil
				},
			},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Multi URI Client",
			RedirectURIs:            []string{"https://example.com/callback", "https://example.com/callback2"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Len(t, capturedURIs, 2)
		assert.Equal(t, "https://example.com/callback", capturedURIs[0].uri)
		assert.Equal(t, "https://example.com/callback2", capturedURIs[1].uri)
	})

	t.Run("explicit grant_types and response_types", func(t *testing.T) {
		var savedClient *Client
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					savedClient = c
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Custom Grants Client",
			RedirectURIs: []string{"https://example.com/callback"},
			GrantTypes:   []string{"client_credentials", "refresh_token"},
			// A client_credentials client must declare its scopes: an empty
			// allowlist means "every scope", which is unbounded for a machine
			// credential (ValidateClientOAuthMatrix).
			Scope:                   "orders:read",
			ResponseTypes:           []string{"token"},
			TokenEndpointAuthMethod: "client_secret_basic",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, []string{"client_credentials", "refresh_token"}, result.GrantTypes)
		assert.Equal(t, []string{"token"}, result.ResponseTypes)
		require.NotNil(t, savedClient)
		assert.ElementsMatch(t, []string{"client_credentials", "refresh_token"}, []string(savedClient.GrantTypes))
		assert.ElementsMatch(t, []string{"token"}, []string(savedClient.ResponseTypes))
		assert.ElementsMatch(t, []string{"orders:read"}, []string(savedClient.AllowedScopes))
	})

	t.Run("a client_credentials registration with no scope is refused", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) { c.ClientID = 10; return c, nil },
			},
			&mockClientURIRepo{},
			&mockTenantRepo{},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Unbounded M2M",
			RedirectURIs:            []string{"https://example.com/callback"},
			GrantTypes:              []string{"client_credentials"},
			TokenEndpointAuthMethod: "client_secret_basic",
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("client URI create error — registration succeeds", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{
				createFn: func(uri *ClientURI) (*ClientURI, error) {
					return nil, errors.New("uri create error")
				},
			},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Test Client",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.ClientID)
		assert.Equal(t, "Test Client", result.ClientName)
	})

	// INVERTED, same reason as above: a nil system tenant used to abort
	// registration because the system tenant WAS the registration tenant.
	t.Run("a nil system tenant no longer blocks registration", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return nil, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Test",
			RedirectURIs: []string{"https://example.com/callback"},
		}, 1)
		require.Nil(t, oerr)
	})

	t.Run("a forbidden redirect scheme is refused before anything is written", func(t *testing.T) {
		created := false
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					created = true
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Test",
			RedirectURIs:            []string{"javascript:alert(1)"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.False(t, created)
	})

	t.Run("a grant type outside the dynamic-registration allowlist is refused", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
			&mockClientURIRepo{},
			&mockTenantRepo{},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Test",
			RedirectURIs:            []string{"https://example.com/callback"},
			GrantTypes:              []string{GrantTypeTokenExchange},
			TokenEndpointAuthMethod: "client_secret_basic",
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
	})

	t.Run("client_type is a value the CHECK constraint admits", func(t *testing.T) {
		var savedClient *Client
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					savedClient = c
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{},
			&mockAuthEventService{})

		// "public"/"confidential" — what this used to write — violate
		// chk_clients_client_type, which is why every registration 500'd.
		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Test",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, savedClient)
		assert.Contains(t, []string{
			shared.ClientTypeTraditional, shared.ClientTypeSPA,
			shared.ClientTypeMobile, shared.ClientTypeM2M,
		}, savedClient.ClientType)
	})

	t.Run("GenerateRandomString error for clientID", func(t *testing.T) {
		orig := oauthRegisterGenerateRandomString
		defer func() { oauthRegisterGenerateRandomString = orig }()
		oauthRegisterGenerateRandomString = func(int) (string, error) { return "", errors.New("rand failure") }

		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Test",
			RedirectURIs: []string{"https://example.com/callback"},
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("GenerateRandomString error for clientSecret", func(t *testing.T) {
		orig := oauthRegisterGenerateRandomString
		defer func() { oauthRegisterGenerateRandomString = orig }()
		callCount := 0
		oauthRegisterGenerateRandomString = func(int) (string, error) {
			callCount++
			if callCount == 1 {
				return "valid-client-id-with-24-chars!", nil
			}
			return "", errors.New("rand failure")
		}

		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Secret Client",
			RedirectURIs: []string{"https://example.com/callback"},
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("HashClientSecret error", func(t *testing.T) {
		orig := oauthRegisterHashClientSecret
		defer func() { oauthRegisterHashClientSecret = orig }()
		oauthRegisterHashClientSecret = func(context.Context, string) (string, error) {
			return "", errors.New("hash failure")
		}

		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Secret Client",
			RedirectURIs: []string{"https://example.com/callback"},
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("empty token_endpoint_auth_method defaults to secret", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:   "Default Auth Client",
			RedirectURIs: []string{"https://example.com/callback"},
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.ClientID)
		assert.NotEmpty(t, result.ClientSecret)
	})

	t.Run("empty grant and response types use defaults", func(t *testing.T) {
		var savedClient *Client
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					savedClient = c
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Default Grants Client",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, []string{GrantTypeAuthorizationCode}, result.GrantTypes)
		assert.Equal(t, []string{"code"}, result.ResponseTypes)
		require.NotNil(t, savedClient)
		assert.Equal(t, pq.StringArray{GrantTypeAuthorizationCode}, savedClient.GrantTypes)
		assert.Equal(t, pq.StringArray{"code"}, savedClient.ResponseTypes)
	})

	t.Run("EncryptAtRest error", func(t *testing.T) {
		orig := crypto.EncryptAtRest
		defer func() { crypto.EncryptAtRest = orig }()
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failure") }

		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
			&mockClientURIRepo{},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Secret Client",
			RedirectURIs:            []string{"https://example.com/callback"},
			TokenEndpointAuthMethod: "client_secret_basic",
		}, 1)
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("redirect URI sanitizes to empty is skipped", func(t *testing.T) {
		var createdURIs []string
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{
				createFn: func(c *Client) (*Client, error) {
					c.ClientID = 10
					return c, nil
				},
			},
			&mockClientURIRepo{
				createFn: func(uri *ClientURI) (*ClientURI, error) {
					createdURIs = append(createdURIs, uri.URI)
					return uri, nil
				},
			},
			&mockTenantRepo{
				findSystemFn: func() (*Tenant, error) {
					return &Tenant{TenantID: 1, Status: "active"}, nil
				},
			},
			&mockAuthEventService{})

		result, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName:              "Test Client",
			RedirectURIs:            []string{"https://example.com/callback", ""},
			TokenEndpointAuthMethod: "none",
		}, 1)
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Len(t, createdURIs, 1)
		assert.Equal(t, "https://example.com/callback", createdURIs[0])
	})
}
