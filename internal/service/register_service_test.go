package service

import (
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
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
	tenantUser   *mockTenantUserRepo
}

// defaultRegPublicMocks returns mocks configured for a successful RegisterPublic.
func defaultRegPublicMocks() *regMocks {
	domain := "example.com"
	identifier := "test-client"
	return &regMocks{
		client: &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
				return &model.Client{
					ClientID:   1,
					Status:     model.StatusActive,
					Domain:     &domain,
					Identifier: &identifier,
					IdentityProvider: &model.IdentityProvider{
						Identifier: "test-provider",
						TenantID:   1,
						Tenant:     &model.Tenant{TenantID: 1},
					},
				}, nil
			},
		},
		idp: &mockIdentityProviderRepo{
			findByIdentifierFn: func(_ string) (*model.IdentityProvider, error) {
				return &model.IdentityProvider{TenantID: 1}, nil
			},
		},
		user: &mockUserRepo{
			findByUsernameFn: func(_ string) (*model.User, error) { return nil, nil },
			findByEmailFn:    func(_ string) (*model.User, error) { return nil, nil },
			findByPhoneFn:    func(_ string) (*model.User, error) { return nil, nil },
			createFn:         func(u *model.User) (*model.User, error) { u.UserID = 1; return u, nil },
		},
		userIdentity: &mockUserIdentityRepo{
			createFn: func(ui *model.UserIdentity) (*model.UserIdentity, error) { return ui, nil },
		},
		role: &mockRoleRepo{
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return &repository.PaginationResult[model.Role]{Data: []model.Role{{RoleID: 1}}}, nil
			},
		},
		userRole:   &mockUserRoleRepo{},
		userToken:  &mockUserTokenRepo{},
		invite:     &mockInviteRepo{},
		tenantUser: &mockTenantUserRepo{},
	}
}

// defaultRegInternalMocks returns mocks for a successful Register (internal) flow.
// Uses FindByClientIDAndIdentityProvider path when clientID and providerID are provided.
func defaultRegInternalMocks() *regMocks {
	domain := "example.com"
	identifier := "test-client"
	return &regMocks{
		client: &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
				return &model.Client{
					ClientID:   1,
					Status:     model.StatusActive,
					Domain:     &domain,
					Identifier: &identifier,
					IdentityProvider: &model.IdentityProvider{
						Identifier: "test-provider",
						TenantID:   1,
						Tenant:     &model.Tenant{TenantID: 1},
					},
				}, nil
			},
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{
					ClientID:   1,
					Status:     model.StatusActive,
					Domain:     &domain,
					Identifier: &identifier,
					IdentityProvider: &model.IdentityProvider{
						Identifier: "test-provider",
						TenantID:   1,
						Tenant:     &model.Tenant{TenantID: 1},
					},
				}, nil
			},
		},
		idp: &mockIdentityProviderRepo{},
		user: &mockUserRepo{
			findByUsernameFn: func(_ string) (*model.User, error) { return nil, nil },
			createFn:         func(u *model.User) (*model.User, error) { u.UserID = 1; return u, nil },
		},
		userIdentity: &mockUserIdentityRepo{
			createFn: func(ui *model.UserIdentity) (*model.UserIdentity, error) { return ui, nil },
		},
		role: &mockRoleRepo{
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return &repository.PaginationResult[model.Role]{Data: []model.Role{{RoleID: 1}}}, nil
			},
		},
		userRole:   &mockUserRoleRepo{},
		userToken:  &mockUserTokenRepo{},
		invite:     &mockInviteRepo{},
		tenantUser: &mockTenantUserRepo{},
	}
}

// ---------------------------------------------------------------------------
// findDefaultRole
// ---------------------------------------------------------------------------

