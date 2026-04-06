package service

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
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
	tuRepo *mockTenantUserRepo,
) (*gorm.DB, UserService) {
	t.Helper()
	db, _ := newMockGormDB(t)
	svc := NewUserService(db, userRepo, uiRepo, urRepo, roleRepo, tenantRepo, idpRepo, clientRepo, tuRepo)
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
	tuRepo *mockTenantUserRepo,
) (*gorm.DB, sqlmock.Sqlmock, UserService) {
	t.Helper()
	db, mock := newMockGormDB(t)
	svc := NewUserService(db, userRepo, uiRepo, urRepo, roleRepo, tenantRepo, idpRepo, clientRepo, tuRepo)
	return db, mock, svc
}

func defaultMocks() (*mockUserRepo, *mockUserIdentityRepo, *mockUserRoleRepo, *mockRoleRepo, *mockTenantRepo, *mockIdentityProviderRepo, *mockClientRepo, *mockTenantUserRepo) {
	return &mockUserRepo{}, &mockUserIdentityRepo{}, &mockUserRoleRepo{}, &mockRoleRepo{},
		&mockTenantRepo{}, &mockIdentityProviderRepo{}, &mockClientRepo{}, &mockTenantUserRepo{}
}

// User with tenant access (tenantID=1) and default-tenant identity for ValidateTenantAccess
func userWithAccess(userID int64, tenantID int64) *model.User {
	return &model.User{
		UserID:   userID,
		UserUUID: uuid.New(),
		UserIdentities: []model.UserIdentity{{
			TenantID: tenantID,
			Tenant:   &model.Tenant{TenantID: tenantID, IsDefault: true},
		}},
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestUserService_Get(t *testing.T) {
	t.Run("invalid role UUID", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		bad := "not-a-uuid"
		_, err := svc.Get(UserServiceGetFilter{RoleUUID: &bad, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role UUID")
	})

	t.Run("role not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		rid := uuid.New().String()
		_, err := svc.Get(UserServiceGetFilter{RoleUUID: &rid, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindPaginated error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findPaginatedFn = func(_ repository.UserRepositoryGetFilter) (*repository.PaginationResult[model.User], error) {
			return nil, errors.New("db error")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.Get(UserServiceGetFilter{TenantID: 1})
		require.Error(t, err)
	})

	t.Run("success with role filter", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) {
			return &model.Role{RoleID: 5}, nil
		}
		ur.findPaginatedFn = func(f repository.UserRepositoryGetFilter) (*repository.PaginationResult[model.User], error) {
			assert.NotNil(t, f.RoleID)
			return &repository.PaginationResult[model.User]{Data: []model.User{{UserUUID: uuid.New()}}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		rid := uuid.New().String()
		res, err := svc.Get(UserServiceGetFilter{RoleUUID: &rid, TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("success without role filter", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.Get(UserServiceGetFilter{TenantID: 1})
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.GetByUUID(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.GetByUUID(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.GetByUUID(uid, 1)
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, "bad-uuid", creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tenant UUID")
	})

	t.Run("tenant not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("creator user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 { // creator lookup
				return nil, nil
			}
			return nil, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "creator user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 2, UserIdentities: []model.UserIdentity{}}, nil // no identities → error
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("FindByUsername error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		ur.findByUsernameFn = func(_ string) (*model.User, error) { return nil, errors.New("username err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username err")
	})

	t.Run("username already exists", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		ur.findByUsernameFn = func(_ string) (*model.User, error) { return &model.User{}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username already exists")
	})

	t.Run("FindByEmail error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		ur.findByEmailFn = func(_ string) (*model.User, error) { return nil, errors.New("email err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", &email, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email err")
	})

	t.Run("email already exists", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		ur.findByEmailFn = func(_ string) (*model.User, error) { return &model.User{}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", &email, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email already exists")
	})

	t.Run("Create user error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		ur.createFn = func(_ *model.User) (*model.User, error) { return nil, errors.New("create user err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create user err")
	})

	t.Run("FindDefaultByTenantID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return nil, errors.New("no client") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default auth client not found")
	})

	t.Run("Create identity error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		ui.createFn = func(_ *model.UserIdentity) (*model.UserIdentity, error) { return nil, errors.New("ident err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ident err")
	})

	t.Run("Create tenant user error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		tu.createFn = func(_ *model.TenantUser) (*model.TenantUser, error) { return nil, errors.New("tu err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tu err")
	})

	t.Run("findDefaultRole error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return nil, errors.New("role paginate err")
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role paginate err")
	})

	t.Run("findDefaultRole fallback — no default or registered", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return &repository.PaginationResult[model.Role]{Data: []model.Role{}}, nil
		}
		// FindByNameAndTenantID returns nil → no default role found
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no default role found")
	})

	t.Run("findDefaultRole fallback — FindByNameAndTenantID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return &repository.PaginationResult[model.Role]{Data: []model.Role{}}, nil
		}
		rr.findByNameAndTenantIDFn = func(_ string, _ int64) (*model.Role, error) { return nil, errors.New("name err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("Create user role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return &repository.PaginationResult[model.Role]{Data: []model.Role{{RoleID: 1}}}, nil
		}
		urr.createFn = func(_ *model.UserRole) (*model.UserRole, error) { return nil, errors.New("ur create err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ur create err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount <= 1 {
				return userWithAccess(2, 1), nil // creator
			}
			return nil, errors.New("fetch err") // final fetch
		}
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return &repository.PaginationResult[model.Role]{Data: []model.Role{{RoleID: 1}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success with email and phone", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		phone := "555"
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount <= 1 {
				return userWithAccess(2, 1), nil // creator
			}
			return &model.User{UserUUID: uuid.New(), Username: "user"}, nil // fetch result
		}
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return &repository.PaginationResult[model.Role]{Data: []model.Role{{RoleID: 1}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.Create("user", "name", &email, &phone, "pass", "active", datatypes.JSON([]byte("{}")), tenantUUID, creatorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("findDefaultRole — fallback to registered role success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount <= 1 {
				return userWithAccess(2, 1), nil
			}
			return &model.User{UserUUID: uuid.New()}, nil
		}
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		rr.findPaginatedFn = func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
			return &repository.PaginationResult[model.Role]{Data: []model.Role{}}, nil // no default
		}
		rr.findByNameAndTenantIDFn = func(_ string, _ int64) (*model.Role, error) {
			return &model.Role{RoleID: 5}, nil // fallback registered
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("HashPassword error", func(t *testing.T) {
		origHash := util.HashPassword
		defer func() { util.HashPassword = origHash }()
		util.HashPassword = func(_ []byte) ([]byte, error) { return nil, errors.New("hash error") }

		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		tr.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return &model.Tenant{TenantID: 1}, nil }
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return userWithAccess(2, 1), nil }
		cr.findDefaultByTenantIDFn = func(_ int64) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create("user", "name", nil, nil, "pass", "active", nil, tenantUUID, creatorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hash error")
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "u", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "u", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("updater user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "old", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return nil, nil // updater not found
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "old", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "updater user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "old", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return &model.User{UserID: 2, UserIdentities: []model.UserIdentity{}}, nil // updater with no identities
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "old", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("username change → FindByUsername error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "old", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.findByUsernameFn = func(_ string) (*model.User, error) { return nil, errors.New("uname err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "new-name", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "uname err")
	})

	t.Run("username conflict", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "old", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.findByUsernameFn = func(_ string) (*model.User, error) { return &model.User{UserID: 999}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "new-name", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username already exists")
	})

	t.Run("email change → FindByEmail error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "u", Email: "old@t.com", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		newEmail := "new@t.com"
		ur.findByEmailFn = func(_ string) (*model.User, error) { return nil, errors.New("email err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "u", "f", &newEmail, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email err")
	})

	t.Run("email conflict", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "u", Email: "old@t.com", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		newEmail := "new@t.com"
		ur.findByEmailFn = func(_ string) (*model.User, error) { return &model.User{UserID: 999}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "u", "f", &newEmail, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email already exists")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "u", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*model.User, error) { return nil, errors.New("update err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "u", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "u", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return nil, errors.New("fetch err")
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(uid, tenantID, "u", "f", nil, nil, "active", nil, updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success with all fields", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, Username: "u", Email: "old@t.com", UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return &model.User{UserUUID: uid, Username: "u"}, nil // fetch after update
		}
		newEmail := "new@t.com"
		phone := "555"
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.Update(uid, tenantID, "u", "f", &newEmail, &phone, "active", datatypes.JSON([]byte("{}")), updaterUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.SetStatus(uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.SetStatus(uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("updater not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return nil, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.SetStatus(uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "updater user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return &model.User{UserID: 2, UserIdentities: []model.UserIdentity{}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.SetStatus(uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("SetStatus error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.setStatusFn = func(_ uuid.UUID, _ string) error { return errors.New("status err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.SetStatus(uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return nil, errors.New("fetch err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.SetStatus(uid, tenantID, "inactive", updaterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			if callCount == 2 {
				return userWithAccess(2, 1), nil
			}
			return &model.User{UserUUID: uid, Status: "inactive"}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.SetStatus(uid, tenantID, "inactive", updaterUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// VerifyEmail
// ---------------------------------------------------------------------------

func TestUserService_VerifyEmail(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyEmail(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyEmail(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*model.User, error) { return nil, errors.New("upd err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyEmail(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upd err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyEmail(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return &model.User{UserUUID: uid, IsEmailVerified: true}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.VerifyEmail(uid, 1)
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyPhone(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyPhone(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*model.User, error) { return nil, errors.New("upd err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyPhone(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upd err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.VerifyPhone(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return &model.User{UserUUID: uid}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.VerifyPhone(uid, 1)
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.CompleteAccount(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.CompleteAccount(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		ur.updateByUUIDFn = func(_, _ any) (*model.User, error) { return nil, errors.New("upd err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.CompleteAccount(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upd err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.CompleteAccount(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return &model.User{UserUUID: uid}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.CompleteAccount(uid, 1)
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.DeleteByUUID(uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.DeleteByUUID(uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("deleter not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return nil, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.DeleteByUUID(uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deleter user not found")
	})

	t.Run("ValidateTenantAccess error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return &model.User{UserID: 2, UserIdentities: []model.UserIdentity{}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.DeleteByUUID(uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user has no identities")
	})

	t.Run("DeleteByUUID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		ur.deleteByUUIDFn = func(_ any) error { return errors.New("del err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.DeleteByUUID(uid, 1, deleterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1, Tenant: &model.Tenant{TenantID: 1}}}}, nil
			}
			return userWithAccess(2, 1), nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.DeleteByUUID(uid, 1, deleterUUID)
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("FindByUUID role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return nil, errors.New("role err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role err")
	})

	t.Run("role not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByUserIDAndRoleID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		urr.findByUserIDAndRoleIDFn = func(_, _ int64) (*model.UserRole, error) { return nil, errors.New("ur find err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ur find err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("role already assigned → skip", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return &model.User{UserUUID: uid}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		urr.findByUserIDAndRoleIDFn = func(_, _ int64) (*model.UserRole, error) { return &model.UserRole{}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Create user role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		urr.createFn = func(_ *model.UserRole) (*model.UserRole, error) { return nil, errors.New("ur create err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ur create err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return &model.User{UserUUID: uid}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.AssignUserRoles(uid, []uuid.UUID{roleUUID}, 1)
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
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 99}}}, nil
		}
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("FindByUUID role error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return nil, errors.New("role err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role err")
	})

	t.Run("role not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return nil, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("final fetch error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return nil, errors.New("fetch err")
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("DeleteByUserIDAndRoleID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		urr.deleteByUserIDAndRoleIDFn = func(_, _ int64) error { return errors.New("del ur err") }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.RemoveUserRole(uid, roleUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del ur err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		callCount := 0
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) {
			callCount++
			if callCount == 1 {
				return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{{TenantID: 1}}}, nil
			}
			return &model.User{UserUUID: uid}, nil
		}
		rr.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return &model.Role{RoleID: 5}, nil }
		_, mock, svc := fullUserSvcWithMock(t, ur, ui, urr, rr, tr, idp, cr, tu)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.RemoveUserRole(uid, roleUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// GetUserRoles
// ---------------------------------------------------------------------------

func TestUserService_GetUserRoles(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.GetUserRoles(uid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindRoles error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return &model.User{UserID: 1}, nil }
		ur.findRolesFn = func(_ int64) ([]model.Role, error) { return nil, errors.New("roles err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.GetUserRoles(uid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "roles err")
	})

	t.Run("success", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return &model.User{UserID: 1}, nil }
		ur.findRolesFn = func(_ int64) ([]model.Role, error) {
			return []model.Role{{RoleUUID: uuid.New(), Name: "editor"}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.GetUserRoles(uid)
		require.NoError(t, err)
		assert.Len(t, res, 1)
	})
}

// ---------------------------------------------------------------------------
// GetUserIdentities
// ---------------------------------------------------------------------------

func TestUserService_GetUserIdentities(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return nil, nil }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.GetUserIdentities(uid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("FindByUserID error", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return &model.User{UserID: 1}, nil }
		ui.findByUserIDFn = func(_ int64) ([]model.UserIdentity, error) { return nil, errors.New("ident err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		_, err := svc.GetUserIdentities(uid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ident err")
	})

	t.Run("success with client loaded", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return &model.User{UserID: 1}, nil }
		ui.findByUserIDFn = func(_ int64) ([]model.UserIdentity, error) {
			return []model.UserIdentity{{UserIdentityUUID: uuid.New(), ClientID: 5, Provider: "default"}}, nil
		}
		cr.findByIDFn = func(_ any, _ ...string) (*model.Client, error) {
			return &model.Client{ClientUUID: uuid.New(), Name: "main"}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.GetUserIdentities(uid)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.NotNil(t, res[0].Client)
	})

	t.Run("success with client ID zero", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return &model.User{UserID: 1}, nil }
		ui.findByUserIDFn = func(_ int64) ([]model.UserIdentity, error) {
			return []model.UserIdentity{{UserIdentityUUID: uuid.New(), ClientID: 0}}, nil
		}
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.GetUserIdentities(uid)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Nil(t, res[0].Client)
	})

	t.Run("FindByID error → client nil", func(t *testing.T) {
		ur, ui, urr, rr, tr, idp, cr, tu := defaultMocks()
		ur.findByUUIDFn = func(_ any, _ ...string) (*model.User, error) { return &model.User{UserID: 1}, nil }
		ui.findByUserIDFn = func(_ int64) ([]model.UserIdentity, error) {
			return []model.UserIdentity{{UserIdentityUUID: uuid.New(), ClientID: 5}}, nil
		}
		cr.findByIDFn = func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("find err") }
		_, svc := fullUserSvc(t, ur, ui, urr, rr, tr, idp, cr, tu)
		res, err := svc.GetUserIdentities(uid)
		require.NoError(t, err)
		assert.Nil(t, res[0].Client)
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
		res := toUserServiceDataResult(&model.User{UserUUID: uuid.New()})
		assert.NotNil(t, res)
		assert.Nil(t, res.Tenant)
		assert.Nil(t, res.UserIdentities)
		assert.Nil(t, res.Roles)
	})

	t.Run("with tenant from first identity", func(t *testing.T) {
		tUUID := uuid.New()
		res := toUserServiceDataResult(&model.User{
			UserUUID: uuid.New(),
			UserIdentities: []model.UserIdentity{{
				TenantID: 1,
				Tenant:   &model.Tenant{TenantUUID: tUUID, Name: "main"},
			}},
		})
		assert.NotNil(t, res.Tenant)
		assert.Equal(t, tUUID, res.Tenant.TenantUUID)
	})

	t.Run("with identities and client", func(t *testing.T) {
		cUUID := uuid.New()
		res := toUserServiceDataResult(&model.User{
			UserUUID: uuid.New(),
			UserIdentities: []model.UserIdentity{{
				UserIdentityUUID: uuid.New(),
				Provider:         "google",
				Client:           &model.Client{ClientUUID: cUUID},
			}},
		})
		require.NotNil(t, res.UserIdentities)
		assert.Len(t, *res.UserIdentities, 1)
		assert.NotNil(t, (*res.UserIdentities)[0].Client)
	})

	t.Run("with identities no client", func(t *testing.T) {
		res := toUserServiceDataResult(&model.User{
			UserUUID: uuid.New(),
			UserIdentities: []model.UserIdentity{{
				UserIdentityUUID: uuid.New(),
				Provider:         "default",
			}},
		})
		require.NotNil(t, res.UserIdentities)
		assert.Nil(t, (*res.UserIdentities)[0].Client)
	})

	t.Run("with roles", func(t *testing.T) {
		res := toUserServiceDataResult(&model.User{
			UserUUID: uuid.New(),
			Roles:    []model.Role{{RoleUUID: uuid.New(), Name: "admin"}},
		})
		require.NotNil(t, res.Roles)
		assert.Len(t, *res.Roles, 1)
	})

	t.Run("identity with nil tenant", func(t *testing.T) {
		res := toUserServiceDataResult(&model.User{
			UserUUID: uuid.New(),
			UserIdentities: []model.UserIdentity{{
				UserIdentityUUID: uuid.New(),
				Tenant:           nil,
			}},
		})
		assert.Nil(t, res.Tenant)
	})
}
