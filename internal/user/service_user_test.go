package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// helper: build a full UserService with all repos wired
// ---------------------------------------------------------------------------

func fullUserSvc(
	t *testing.T,
	userRepo *mockUserRepo,
	uiRepo *mockUserIdentityRepo,
	urRepo *mockUserRoleRepo,
	roleRepo *mockRoleRepo,
	tenantRepo *mockTenantRepo,
	idpRepo *mockIdentityProviderRepo,
	clientRepo *mockClientRepo,
) (*gorm.DB, UserService) {
	t.Helper()
	db, _ := newMockGormDB(t)
	svc := NewUserService(db, userRepo, uiRepo, urRepo, roleRepo, tenantRepo, idpRepo, clientRepo, cache.NopInvalidator{}, nil, nil, nil, nil, nil)
	return db, svc
}

func fullUserSvcWithMock(
	t *testing.T,
	userRepo *mockUserRepo,
	uiRepo *mockUserIdentityRepo,
	urRepo *mockUserRoleRepo,
	roleRepo *mockRoleRepo,
	tenantRepo *mockTenantRepo,
	idpRepo *mockIdentityProviderRepo,
	clientRepo *mockClientRepo,
) (*gorm.DB, sqlmock.Sqlmock, UserService) {
	t.Helper()
	db, mock := newMockGormDB(t)
	svc := NewUserService(db, userRepo, uiRepo, urRepo, roleRepo, tenantRepo, idpRepo, clientRepo, cache.NopInvalidator{}, nil, nil, nil, nil, nil)
	return db, mock, svc
}

func defaultMocks() (*mockUserRepo, *mockUserIdentityRepo, *mockUserRoleRepo, *mockRoleRepo, *mockTenantRepo, *mockIdentityProviderRepo, *mockClientRepo) {
	return &mockUserRepo{}, &mockUserIdentityRepo{}, &mockUserRoleRepo{}, &mockRoleRepo{},
		&mockTenantRepo{}, &mockIdentityProviderRepo{}, &mockClientRepo{}
}

// expectCredentialRevocation pins the three UPDATEs revokeLiveCredentials issues
// — the canonical session store, the OAuth refresh tokens, and the legacy
// user_tokens session rows. Ordered expectations inside Begin/Commit are how the
// tests prove the eviction commits or rolls back WITH the status change, rather
// than on its own connection.
func expectCredentialRevocation(mock sqlmock.Sqlmock) {
	mock.ExpectExec(`UPDATE "user_sessions"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "oauth_refresh_tokens"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "user_tokens"`).WillReturnResult(sqlmock.NewResult(0, 1))
}

