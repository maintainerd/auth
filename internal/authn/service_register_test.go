package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// regMocks bundles every mock repo needed by NewRegistrationService.
type regMocks struct {
	client       *mockClientRepo
	idp          *mockIdentityProviderRepo
	user         *mockUserRepo
	userRole     *mockUserRoleRepo
	userToken    *mockUserTokenRepo
	userIdentity *mockUserIdentityRepo
	role         *mockRoleRepo
	invite       *mockInviteRepo
}

// defaultRegPublicMocks returns mocks configured for a successful RegisterPublic.
func defaultRegPublicMocks() *regMocks {
	domain := "example.com"
	identifier := "test-client"
	return &regMocks{
		client: &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID:   1,
					Status:     shared.StatusActive,
					Domain:     &domain,
					Identifier: &identifier,
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
						TenantID:   1,
						Tenant:     &Tenant{TenantID: 1},
					},
				}, nil
			},
		},
		idp: &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*IdentityProvider, error) {
				return &IdentityProvider{TenantID: 1}, nil
			},
		},
		user: &mockUserRepo{
			findByUsernameFn: func(_ string) (*User, error) { return nil, nil },
			findByEmailFn:    func(_ string) (*User, error) { return nil, nil },
			findByPhoneFn:    func(_ string) (*User, error) { return nil, nil },
			createFn:         func(u *User) (*User, error) { u.UserID = 1; return u, nil },
		},
		userIdentity: &mockUserIdentityRepo{
			createFn: func(ui *UserIdentity) (*UserIdentity, error) { return ui, nil },
		},
		role: &mockRoleRepo{
			findPaginatedFn: func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{Data: []Role{{RoleID: 1}}}, nil
			},
		},
		userRole:  &mockUserRoleRepo{},
		userToken: &mockUserTokenRepo{},
		invite:    &mockInviteRepo{},
	}
}

// defaultRegInternalMocks returns mocks for a successful Register (internal) flow.
// Uses FindByClientIDAndIdentityProvider path when clientID and providerID are provided.
func defaultRegInternalMocks() *regMocks {
	domain := "example.com"
	identifier := "test-client"
	return &regMocks{
		client: &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
				return &Client{
					ClientID:   1,
					Status:     shared.StatusActive,
					Domain:     &domain,
					Identifier: &identifier,
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
						TenantID:   1,
						Tenant:     &Tenant{TenantID: 1},
					},
				}, nil
			},
			findSystemFn: func() (*Client, error) {
				return &Client{
					ClientID:   1,
					Status:     shared.StatusActive,
					Domain:     &domain,
					Identifier: &identifier,
					IdentityProvider: &IdentityProvider{
						Identifier: "test-provider",
						TenantID:   1,
						Tenant:     &Tenant{TenantID: 1},
					},
				}, nil
			},
		},
		idp: &mockIdentityProviderRepo{},
		user: &mockUserRepo{
			findByUsernameFn: func(_ string) (*User, error) { return nil, nil },
			createFn:         func(u *User) (*User, error) { u.UserID = 1; return u, nil },
		},
		userIdentity: &mockUserIdentityRepo{
			createFn: func(ui *UserIdentity) (*UserIdentity, error) { return ui, nil },
		},
		role: &mockRoleRepo{
			findPaginatedFn: func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{Data: []Role{{RoleID: 1}}}, nil
			},
		},
		userRole:  &mockUserRoleRepo{},
		userToken: &mockUserTokenRepo{},
		invite:    &mockInviteRepo{},
	}
}

// ---------------------------------------------------------------------------
// findDefaultRole
// ---------------------------------------------------------------------------

func TestRegisterService_FindDefaultRole(t *testing.T) {
	t.Run("FindPaginated error", func(t *testing.T) {
		svc := &registerService{}
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return nil, errors.New("db error")
			},
		}
		role, err := svc.findDefaultRole(roleRepo, 1)
		require.Error(t, err)
		assert.Nil(t, role)
	})

	t.Run("default role found via FindPaginated", func(t *testing.T) {
		svc := &registerService{}
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{
					Data: []Role{{RoleID: 42}},
				}, nil
			},
		}
		role, err := svc.findDefaultRole(roleRepo, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(42), role.RoleID)
	})

	t.Run("fallback FindByNameAndTenantID error", func(t *testing.T) {
		svc := &registerService{}
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{Data: []Role{}}, nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*Role, error) {
				return nil, errors.New("fallback error")
			},
		}
		role, err := svc.findDefaultRole(roleRepo, 1)
		require.Error(t, err)
		assert.Nil(t, role)
	})

	t.Run("fallback returns nil - no default role", func(t *testing.T) {
		svc := &registerService{}
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{Data: []Role{}}, nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*Role, error) {
				return nil, nil
			},
		}
		role, err := svc.findDefaultRole(roleRepo, 1)
		require.Error(t, err)
		assert.Nil(t, role)
		assert.Contains(t, err.Error(), "no default role found for tenant")
	})

	t.Run("fallback success", func(t *testing.T) {
		svc := &registerService{}
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
				return &PaginationResult[Role]{Data: []Role{}}, nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*Role, error) {
				return &Role{RoleID: 99}, nil
			},
		}
		role, err := svc.findDefaultRole(roleRepo, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(99), role.RoleID)
	})
}

