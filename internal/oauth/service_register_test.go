package oauth

import (
	"context"
	"errors"
	"testing"

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
		})
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
		})
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
		})
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
		})
		require.Nil(t, oerr)
		require.NotNil(t, result)
		assert.Equal(t, "Registered Client", result.ClientName)
	})

	t.Run("tenant not found", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
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
		})
		require.NotNil(t, oerr)
		assert.Equal(t, "server_error", oerr.Code)
	})

	t.Run("no redirect_uris", func(t *testing.T) {
		svc := newOAuthRegisterSvc(nil,
			&mockClientRepo{},
			&mockClientURIRepo{},
			&mockTenantRepo{},
			&mockAuthEventService{})

		_, oerr := svc.Register(ctx, OAuthClientRegistrationRequestDTO{
			ClientName: "Test",
		})
		require.NotNil(t, oerr)
		assert.Equal(t, "invalid_request", oerr.Code)
		assert.Contains(t, oerr.Description, "at least one redirect_uri is required")
	})
}