func TestRegisterService_FindDefaultRole(t *testing.T) {
	t.Run("FindPaginated error", func(t *testing.T) {
		svc := &registerService{}
		roleRepo := &mockRoleRepo{
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
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
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return &repository.PaginationResult[model.Role]{
					Data: []model.Role{{RoleID: 42}},
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
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return &repository.PaginationResult[model.Role]{Data: []model.Role{}}, nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
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
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return &repository.PaginationResult[model.Role]{Data: []model.Role{}}, nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
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
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return &repository.PaginationResult[model.Role]{Data: []model.Role{}}, nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
				return &model.Role{RoleID: 99}, nil
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
	util.InitRateLimiter(rdb)
	require.NoError(t, mr.Set("rl:lock:"+identifier, "1"))
	return func() {
		util.InitRateLimiter(nil)
		rdb.Close()
		mr.Close()
	}
}

// ---------------------------------------------------------------------------
// RegisterPublic – rate limit
// ---------------------------------------------------------------------------

func TestRegisterPublic_RateLimited(t *testing.T) {
	cleanup := lockedRateLimiterReg(t, "ratelimited-user")
	defer cleanup()

	gormDB, mock := newMockGormDB(t)
	_ = mock
	m := defaultRegPublicMocks()
	svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
		m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
	resp, err := svc.RegisterPublic("ratelimited-user", "F", "P@ss1!", nil, nil, "c", "p")
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
		m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
	resp, err := svc.Register("ratelimited-user2", "F", "P@ss1!", nil, nil, nil, nil)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "locked")
}

// ---------------------------------------------------------------------------
// RegisterPublic
// ---------------------------------------------------------------------------

func TestRegisterService_RegisterPublic(t *testing.T) {
	t.Run("client not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
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
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client inactive", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		domain := "example.com"
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
			return &model.Client{Status: model.StatusInactive, Domain: &domain}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
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
		m.idp.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
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
		m.idp.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "identity provider not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("username lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("username already taken", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return &model.User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
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
		m.user.findByEmailFn = func(_ string) (*model.User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		email := "a@b.com"
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", &email, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("email already registered", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByEmailFn = func(_ string) (*model.User, error) {
			return &model.User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		email := "a@b.com"
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", &email, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "email already registered")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("phone lookup error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByPhoneFn = func(_ string) (*model.User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		phone := "+1234567890"
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, &phone, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("phone already registered", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.findByPhoneFn = func(_ string) (*model.User, error) {
			return &model.User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		phone := "+1234567890"
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, &phone, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "phone number already registered")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.user.createFn = func(_ *model.User) (*model.User, error) {
			return nil, errors.New("create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.userIdentity.createFn = func(_ *model.UserIdentity) (*model.UserIdentity, error) {
			return nil, errors.New("identity error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("findDefaultRole error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.role.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return nil, errors.New("role error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.userRole.createFn = func(_ *model.UserRole) (*model.UserRole, error) {
			return nil, errors.New("role assign error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user token create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.userToken.createFn = func(_ *model.UserToken) (*model.UserToken, error) {
			return nil, errors.New("token error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("generateTokenResponse error", func(t *testing.T) {
		util.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "F", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits but generateTokenResponse fails - no email phone", func(t *testing.T) {
		initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterPublic("u", "Full Name", "P@ss1!", nil, nil, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits but generateTokenResponse fails - with email phone", func(t *testing.T) {
		initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		email := "a@b.com"
		phone := "+1234567890"
		resp, err := svc.RegisterPublic("u", "Full Name", "P@ss1!", &email, &phone, "c", "p")
		require.Error(t, err)
		assert.Nil(t, resp)
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
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
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
		m.client.findDefaultFn = func() (*model.Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client nil or inactive", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.client.findDefaultFn = func() (*model.Client, error) { return nil, nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, nil, nil)
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
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
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
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return nil, errors.New("record not found")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
		// tx commits but generateTokenResponse fails (userIdentitySub empty)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user already exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return &model.User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
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
		m.user.createFn = func(_ *model.User) (*model.User, error) {
			return nil, errors.New("create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.userIdentity.createFn = func(_ *model.UserIdentity) (*model.UserIdentity, error) {
			return nil, errors.New("identity error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tenant user create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.tenantUser.createFn = func(_ *model.TenantUser) (*model.TenantUser, error) {
			return nil, errors.New("tenant user error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("findDefaultRole error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.role.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return nil, errors.New("role error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.userRole.createFn = func(_ *model.UserRole) (*model.UserRole, error) {
			return nil, errors.New("role assign error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user token create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.userToken.createFn = func(_ *model.UserToken) (*model.UserToken, error) {
			return nil, errors.New("token error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("generateTokenResponse error", func(t *testing.T) {
		util.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, &cid, &pid)
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
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		email := "a@b.com"
		phone := "+1234567890"
		resp, err := svc.Register("u", "F", "P@ss1!", &email, &phone, &cid, &pid)
		// userIdentitySub is never set in tx → generateTokenResponse fails
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits with default client no email no phone", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.Register("u", "F", "P@ss1!", nil, nil, nil, nil)
		// userIdentitySub is never set in tx → generateTokenResponse fails
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// RegisterInvite (internal)
// ---------------------------------------------------------------------------

func TestRegisterService_RegisterInvite(t *testing.T) {
	cid := "client-id"
	pid := "provider-id"

	validInvite := func() *model.Invite {
		future := time.Now().Add(time.Hour)
		return &model.Invite{
			InviteUUID:   uuid.New(),
			InvitedEmail: "invite@test.com",
			Status:       model.StatusPending,
			ExpiresAt:    &future,
			Roles:        []model.Role{{RoleID: 10}},
		}
	}

	t.Run("FindByClientIDAndIdentityProvider error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
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
		m.client.findDefaultFn = func() (*model.Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client nil or inactive", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.client.findDefaultFn = func() (*model.Client, error) { return nil, nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "auth client not found or inactive")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite FindByToken error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid invite token")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite nil", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invite token is invalid or expired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite not pending", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return &model.Invite{Status: "accepted"}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invite token is invalid or expired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite expired", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		past := time.Now().Add(-time.Hour)
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return &model.Invite{Status: model.StatusPending, ExpiresAt: &past}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invite token is invalid or expired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByUsername non-record-not-found error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user already exists", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return &model.User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.createFn = func(_ *model.User) (*model.User, error) {
			return nil, errors.New("create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.userIdentity.createFn = func(_ *model.UserIdentity) (*model.UserIdentity, error) {
			return nil, errors.New("identity error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tenant user create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.tenantUser.createFn = func(_ *model.TenantUser) (*model.TenantUser, error) {
			return nil, errors.New("tenant user error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("findDefaultRole error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.role.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return nil, errors.New("role error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("default user role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.userRole.createFn = func(_ *model.UserRole) (*model.UserRole, error) {
			return nil, errors.New("role assign error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		callCount := 0
		m.userRole.createFn = func(_ *model.UserRole) (*model.UserRole, error) {
			callCount++
			if callCount > 1 { // first call is default role, second is invite role
				return nil, errors.New("invite role error")
			}
			return &model.UserRole{}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("MarkAsUsed error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.invite.markAsUsedFn = func(_ uuid.UUID) error {
			return errors.New("mark error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("generateTokenResponse error", func(t *testing.T) {
		util.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits with clientID and providerID", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", &cid, &pid)
		// userIdentitySub is never set in tx → generateTokenResponse fails
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tx commits with default client", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegInternalMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvite("u", "P@ss1!", "token", nil, nil)
		// userIdentitySub is never set in tx → generateTokenResponse fails
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// RegisterInvitePublic
// ---------------------------------------------------------------------------

func TestRegisterService_RegisterInvitePublic(t *testing.T) {
	validInvite := func() *model.Invite {
		future := time.Now().Add(time.Hour)
		return &model.Invite{
			InviteUUID:   uuid.New(),
			InvitedEmail: "invite@test.com",
			Status:       model.StatusPending,
			ExpiresAt:    &future,
			Roles:        []model.Role{{RoleID: 10}},
		}
	}

	t.Run("client repo error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client nil or inactive", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.client.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.idp.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.idp.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return nil, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return &model.Invite{Status: "accepted"}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) {
			return &model.Invite{Status: model.StatusPending, ExpiresAt: &past}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("username already taken", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.findByUsernameFn = func(_ string) (*model.User, error) {
			return &model.User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.findByEmailFn = func(_ string) (*model.User, error) {
			return nil, errors.New("db error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invited email already registered", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.findByEmailFn = func(_ string) (*model.User, error) {
			return &model.User{UserID: 99}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.user.createFn = func(_ *model.User) (*model.User, error) {
			return nil, errors.New("create error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.userIdentity.createFn = func(_ *model.UserIdentity) (*model.UserIdentity, error) {
			return nil, errors.New("identity error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tenant user create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.tenantUser.createFn = func(_ *model.TenantUser) (*model.TenantUser, error) {
			return nil, errors.New("tenant user error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("findDefaultRole error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.role.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return nil, errors.New("role error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("default user role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.userRole.createFn = func(_ *model.UserRole) (*model.UserRole, error) {
			return nil, errors.New("role assign error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByUserIDAndRoleID error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.userRole.findByUserIDAndRoleIDFn = func(_, _ int64) (*model.UserRole, error) {
			return nil, errors.New("lookup error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite role already exists - skipped", func(t *testing.T) {
		initTestJWTKeysService(t)
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.userRole.findByUserIDAndRoleIDFn = func(_, _ int64) (*model.UserRole, error) {
			return &model.UserRole{UserID: 1, RoleID: 10}, nil // already exists
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		// tx commits but generateTokenResponse fails (userIdentitySub empty)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite role create error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		createCount := 0
		m.userRole.createFn = func(_ *model.UserRole) (*model.UserRole, error) {
			createCount++
			if createCount > 1 { // first = default role, second = invite role
				return nil, errors.New("invite role error")
			}
			return &model.UserRole{}, nil
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("MarkAsUsed error", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		m.invite.markAsUsedFn = func(_ uuid.UUID) error {
			return errors.New("mark error")
		}
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("generateTokenResponse error", func(t *testing.T) {
		util.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		m := defaultRegPublicMocks()
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
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
		m.invite.findByTokenFn = func(_ string) (*model.Invite, error) { return validInvite(), nil }
		svc := NewRegistrationService(gormDB, m.client, m.user, m.userRole, m.userToken,
			m.userIdentity, m.role, m.invite, m.idp, m.tenantUser)
		resp, err := svc.RegisterInvitePublic("u", "P@ss1!", "c", "p", "token")
		// userIdentitySub is never set in tx → generateTokenResponse fails
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// generateTokenResponse (tested directly)
// ---------------------------------------------------------------------------

func TestRegisterService_GenerateTokenResponse(t *testing.T) {
	domain := "example.com"
	identifier := "test-client"
	client := &model.Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &model.IdentityProvider{
			Identifier: "test-provider",
		},
	}

	t.Run("GenerateAccessToken error", func(t *testing.T) {
		util.ResetJWTKeys()
		defer initTestJWTKeysService(t)

		svc := &registerService{}
		resp, err := svc.generateTokenResponse("sub", &model.User{}, client)
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("success", func(t *testing.T) {
		initTestJWTKeysService(t)

		svc := &registerService{}
		resp, err := svc.generateTokenResponse("sub", &model.User{
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