// User with tenant access (tenantID=1) and default-tenant identity for ValidateTenantAccess
func userWithAccess(userID int64, tenantID int64) *User {
	return &User{
		UserID:   userID,
		UserUUID: uuid.New(),
		UserIdentities: []UserIdentity{{
			TenantID: tenantID,
			Tenant:   &Tenant{TenantID: tenantID, IsSystem: true},
		}},
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestUserService_Get(t *testing.T) {
	t.Run("invalid role UUID", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		bad := "not-a-uuid"
		_, err := svc.Get(context.Background(), UserServiceGetFilter{RoleUUID: &bad, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role UUID")
	})

	t.Run("role not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		rid := uuid.New().String()
		_, err := svc.Get(context.Background(), UserServiceGetFilter{RoleUUID: &rid, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindPaginated error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findPaginatedFn = func(_ UserRepositoryGetFilter) (*PaginationResult[User], error) {
			return nil, errors.New("db error")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.Get(context.Background(), UserServiceGetFilter{TenantID: 1})
		require.Error(t, err)
	})

	t.Run("success with role filter", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 5}, nil
		}
		ur.findPaginatedFn = func(f UserRepositoryGetFilter) (*PaginationResult[User], error) {
			assert.NotNil(t, f.RoleID)
			return &PaginationResult[User]{Data: []User{{UserUUID: uuid.New()}}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		rid := uuid.New().String()
		res, err := svc.Get(context.Background(), UserServiceGetFilter{RoleUUID: &rid, TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("success without role filter", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, err := svc.Get(context.Background(), UserServiceGetFilter{TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("invalid client UUID", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		bad := "not-a-uuid"
		_, err := svc.Get(context.Background(), UserServiceGetFilter{ClientUUID: &bad, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid client UUID")
	})

	t.Run("client not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		cr.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*Client, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		cid := uuid.New().String()
		_, err := svc.Get(context.Background(), UserServiceGetFilter{ClientUUID: &cid, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client not found")
	})

	t.Run("with client filter success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		cr.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*Client, error) {
			return &Client{ClientID: 5}, nil
		}
		ur.findPaginatedFn = func(f UserRepositoryGetFilter) (*PaginationResult[User], error) {
			assert.NotNil(t, f.ClientID)
			return &PaginationResult[User]{Data: []User{}, Total: 0, Page: 1, Limit: 10, TotalPages: 0}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		cid := uuid.New().String()
		res, err := svc.Get(context.Background(), UserServiceGetFilter{ClientUUID: &cid, TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestUserService_GetByUUID(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.GetByUUID(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindByUUID db error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, errors.New("db error") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.GetByUUID(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.GetByUUID(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, err := svc.GetByUUID(context.Background(), uid, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestUserService_Create(t *testing.T) {
	creatorUUID := uuid.New()
	tenantUUID := uuid.New().String()
	email := "test@test.com"

	t.Run("invalid tenant UUID", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, "bad-uuid", creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tenant UUID")
	})

	t.Run("tenant not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("creator user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 { // creator lookup
				return nil, nil
			}
			return nil, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "creator user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 2, UserIdentities: []UserIdentity{}}, nil // no identities → error
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("FindByUsername error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		ur.findByUsernameFn = func(_ string) (*User, error) { return nil, errors.New("username err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username err")
	})

	t.Run("username already exists", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		ur.findByUsernameFn = func(_ string) (*User, error) { return &User{}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username already exists")
	})

	t.Run("FindByEmail error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		ur.findByEmailFn = func(_ string) (*User, error) { return nil, errors.New("email err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", &email, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email err")
	})

	t.Run("email already exists", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		ur.findByEmailFn = func(_ string) (*User, error) { return &User{}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", &email, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email already exists")
	})

	t.Run("Create user error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		ur.createFn = func(_ *User) (*User, error) { return nil, errors.New("create user err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create user err")
	})

	t.Run("FindDefaultByTenantID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return nil, errors.New("no client") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default auth client not found")
	})

	t.Run("Create identity error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		ui.createFn = func(_ *UserIdentity) (*UserIdentity, error) { return nil, errors.New("ident err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ident err")
	})
	t.Run("findDefaultRole error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return nil, errors.New("role paginate err")
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role paginate err")
	})

	t.Run("findDefaultRole fallback — no default or registered", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{Data: []Role{}}, nil
		}
		// FindByNameAndTenantID returns nil → no default role found
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no default role found")
	})

	t.Run("findDefaultRole fallback — FindByNameAndTenantID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{Data: []Role{}}, nil
		}
		rr.findByNameAndTenantIDFn = func(_ string, _ int64) (*Role, error) { return nil, errors.New("name err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("password validation failure", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "short", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password must")
	})

	t.Run("hash password error", func(t *testing.T) {
		original := userHashPasswordWithPolicy
		userHashPasswordWithPolicy = func(context.Context, []byte, security.PasswordPolicy) ([]byte, error) {
			return nil, errors.New("hash error")
		}
		t.Cleanup(func() { userHashPasswordWithPolicy = original })
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()

		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "hash error")
	})

	t.Run("Create user role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{Data: []Role{{RoleID: 1}}}, nil
		}
		urr.createFn = func(_ *UserRole) (*UserRole, error) { return nil, errors.New("ur create err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ur create err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount <= 1 {
				return userWithAccess(2, 1), nil // creator
			}
			return nil, errors.New("fetch err") // final fetch
		}
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{Data: []Role{{RoleID: 1}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success with email and phone", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		phone := "555"
		var createdUser *User
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount <= 1 {
				return userWithAccess(2, 1), nil // creator
			}
			return &User{UserUUID: uuid.New(), Username: "user"}, nil // fetch result
		}
		ur.createFn = func(u *User) (*User, error) {
			u.UserID = 10
			createdUser = u
			return u, nil
		}
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{Data: []Role{{RoleID: 1}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.Create(context.Background(), "user", &email, &phone, "P@ssW0rd1Long", "active", datatypes.JSON([]byte("{}")), tenantUUID, creatorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
		require.NotNil(t, createdUser)
		assert.True(t, createdUser.ForcePasswordChange)
		require.NotNil(t, createdUser.TemporaryPasswordExpiresAt)
		require.NotNil(t, createdUser.PasswordChangedAt)
		assert.WithinDuration(t, createdUser.PasswordChangedAt.Add(72*time.Hour), *createdUser.TemporaryPasswordExpiresAt, time.Minute)
	})

	t.Run("findDefaultRole — fallback to registered role success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return &Tenant{TenantID: 1}, nil }
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount <= 1 {
				return userWithAccess(2, 1), nil
			}
			return &User{UserUUID: uuid.New()}, nil
		}
		cr.findDefaultByTenantIDFn = func(_ int64) (*Client, error) { return &Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{Data: []Role{}}, nil // no default
		}
		rr.findByNameAndTenantIDFn = func(_ string, _ int64) (*Role, error) {
			return &Role{RoleID: 5}, nil // fallback registered
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.Create(context.Background(), "user", nil, nil, "P@ssW0rd1Long", "active", nil, tenantUUID, creatorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserService_Update(t *testing.T) {
	uid := uuid.New()
	updaterUUID := uuid.New()
	tenantID := int64(1)

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("updater user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "old", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return nil, nil // updater not found
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "old", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "updater user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "old", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return &User{UserID: 2, UserIdentities: []UserIdentity{}}, nil // updater with no identities
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "old", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("username change → FindByUsername error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "old", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.findByUsernameFn = func(_ string) (*User, error) { return nil, errors.New("uname err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "new-name", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "uname err")
	})

	t.Run("username conflict", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "old", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.findByUsernameFn = func(_ string) (*User, error) { return &User{UserID: 999}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "new-name", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username already exists")
	})

	t.Run("email change → FindByEmail error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "u", Email: "old@t.com", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		newEmail := "new@t.com"
		ur.findByEmailFn = func(_ string) (*User, error) { return nil, errors.New("email err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", &newEmail, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email err")
	})

	t.Run("email conflict", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "u", Email: "old@t.com", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		newEmail := "new@t.com"
		ur.findByEmailFn = func(_ string) (*User, error) { return &User{UserID: 999}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", &newEmail, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email already exists")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "u", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*User, error) { return nil, errors.New("update err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "u", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return nil, errors.New("fetch err")
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success with all fields", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "u", Email: "old@t.com", UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return &User{UserUUID: uid, Username: "u"}, nil // fetch after update
		}
		newEmail := "new@t.com"
		phone := "555"
		var written map[string]any
		ur.updateByUUIDFn = func(_ any, data any) (*User, error) {
			written, _ = data.(map[string]any)
			return &User{UserUUID: uid, Username: "u"}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		// Rewriting the sign-in address is a credential change, so the eviction runs.
		expectCredentialRevocation(mock)
		mock.ExpectCommit()
		res, err := svc.Update(context.Background(), uid, tenantID, "u", &newEmail, &phone, "active", datatypes.JSON([]byte("{}")), updaterUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)

		// An admin rewriting the email inherited the OLD address's verification.
		// That turned user:update into an account-takeover primitive: point the
		// account at an attacker inbox and the still-set flag let the recovery
		// flows treat it as proven. The write must clear both halves.
		require.NotNil(t, written, "the update must be a map, not a struct: GORM drops false/nil from Updates(struct)")
		assert.Equal(t, false, written["is_email_verified"])
		assert.Nil(t, written["email_verified_at"])
		assert.Equal(t, false, written["is_phone_verified"])
		assert.Nil(t, written["phone_verified_at"])
	})

	t.Run("an unchanged email leaves the verified flag alone", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "u", Email: "same@t.com", IsEmailVerified: true, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return &User{UserUUID: uid, Username: "u"}, nil
		}
		sameEmail := "same@t.com"
		var written map[string]any
		ur.updateByUUIDFn = func(_ any, data any) (*User, error) {
			written, _ = data.(map[string]any)
			return &User{UserUUID: uid, Username: "u"}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectCommit()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", &sameEmail, nil, "active", nil, updaterUUID)
		require.NoError(t, err)
		_, cleared := written["is_email_verified"]
		assert.False(t, cleared, "a no-op email write must not un-verify an address the user really does control")
	})

	t.Run("deactivating through the general update evicts too", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, Username: "u", Status: shared.StatusActive, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return &User{UserUUID: uid, Username: "u"}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		expectCredentialRevocation(mock)
		mock.ExpectCommit()
		_, err := svc.Update(context.Background(), uid, tenantID, "u", nil, nil, shared.StatusSuspended, nil, updaterUUID)
		require.NoError(t, err)
		// Routing round PATCH /status must not be a way to keep a disabled user
		// signed in.
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// SetStatus
// ---------------------------------------------------------------------------

func TestUserService_SetStatus(t *testing.T) {
	uid := uuid.New()
	updaterUUID := uuid.New()
	tenantID := int64(1)

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.SetStatus(context.Background(), uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.SetStatus(context.Background(), uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("updater not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return nil, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.SetStatus(context.Background(), uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "updater user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return &User{UserID: 2, UserIdentities: []UserIdentity{}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.SetStatus(context.Background(), uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("SetStatus error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.setStatusFn = func(_ uuid.UUID, _ string) error { return errors.New("status err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.SetStatus(context.Background(), uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return nil, errors.New("fetch err")
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		expectCredentialRevocation(mock)
		mock.ExpectRollback()
		_, err := svc.SetStatus(context.Background(), uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return &User{UserUUID: uid, Status: "inactive"}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		// Deactivation used to write the column, emit an event, drop the cache —
		// and leave every session and refresh token live, so the user stayed
		// signed in and could refresh indefinitely. It must evict.
		expectCredentialRevocation(mock)
		mock.ExpectCommit()
		res, err := svc.SetStatus(context.Background(), uid, tenantID, "inactive", updaterUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("reactivating does not evict", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return &User{UserUUID: uid, Status: shared.StatusActive}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectCommit()
		_, err := svc.SetStatus(context.Background(), uid, tenantID, shared.StatusActive, updaterUUID)
		require.NoError(t, err)
		// Re-enabling an account is not a reason to sign anyone out.
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a failed revoke aborts the status change", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE "user_sessions"`).WillReturnError(errors.New("revoke boom"))
		mock.ExpectRollback()
		_, err := svc.SetStatus(context.Background(), uid, tenantID, shared.StatusSuspended, updaterUUID)
		// Fail CLOSED: reporting a user as suspended while they still hold live
		// credentials is the exact failure this prevents, so the whole change rolls
		// back and the operator is told.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to revoke sessions")
	})
}

// ---------------------------------------------------------------------------
// VerifyEmail
// ---------------------------------------------------------------------------

func TestUserService_VerifyEmail(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyEmail(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyEmail(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*User, error) { return nil, errors.New("upd err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyEmail(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upd err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyEmail(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid, IsEmailVerified: true}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, err := svc.VerifyEmail(context.Background(), uid, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// VerifyPhone
// ---------------------------------------------------------------------------

func TestUserService_VerifyPhone(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyPhone(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyPhone(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*User, error) { return nil, errors.New("upd err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyPhone(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upd err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.VerifyPhone(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, err := svc.VerifyPhone(context.Background(), uid, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// CompleteAccount
// ---------------------------------------------------------------------------

func TestUserService_CompleteAccount(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.CompleteAccount(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.CompleteAccount(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*User, error) { return nil, errors.New("upd err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.CompleteAccount(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upd err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.CompleteAccount(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, err := svc.CompleteAccount(context.Background(), uid, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// DeleteByUUID
// ---------------------------------------------------------------------------

func TestUserService_DeleteByUUID(t *testing.T) {
	uid := uuid.New()
	deleterUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.DeleteByUUID(context.Background(), uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.DeleteByUUID(context.Background(), uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("deleter not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return nil, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.DeleteByUUID(context.Background(), uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deleter user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return &User{UserID: 2, UserIdentities: []UserIdentity{}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.DeleteByUUID(context.Background(), uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	for _, tc := range []struct {
		name        string
		isSystem    bool
		errContains string
	}{
		{name: "regular tenant owner cannot be deleted", errContains: "remove their ownership first"},
		{name: "system tenant owner cannot be deleted", isSystem: true, errContains: "cannot delete the owner of the system tenant"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ur, ui, urr, rr, tr, idp, cr := defaultMocks()
			callCount := 0
			ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
				callCount++
				if callCount == 1 {
					return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
				}
				return userWithAccess(2, 1), nil
			}
			_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
			mock.ExpectQuery(`SELECT .+ FROM "tenant_members"`).
				WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "is_system"}).AddRow(1, tc.isSystem))

			_, err := svc.DeleteByUUID(context.Background(), uid, 1, deleterUUID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errContains)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("DeleteByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.deleteByUUIDFn = func(_ any) error { return errors.New("del err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectQuery(`SELECT .+ FROM "tenant_members"`).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "is_system"}))
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.DeleteByUUID(context.Background(), uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1, Tenant: &Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectQuery(`SELECT .+ FROM "tenant_members"`).WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "is_system"}))
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.DeleteByUUID(context.Background(), uid, 1, deleterUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// AssignUserRoles
// ---------------------------------------------------------------------------

func TestUserService_AssignUserRoles(t *testing.T) {
	uid := uuid.New()
	roleUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("authevent.FindByUUID role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, errors.New("role err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role err")
	})

	t.Run("role not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByUserIDAndRoleID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		urr.findByUserIDAndRoleIDFn = func(_, _ int64) (*UserRole, error) { return nil, errors.New("ur find err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ur find err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("role already assigned → skip", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		urr.findByUserIDAndRoleIDFn = func(_, _ int64) (*UserRole, error) { return &UserRole{}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("with token repo revocation", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		mockTokenRepo := &mockUserTokenRepo{}
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		// Inject token repo manually since fullUserSvc passes nil
		svcImpl := svc.(*userService)
		svcImpl.userTokenRepo = mockTokenRepo
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Create user role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		urr.createFn = func(_ *UserRole) (*UserRole, error) { return nil, errors.New("ur create err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ur create err")
	})

	t.Run("success with dedup identities", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid, UserIdentities: []UserIdentity{
				{TenantID: 1, Sub: "same-sub"},
				{TenantID: 1, Sub: "same-sub"},
			}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.AssignUserRoles(context.Background(), uid, []uuid.UUID{roleUUID}, int64(1), uuid.New())
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// RemoveUserRole
// ---------------------------------------------------------------------------

func TestUserService_RemoveUserRole(t *testing.T) {
	uid := uuid.New()
	roleUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("authevent.FindByUUID role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, errors.New("role err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role err")
	})

	t.Run("role not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("cannot revoke super-admin from the tenant owner", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		// "super-admin" == shared.RoleSuperAdmin — the role whose removal from an
		// owner is forbidden (ownership must be transferred first).
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) {
			return &Role{RoleID: 5, TenantID: 1, Name: "super-admin"}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		// The guard queries tenant_members; this user IS the tenant owner (count 1),
		// so the revoke is refused and the tx rolled back with no DELETE.
		mock.ExpectQuery(`tenant_members`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "transfer ownership first")
	})

	t.Run("DeleteByUserIDAndRoleID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		urr.deleteByUserIDAndRoleIDFn = func(_, _ int64) error { return errors.New("del ur err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del ur err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("with token repo revocation", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		mockTokenRepo := &mockUserTokenRepo{}
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			callCount++
			if callCount == 1 {
				return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
			}
			return &User{UserUUID: uid}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*Role, error) { return &Role{RoleID: 5, TenantID: 1}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		svcImpl := svc.(*userService)
		svcImpl.userTokenRepo = mockTokenRepo
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.RemoveUserRole(context.Background(), uid, roleUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// GetUserRoles
// ---------------------------------------------------------------------------

func TestUserService_GetUserRoles(t *testing.T) {
	uid := uuid.New()
	tenantID := int64(1)
	filter := GetUserRolesFilter{Page: 1, Limit: 10}

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserRoles(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindByUUID db error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, errors.New("db error") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserRoles(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserRoles(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("FindRolesPaginated error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(1, tenantID), nil }
		ur.findRolesPaginatedFn = func(_ GetUserRolesFilter) (*PaginationResult[Role], error) { return nil, errors.New("roles err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserRoles(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "roles err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(1, tenantID), nil }
		ur.findRolesPaginatedFn = func(_ GetUserRolesFilter) (*PaginationResult[Role], error) {
			return &PaginationResult[Role]{
				Data:  []Role{{RoleUUID: uuid.New(), Name: "editor"}},
				Total: 1,
			}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, total, err := svc.GetUserRoles(context.Background(), uid, tenantID, filter)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, int64(1), total)
	})
}

// ---------------------------------------------------------------------------
// GetUserIdentities
// ---------------------------------------------------------------------------

func TestUserService_GetUserIdentities(t *testing.T) {
	uid := uuid.New()
	tenantID := int64(1)
	filter := GetUserIdentitiesFilter{Page: 1, Limit: 10}

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserIdentities(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindByUUID db error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, errors.New("db error") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserIdentities(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: 1, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserIdentities(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("FindUserIdentitiesPaginated error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(1, tenantID), nil }
		ui.findUserIdentitiesPaginatedFn = func(_ GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error) {
			return nil, errors.New("ident err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, _, err := svc.GetUserIdentities(context.Background(), uid, tenantID, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ident err")
	})

	// An identity is reported with the PROVIDER that issued it — never with a
	// client. Applications are not owners of an identity (migration 030).
	t.Run("success with identity provider loaded", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		idpUUID := uuid.New()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(1, tenantID), nil }
		ui.findUserIdentitiesPaginatedFn = func(_ GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error) {
			return &PaginationResult[UserIdentity]{
				Data:  []UserIdentity{{UserIdentityUUID: uuid.New(), IdentityProviderID: 5, Provider: "default"}},
				Total: 1,
			}, nil
		}
		idp.findByIDFn = func(_ any, _ ...string) (*IdentityProvider, error) {
			return &IdentityProvider{IdentityProviderID: 5, TenantID: tenantID, IdentityProviderUUID: idpUUID, Name: "Built-in"}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, _, err := svc.GetUserIdentities(context.Background(), uid, tenantID, filter)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		require.NotNil(t, res[0].IdentityProviderUUID)
		assert.Equal(t, idpUUID, *res[0].IdentityProviderUUID)
		assert.Equal(t, "Built-in", res[0].IdentityProviderName)
	})

	// Cross-tenant leak guard: a provider row belonging to another tenant is
	// dropped rather than surfaced, even though the identity references it.
	t.Run("provider from another tenant is not surfaced", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(1, tenantID), nil }
		ui.findUserIdentitiesPaginatedFn = func(_ GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error) {
			return &PaginationResult[UserIdentity]{
				Data:  []UserIdentity{{UserIdentityUUID: uuid.New(), IdentityProviderID: 5}},
				Total: 1,
			}, nil
		}
		idp.findByIDFn = func(_ any, _ ...string) (*IdentityProvider, error) {
			return &IdentityProvider{IdentityProviderID: 5, TenantID: tenantID + 1, Name: "Other tenant"}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, _, err := svc.GetUserIdentities(context.Background(), uid, tenantID, filter)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Nil(t, res[0].IdentityProviderUUID)
		assert.Empty(t, res[0].IdentityProviderName)
	})

	t.Run("FindByID error → provider nil", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return userWithAccess(1, tenantID), nil }
		ui.findUserIdentitiesPaginatedFn = func(_ GetUserIdentitiesFilter) (*PaginationResult[UserIdentity], error) {
			return &PaginationResult[UserIdentity]{
				Data:  []UserIdentity{{UserIdentityUUID: uuid.New(), IdentityProviderID: 5}},
				Total: 1,
			}, nil
		}
		idp.findByIDFn = func(_ any, _ ...string) (*IdentityProvider, error) { return nil, errors.New("find err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		res, _, err := svc.GetUserIdentities(context.Background(), uid, tenantID, filter)
		require.NoError(t, err)
		assert.Nil(t, res[0].IdentityProviderUUID)
	})
}

// ---------------------------------------------------------------------------
// toUserServiceDataResult
// ---------------------------------------------------------------------------

func TestToUserServiceDataResult(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, toUserServiceDataResult(nil))
	})

	t.Run("empty user", func(t *testing.T) {
		res := toUserServiceDataResult(&User{UserUUID: uuid.New()})
		assert.NotNil(t, res)
		assert.Nil(t, res.Tenant)
		assert.Nil(t, res.UserIdentities)
		assert.Nil(t, res.Roles)
	})

	t.Run("with tenant from first identity", func(t *testing.T) {
		tUUID := uuid.New()
		res := toUserServiceDataResult(&User{
			UserUUID: uuid.New(),
			UserIdentities: []UserIdentity{{
				TenantID: 1,
				Tenant:   &Tenant{TenantUUID: tUUID, Name: "main"},
			}},
		})
		assert.NotNil(t, res.Tenant)
		assert.Equal(t, tUUID, res.Tenant.TenantUUID)
	})

	t.Run("with identities and identity provider", func(t *testing.T) {
		idpUUID := uuid.New()
		res := toUserServiceDataResult(&User{
			UserUUID: uuid.New(),
			UserIdentities: []UserIdentity{{
				UserIdentityUUID: uuid.New(),
				Provider:         "google",
				IdentityProvider: &IdentityProvider{IdentityProviderUUID: idpUUID, Name: "Google Workspace"},
			}},
		})
		require.NotNil(t, res.UserIdentities)
		assert.Len(t, *res.UserIdentities, 1)
		require.NotNil(t, (*res.UserIdentities)[0].IdentityProviderUUID)
		assert.Equal(t, idpUUID, *(*res.UserIdentities)[0].IdentityProviderUUID)
		assert.Equal(t, "Google Workspace", (*res.UserIdentities)[0].IdentityProviderName)
	})

	t.Run("with identities and no preloaded provider", func(t *testing.T) {
		res := toUserServiceDataResult(&User{
			UserUUID: uuid.New(),
			UserIdentities: []UserIdentity{{
				UserIdentityUUID: uuid.New(),
				Provider:         "default",
			}},
		})
		require.NotNil(t, res.UserIdentities)
		assert.Nil(t, (*res.UserIdentities)[0].IdentityProviderUUID)
	})

	t.Run("with roles", func(t *testing.T) {
		res := toUserServiceDataResult(&User{
			UserUUID: uuid.New(),
			Roles:    []Role{{RoleUUID: uuid.New(), Name: "admin"}},
		})
		require.NotNil(t, res.Roles)
		assert.Len(t, *res.Roles, 1)
	})

	t.Run("identity with nil tenant", func(t *testing.T) {
		res := toUserServiceDataResult(&User{
			UserUUID: uuid.New(),
			UserIdentities: []UserIdentity{{
				UserIdentityUUID: uuid.New(),
				Tenant:           nil,
			}},
		})
		assert.Nil(t, res.Tenant)
	})
}

func TestUserService_ForcePasswordChange(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(any, ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		err := svc.ForcePasswordChange(context.Background(), uuid.New(), 1, true)
		require.Error(t, err)
	})

	t.Run("wrong tenant returns forbidden", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		userUUID := uuid.New()
		ur.findByUUIDFn = func(any, ...string) (*User, error) {
			return &User{UserUUID: userUUID, UserID: 100}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		err := svc.ForcePasswordChange(context.Background(), userUUID, 1, true)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		userUUID := uuid.New()
		ur.findByUUIDFn = func(any, ...string) (*User, error) {
			return &User{
				UserUUID: userUUID,
				UserID:   100,
				UserIdentities: []UserIdentity{
					{TenantID: 1},
				},
			}, nil
		}
		ur.setForcePasswordChangeFn = func(uuid.UUID, bool) error { return nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		err := svc.ForcePasswordChange(context.Background(), userUUID, 1, true)
		require.NoError(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		userUUID := uuid.New()
		ur.findByUUIDFn = func(any, ...string) (*User, error) {
			return &User{
				UserUUID: userUUID,
				UserID:   100,
				UserIdentities: []UserIdentity{
					{TenantID: 1},
				},
			}, nil
		}
		ur.setForcePasswordChangeFn = func(uuid.UUID, bool) error { return errors.New("db error") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		err := svc.ForcePasswordChange(context.Background(), userUUID, 1, true)
		require.Error(t, err)
	})
}

func TestUserService_FindBySubAndClientID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findBySubAndClientIDFn = func(string, string) (*User, error) {
			return &User{UserUUID: uuid.New(), Status: shared.StatusActive}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		user, err := svc.FindBySubAndClientID(context.Background(), "sub1", "client1")
		require.NoError(t, err)
		assert.NotNil(t, user)
	})

	// This used to assert that ANY resolved row was returned regardless of
	// users.status — the fixture above did not even set a status. That was the
	// bug: nothing on the request path read the column back, so deactivating or
	// suspending an account left its access token working until expiry and its
	// refresh token minting replacements forever. The middleware turns nil into
	// 401, so a disabled user now stops being served on the next request.
	for _, status := range []string{shared.StatusInactive, shared.StatusSuspended, shared.StatusPending, ""} {
		t.Run("non-active status resolves to nil: "+status, func(t *testing.T) {
			ur, ui, urr, rr, tr, idp, cr := defaultMocks()
			ur.findBySubAndClientIDFn = func(string, string) (*User, error) {
				return &User{UserUUID: uuid.New(), Status: status}, nil
			}
			_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
			user, err := svc.FindBySubAndClientID(context.Background(), "sub1", "client1")
			require.NoError(t, err)
			assert.Nil(t, user)
		})
	}

	t.Run("FindByUserID applies the same status gate", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByIDFn = func(any, ...string) (*User, error) {
			return &User{UserUUID: uuid.New(), Status: shared.StatusSuspended}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		user, err := svc.FindByUserID(context.Background(), 7)
		require.NoError(t, err)
		assert.Nil(t, user, "the federated/multi-issuer path must not admit a suspended user either")
	})

	t.Run("repo error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findBySubAndClientIDFn = func(string, string) (*User, error) {
			return nil, errors.New("db error")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		user, err := svc.FindBySubAndClientID(context.Background(), "sub1", "client1")
		require.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestValidateTenantAccess(t *testing.T) {
	tenantID := int64(1)

	t.Run("nil actor returns unauthorized", func(t *testing.T) {
		err := ValidateTenantAccess(nil, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("nil target returns not found", func(t *testing.T) {
		err := ValidateTenantAccess(&User{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("actor with no identities returns forbidden", func(t *testing.T) {
		err := ValidateTenantAccess(&User{}, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("matching tenant ID permits access", func(t *testing.T) {
		actor := &User{
			UserIdentities: []UserIdentity{
				{TenantID: tenantID},
			},
		}
		err := ValidateTenantAccess(actor, &Tenant{TenantID: tenantID})
		require.NoError(t, err)
	})

	t.Run("system tenant denied cross-tenant", func(t *testing.T) {
		// Lockdown: a system-tenant identity no longer grants cross-tenant
		// access. The system override lives only in the tenant package.
		actor := &User{
			UserIdentities: []UserIdentity{
				{TenantID: 99, Tenant: &Tenant{TenantID: 99, IsSystem: true}},
			},
		}
		err := ValidateTenantAccess(actor, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant access denied")
	})

	t.Run("wrong tenant ID returns forbidden", func(t *testing.T) {
		actor := &User{
			UserIdentities: []UserIdentity{
				{TenantID: 99},
			},
		}
		err := ValidateTenantAccess(actor, &Tenant{TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant access denied")
	})
}

func TestToTenantServiceDataResult(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, toTenantServiceDataResult(nil))
	})

	t.Run("non-nil returns populated result", func(t *testing.T) {
		tnt := &Tenant{Name: "acme", TenantID: 1}
		res := toTenantServiceDataResult(tnt)
		require.NotNil(t, res)
		assert.Equal(t, "acme", res.Name)
	})
}

func TestToRoleServiceDataResult(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, toRoleServiceDataResult(nil))
	})

	t.Run("non-nil returns populated result", func(t *testing.T) {
		role := &Role{Name: "admin", RoleID: 1}
		res := toRoleServiceDataResult(role)
		require.NotNil(t, res)
		assert.Equal(t, "admin", res.Name)
	})
}

func TestToClientServiceDataResult(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.Nil(t, ToClientServiceDataResult(nil))
	})

	t.Run("non-nil returns populated result", func(t *testing.T) {
		client := &Client{Name: "app", ClientID: 1}
		res := ToClientServiceDataResult(client)
		require.NotNil(t, res)
		assert.Equal(t, "app", res.Name)
	})
}

func TestUserService_GetUserMFA(t *testing.T) {
	uid := uuid.New()
	userID := int64(42)

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.GetUserMFA(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindByUUID db error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) { return nil, errors.New("db error") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.GetUserMFA(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("tenant access denied", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, UserIdentities: []UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr)
		_, err := svc.GetUserMFA(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("backup codes count error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_mfa_backup_codes" WHERE user_id = \$1 AND used = false`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))
		_, err := svc.GetUserMFA(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count backup codes")
	})

	t.Run("webauthn query error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{UserID: userID, UserIdentities: []UserIdentity{{TenantID: 1}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_mfa_backup_codes" WHERE user_id = \$1 AND used = false`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT credential_uuid, name, transport, last_used_at, created_at FROM "user_mfa_webauthn_credentials" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))
		_, err := svc.GetUserMFA(context.Background(), uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to query webauthn credentials")
	})

	t.Run("success with MFA configured", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		mfaAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{
				UserID:             userID,
				IsTOTPEnabled:      true,
				IsWebAuthnEnabled:  true,
				IsPhoneVerified:    true,
				FirstMFAEnrolledAt: &mfaAt,
				UserIdentities:     []UserIdentity{{TenantID: 1}},
			}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_mfa_backup_codes" WHERE user_id = \$1 AND used = false`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
		mock.ExpectQuery(`SELECT credential_uuid, name, transport, last_used_at, created_at FROM "user_mfa_webauthn_credentials" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"credential_uuid", "name", "transport", "last_used_at", "created_at"}).
				AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Phone Key", "internal", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)))
		mock.ExpectQuery(`SELECT is_verified FROM "user_mfa_phones" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"is_verified"}).AddRow(true))
		res, err := svc.GetUserMFA(context.Background(), uid, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.True(t, res.IsTOTPEnabled)
		assert.True(t, res.IsWebAuthnEnabled)
		assert.True(t, res.IsSMSEnabled)
		assert.Equal(t, 5, res.BackupCodesCount)
		assert.NotNil(t, res.FirstMFAEnrolledAt)
		assert.Len(t, res.WebAuthnKeys, 1)
		assert.Equal(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", res.WebAuthnKeys[0].CredentialUUID)
		assert.Equal(t, "Phone Key", res.WebAuthnKeys[0].Name)
	})

	t.Run("success with no MFA configured", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*User, error) {
			return &User{
				UserID:         userID,
				UserIdentities: []UserIdentity{{TenantID: 1}},
			}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "user_mfa_backup_codes" WHERE user_id = \$1 AND used = false`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT credential_uuid, name, transport, last_used_at, created_at FROM "user_mfa_webauthn_credentials" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"credential_uuid", "name", "transport", "last_used_at", "created_at"}))
		mock.ExpectQuery(`SELECT is_verified FROM "user_mfa_phones" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"is_verified"}).AddRow(false))
		res, err := svc.GetUserMFA(context.Background(), uid, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.False(t, res.IsTOTPEnabled)
		assert.False(t, res.IsWebAuthnEnabled)
		assert.False(t, res.IsSMSEnabled)
		assert.Equal(t, 0, res.BackupCodesCount)
		assert.Nil(t, res.FirstMFAEnrolledAt)
		assert.Empty(t, res.WebAuthnKeys)
	})
}