// lockedRateLimiterReg starts a miniredis instance, pre-sets the lock key
// for the given identifier, and returns a cleanup function.
func lockedRateLimiterReg(t *testing.T, identifier string) func() {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	security.InitRateLimiter(rdb)
	require.NoError(t, mr.Set("rl:lock:"+identifier, "1"))
	return func() {
		security.InitRateLimiter(nil)
		_ = rdb.Close()
		mr.Close()
	}
}

// ---------------------------------------------------------------------------
// RegisterPublic – rate limit
// ---------------------------------------------------------------------------

func TestRegisterPublic_RateLimited(t *testing.T) {
	cid, pid := "c", "p"
	cleanup := lockedRateLimiterReg(t, "ratelimited-user")
	defer cleanup()

	gormDB, mock := newMockGormDB(t)
	_ = mock
	m := defaultRegPublicMocks()
	svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
		m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
	resp, err := svc.RegisterPublic(context.Background(), "ratelimited-user", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "locked")
}

// ---------------------------------------------------------------------------
// Register – rate limit
// ---------------------------------------------------------------------------

func TestRegister_RateLimited(t *testing.T) {
	cleanup := lockedRateLimiterReg(t, "ratelimited-user2")
	defer cleanup()

	gormDB, mock := newMockGormDB(t)
	_ = mock
	m := defaultRegInternalMocks()
	svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
		m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
	resp, err := svc.Register(context.Background(), "ratelimited-user2", "F", "P@ssW0rd!", nil, nil, nil, nil)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "locked")
}

// ---------------------------------------------------------------------------
// RegisterPublic
// ---------------------------------------------------------------------------

func TestRegisterService_RegisterPublic(t *testing.T) {
	cid, pid := "c", "p"
	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or inactive auth client")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client repo error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.idp.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
			return nil, errors.New("idp db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "identity provider lookup failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider nil", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.idp.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "identity provider not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate username", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByUsernameFn = func(_ string) (*User, error) {
			return &User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "username already taken")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate email", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByEmailFn = func(_ string) (*User, error) {
			return &User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		email := "dup@test.com"
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", &email, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "email already registered")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate phone", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByPhoneFn = func(_ string) (*User, error) {
			return &User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		phone := "+1234567890"
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, &phone, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "phone number already registered")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("password policy validation error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "weak", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "password must be at least")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.createFn = func(_ *User) (*User, error) {
			return nil, errors.New("create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.userIdentity.createFn = func(_ *UserIdentity) (*UserIdentity, error) {
			return nil, errors.New("identity create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("find default role error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.role.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return nil, errors.New("role error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.userRole.createFn = func(_ *UserRole) (*UserRole, error) {
			return nil, errors.New("role assign error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("true success with token response", func(t *testing.T) {
		initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		m.user.createFn = func(u *User) (*User, error) { u.UserID = 1; u.UserUUID = uuid.New(); return u, nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterPublic(context.Background(), "u", "Full Name", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with email and phone", func(t *testing.T) {
		initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		var capturedEmail, capturedPhone string
		m.user.createFn = func(u *User) (*User, error) {
			u.UserID = 1
			u.UserUUID = uuid.New()
			capturedEmail = u.Email
			capturedPhone = u.Phone
			return u, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		email := "test@example.com"
		phone := "+1234567890"
		resp, err := svc.RegisterPublic(context.Background(), "u", "Full Name", "P@ssW0rd!", &email, &phone, &cid, &pid)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, email, capturedEmail)
		assert.Equal(t, phone, capturedPhone)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

}

// ---------------------------------------------------------------------------
// Register (internal)
// ---------------------------------------------------------------------------

func TestRegisterService_Register(t *testing.T) {
	cid := "client-id"
	pid := "provider-id"

	t.Run("FindByClientIDAndIdentityProvider error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "auth client lookup by client_id and provider_id failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindDefault error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.client.findSystemFn = func() (*Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client nil or inactive", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.client.findSystemFn = func() (*Client, error) { return nil, nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "auth client not found or inactive")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByUsername returns non-record-not-found error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.user.findByUsernameFn = func(_ string) (*User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByUsername returns record-not-found - treated as not found", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		m.user.findByUsernameFn = func(_ string) (*User, error) {
			return nil, errors.New("record not found")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		// With the userIdentitySub bug fixed, token response now succeeds.
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user already exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.user.findByUsernameFn = func(_ string) (*User, error) {
			return &User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "user already exists")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.user.createFn = func(_ *User) (*User, error) {
			return nil, errors.New("create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.userIdentity.createFn = func(_ *UserIdentity) (*UserIdentity, error) {
			return nil, errors.New("identity error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("findDefaultRole error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.role.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return nil, errors.New("role error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.userRole.createFn = func(_ *UserRole) (*UserRole, error) {
			return nil, errors.New("role assign error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("generateTokenResponse error", func(t *testing.T) {
		jwt.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits with clientID providerID email phone", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		email := "a@b.com"
		phone := "+1234567890"
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", &email, &phone, &cid, &pid)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits with default client no email no phone", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.Register(context.Background(), "u", "F", "P@ssW0rd!", nil, nil, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

}

// ---------------------------------------------------------------------------
// RegisterInvitePublic
// ---------------------------------------------------------------------------

func TestRegisterService_RegisterInvitePublic(t *testing.T) {
	validInvite := func() *Invite {
		future := time.Now().Add(time.Hour)
		return &Invite{
			InviteUUID:   uuid.New(),
			InvitedEmail: "invite@test.com",
			Status:       shared.StatusPending,
			ExpiresAt:    &future,
		}
	}

	t.Run("client repo error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client nil or inactive", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*Client, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or inactive auth client")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.idp.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "identity provider lookup failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.idp.findByIdentifierFn = func(_ string) (*IdentityProvider, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "identity provider not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite FindByToken error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid invite token")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite nil", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invite not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite not pending", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) {
			return &Invite{Status: "accepted"}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invite has already been used or is no longer valid")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite expired", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		past := time.Now().Add(-time.Hour)
		m.invite.findByTokenFn = func(_ string) (*Invite, error) {
			return &Invite{Status: shared.StatusPending, ExpiresAt: &past}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invite has expired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("username lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.user.findByUsernameFn = func(_ string) (*User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("username already taken", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.user.findByUsernameFn = func(_ string) (*User, error) {
			return &User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "username already taken")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("email lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.user.findByEmailFn = func(_ string) (*User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invited email already registered", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.user.findByEmailFn = func(_ string) (*User, error) {
			return &User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invited email already registered")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.user.createFn = func(_ *User) (*User, error) {
			return nil, errors.New("create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.userIdentity.createFn = func(_ *UserIdentity) (*UserIdentity, error) {
			return nil, errors.New("identity error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("findDefaultRole error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.role.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return nil, errors.New("role error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("default user role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.userRole.createFn = func(_ *UserRole) (*UserRole, error) {
			return nil, errors.New("role assign error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("MarkAsUsed error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		m.invite.markAsUsedFn = func(_ uuid.UUID) error {
			return errors.New("mark error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("generateTokenResponse error", func(t *testing.T) {
		jwt.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
		resp, err := svc.RegisterInvitePublic(context.Background(), "u", "P@ssW0rd!", "c", "p", "token")
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

}

// ---------------------------------------------------------------------------
// TestRegisterInvitePublic_PasswordPolicyValidationTight
// ---------------------------------------------------------------------------

func TestRegisterInvitePublic_PasswordPolicyValidationTight(t *testing.T) {
	validInvite := func() *Invite {
		future := time.Now().Add(time.Hour)
		return &Invite{
			InviteUUID:   uuid.New(),
			InvitedEmail: "invite@test.com",
			Status:       shared.StatusPending,
			ExpiresAt:    &future,
		}
	}

	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	m := defaultRegPublicMocks()
	m.invite.findByTokenFn = func(_ string) (*Invite, error) { return validInvite(), nil }
	svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
		m.userIdentity, m.role, m.invite, m.idp, nil, nil, nil)
	resp, err := svc.RegisterInvitePublic(context.Background(), "u", "weak", "c", "p", "token")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "password must be at least")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// generateTokenResponse (tested directly)
// ---------------------------------------------------------------------------
// generateTokenResponse (tested directly)
// ---------------------------------------------------------------------------

func TestRegisterService_GenerateTokenResponse(t *testing.T) {
	domain := "example.com"
	identifier := "test-client"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "test-provider",
		},
	}

	t.Run("GenerateAccessToken error", func(t *testing.T) {
		jwt.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		svc := &registerService{}
		resp, err := svc.generateTokenResponse(context.Background(), "sub", &User{}, client)
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)

		svc := &registerService{}
		resp, err := svc.generateTokenResponse(context.Background(), "sub", &User{
			Email:           "test@example.com",
			IsEmailVerified: true,
			Phone:           "+1234567890",
			IsPhoneVerified: true,
		}, client)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.IDToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
	})

}
