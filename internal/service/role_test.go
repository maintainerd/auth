package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/cache"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRole returns a minimal Role fixture.
func newRole(id int64, name string, tenantID int64) *model.Role {
	return &model.Role{
		RoleID:   id,
		RoleUUID: uuid.New(),
		Name:     name,
		TenantID: tenantID,
		Status:   model.StatusActive,
		Tenant:   &model.Tenant{TenantID: tenantID},
	}
}

func newRoleService(roleRepo *mockRoleRepo, permRepo *mockPermissionRepo, rpRepo *mockRolePermissionRepo, userRepo *mockUserRepo, tenantRepo *mockTenantRepo) RoleService {
	return NewRoleService(nil, roleRepo, permRepo, rpRepo, userRepo, tenantRepo, cache.NopInvalidator{})
}

// actor helper: user with default-tenant identity → can access any tenant.
func roleActorUser(tenantID int64) *model.User {
	return &model.User{
		UserID: 1,
		UserIdentities: []model.UserIdentity{
			{TenantID: tenantID, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}},
		},
	}
}

// actor helper: user that has NO identities → will fail ValidateTenantAccess.
func roleActorNoIdentities() *model.User {
	return &model.User{UserID: 1, UserIdentities: []model.UserIdentity{}}
}

// ---------------------------------------------------------------------------
// RoleService.Get (paginated)
// ---------------------------------------------------------------------------

func TestRoleService_Get(t *testing.T) {
	t.Run("success – empty result", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		result, err := svc.Get(context.Background(), RoleServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})

	t.Run("repo error → propagated", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.Get(context.Background(), RoleServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success – with roles", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findPaginatedFn: func(_ repository.RoleRepositoryGetFilter) (*repository.PaginationResult[model.Role], error) {
				return &repository.PaginationResult[model.Role]{
					Data:       []model.Role{{RoleUUID: uuid.New(), Name: "admin"}},
					Total:      1,
					Page:       1,
					Limit:      10,
					TotalPages: 1,
				}, nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		result, err := svc.Get(context.Background(), RoleServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, "admin", result.Data[0].Name)
	})
}

// ---------------------------------------------------------------------------
// RoleService.GetByUUID
// ---------------------------------------------------------------------------

func TestRoleService_GetByUUID(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()

	t.Run("found, correct tenant → success", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		result, err := svc.GetByUUID(context.Background(), roleUUID, tenantID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("not found → error", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByUUID(context.Background(), roleUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByUUID(context.Background(), roleUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("repo error → propagated", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByUUID(context.Background(), roleUUID, tenantID)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// RoleService.GetRolePermissions
// ---------------------------------------------------------------------------

func TestRoleService_GetRolePermissions(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()

	t.Run("role not found → error", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetRolePermissions(context.Background(), RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByUUID error → propagated", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetRolePermissions(context.Background(), RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.Error(t, err)
	})

	t.Run("wrong tenant → error", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetRolePermissions(context.Background(), RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("GetPermissionsByRoleUUID error → propagated", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
			getPermsByRoleUUIDFn: func(_ repository.RoleRepositoryGetPermissionsFilter) (*repository.PaginationResult[model.Permission], error) {
				return nil, errors.New("repo error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetRolePermissions(context.Background(), RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo error")
	})

	t.Run("success – with permissions", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
			getPermsByRoleUUIDFn: func(_ repository.RoleRepositoryGetPermissionsFilter) (*repository.PaginationResult[model.Permission], error) {
				return &repository.PaginationResult[model.Permission]{
					Data:       []model.Permission{{PermissionUUID: uuid.New(), Name: "read"}},
					Total:      1,
					Page:       1,
					Limit:      10,
					TotalPages: 1,
				}, nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		result, err := svc.GetRolePermissions(context.Background(), RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, "read", result.Data[0].Name)
	})

	t.Run("success – empty permissions", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		result, err := svc.GetRolePermissions(context.Background(), RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})
}

// ---------------------------------------------------------------------------
// RoleService.Create – transactional
// ---------------------------------------------------------------------------

func TestRoleService_Create(t *testing.T) {
	tenantID := int64(1)
	tenantUUID := uuid.New()
	actorUUID := uuid.New()

	t.Run("invalid tenant UUID → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, "not-a-uuid", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tenant UUID")
	})

	t.Run("tenant not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil },
		}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("tenant FindByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return nil, errors.New("db error")
			},
		}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("actor user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, TenantUUID: tenantUUID}, nil
			},
		}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorNoIdentities(), nil
			},
		}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, TenantUUID: tenantUUID}, nil
			},
		}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
	})

	t.Run("FindByNameAndTenantID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, TenantUUID: tenantUUID}, nil
			},
		}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.Error(t, err)
	})

	t.Run("role already exists → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
				return &model.Role{RoleID: 1, Name: "admin"}, nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, TenantUUID: tenantUUID}, nil
			},
		}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role already exist")
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			createOrUpdateFn: func(_ *model.Role) (*model.Role, error) {
				return nil, errors.New("create error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, TenantUUID: tenantUUID}, nil
			},
		}, cache.NopInvalidator{})
		_, err := svc.Create(context.Background(), "admin", "desc", false, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create error")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: tenantID, TenantUUID: tenantUUID}, nil
			},
		}, cache.NopInvalidator{})
		result, err := svc.Create(context.Background(), "admin", "desc", true, false, model.StatusActive, tenantUUID.String(), actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "admin", result.Name)
	})
}

// ---------------------------------------------------------------------------
// RoleService.Update – transactional
// ---------------------------------------------------------------------------

func TestRoleService_Update(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	actorUUID := uuid.New()

	t.Run("role not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "new", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "new", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "new", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "new", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorNoIdentities(), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "new", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
	})

	t.Run("system role → cannot update", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		r := newRole(1, "system", tenantID)
		r.IsSystem = true
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return r, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "new", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system role")
	})

	t.Run("duplicate name check error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "newname", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
	})

	t.Run("duplicate name found → role already exists", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		otherUUID := uuid.New()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
				return &model.Role{RoleUUID: otherUUID, Name: "newname"}, nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "newname", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role already exists")
	})

	t.Run("same name (no change) → no dup check, commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.Update(context.Background(), roleUUID, tenantID, "admin", "new desc", false, false, model.StatusActive, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "admin", result.Name)
	})

	t.Run("name changed, no dup → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.Update(context.Background(), roleUUID, tenantID, "editor", "new desc", false, false, model.StatusActive, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "editor", result.Name)
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
			createOrUpdateFn: func(_ *model.Role) (*model.Role, error) {
				return nil, errors.New("save error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.Update(context.Background(), roleUUID, tenantID, "admin", "desc", false, false, model.StatusActive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save error")
	})

	t.Run("dup name same UUID → allowed, commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		r := newRole(1, "admin", tenantID)
		r.RoleUUID = roleUUID // ensure role UUID matches the parameter
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return r, nil },
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Role, error) {
				return &model.Role{RoleUUID: roleUUID, Name: "newname"}, nil // same UUID as current
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.Update(context.Background(), roleUUID, tenantID, "newname", "desc", false, false, model.StatusActive, actorUUID)
		require.NoError(t, err)
		assert.Equal(t, "newname", result.Name)
	})
}

// ---------------------------------------------------------------------------
// RoleService.SetStatusByUUID – transactional
// ---------------------------------------------------------------------------

func TestRoleService_SetStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	actorUUID := uuid.New()

	t.Run("role not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorNoIdentities(), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
	})

	t.Run("system role → cannot update", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		r := newRole(1, "system", tenantID)
		r.IsSystem = true
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return r, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system role")
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
			createOrUpdateFn: func(_ *model.Role) (*model.Role, error) {
				return nil, errors.New("save error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save error")
	})

	t.Run("success → status updated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.SetStatusByUUID(context.Background(), roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.StatusInactive, result.Status)
	})
}

// ---------------------------------------------------------------------------
// RoleService.DeleteByUUID
// ---------------------------------------------------------------------------

func TestRoleService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	actorUUID := uuid.New()

	t.Run("role not found → error", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByUUID error → propagated", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.Error(t, err)
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found → error", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied → error", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorNoIdentities(), nil
			},
		}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
	})

	t.Run("system role → cannot delete", func(t *testing.T) {
		r := newRole(1, "system", tenantID)
		r.IsSystem = true
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return r, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system role")
	})

	t.Run("DeleteByUUID repo error → propagated", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{})
		_, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("success", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{})
		result, err := svc.DeleteByUUID(context.Background(), roleUUID, tenantID, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// ---------------------------------------------------------------------------
// RoleService.AddRolePermissions – transactional
// ---------------------------------------------------------------------------

func TestRoleService_AddRolePermissions(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	actorUUID := uuid.New()
	permUUID1 := uuid.New()
	permUUID2 := uuid.New()

	t.Run("role not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorNoIdentities(), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
	})

	t.Run("system role → cannot update", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		r := newRole(1, "system", tenantID)
		r.IsSystem = true
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return r, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system role")
	})

	t.Run("FindByUUIDs error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDsFn: func(_ []string, _ ...string) ([]model.Permission, error) {
				return nil, errors.New("perm error")
			},
		}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "perm error")
	})

	t.Run("permissions count mismatch → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDsFn: func(_ []string, _ ...string) ([]model.Permission, error) {
				return []model.Permission{{PermissionID: 1}}, nil // only 1 found but requested 2
			},
		}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1, permUUID2}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "one or more permissions not found")
	})

	t.Run("FindByRoleAndPermission error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDsFn: func(_ []string, _ ...string) ([]model.Permission, error) {
				return []model.Permission{{PermissionID: 10}}, nil
			},
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(_, _ int64) (*model.RolePermission, error) {
				return nil, errors.New("check error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "check error")
	})

	t.Run("Create RolePermission error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDsFn: func(_ []string, _ ...string) ([]model.Permission, error) {
				return []model.Permission{{PermissionID: 10}}, nil
			},
		}, &mockRolePermissionRepo{
			createFn: func(_ *model.RolePermission) (*model.RolePermission, error) {
				return nil, errors.New("create rp error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create rp error")
	})

	t.Run("existing association skipped → success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		callCount := 0
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, p ...string) (*model.Role, error) {
				callCount++
				if callCount == 1 {
					return newRole(1, "admin", tenantID), nil
				}
				// Second call: fetch with Permissions
				r := newRole(1, "admin", tenantID)
				r.Permissions = []model.Permission{{PermissionID: 10, Name: "read"}}
				return r, nil
			},
		}, &mockPermissionRepo{
			findByUUIDsFn: func(_ []string, _ ...string) ([]model.Permission, error) {
				return []model.Permission{{PermissionID: 10}}, nil
			},
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(_, _ int64) (*model.RolePermission, error) {
				return &model.RolePermission{RolePermissionID: 1}, nil // already exists
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("fetch with permissions error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		callCount := 0
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, p ...string) (*model.Role, error) {
				callCount++
				if callCount == 1 {
					return newRole(1, "admin", tenantID), nil
				}
				return nil, errors.New("fetch error")
			},
		}, &mockPermissionRepo{
			findByUUIDsFn: func(_ []string, _ ...string) ([]model.Permission, error) {
				return []model.Permission{{PermissionID: 10}}, nil
			},
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(_, _ int64) (*model.RolePermission, error) {
				return &model.RolePermission{RolePermissionID: 1}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch error")
	})

	t.Run("success → new association created", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		callCount := 0
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, p ...string) (*model.Role, error) {
				callCount++
				if callCount == 1 {
					return newRole(1, "admin", tenantID), nil
				}
				r := newRole(1, "admin", tenantID)
				r.Permissions = []model.Permission{{PermissionID: 10, Name: "read"}}
				return r, nil
			},
		}, &mockPermissionRepo{
			findByUUIDsFn: func(_ []string, _ ...string) ([]model.Permission, error) {
				return []model.Permission{{PermissionID: 10}}, nil
			},
		}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		require.NotNil(t, result.Permissions)
		assert.Len(t, *result.Permissions, 1)
	})

	t.Run("FindByUUID role error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permUUID1}, actorUUID)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// RoleService.RemoveRolePermissions – transactional
// ---------------------------------------------------------------------------

func TestRoleService_RemoveRolePermissions(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	actorUUID := uuid.New()
	permUUID := uuid.New()

	t.Run("role not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("wrong tenant → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("actor user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor user not found")
	})

	t.Run("tenant access denied → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorNoIdentities(), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no identities")
	})

	t.Run("system role → cannot update", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		r := newRole(1, "system", tenantID)
		r.IsSystem = true
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return r, nil },
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system role")
	})

	t.Run("FindByUUID permission error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return nil, errors.New("perm error")
			},
		}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "perm error")
	})

	t.Run("permission not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) { return nil, nil },
		}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission not found")
	})

	t.Run("FindByRoleAndPermission error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 10}, nil
			},
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(_, _ int64) (*model.RolePermission, error) {
				return nil, errors.New("rp error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rp error")
	})

	t.Run("association not found → idempotent success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		callCount := 0
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, p ...string) (*model.Role, error) {
				callCount++
				if callCount == 1 {
					return newRole(1, "admin", tenantID), nil
				}
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 10}, nil
			},
		}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("association not found → FindByUUID fetch error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		callCount := 0
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, p ...string) (*model.Role, error) {
				callCount++
				if callCount == 1 {
					return newRole(1, "admin", tenantID), nil
				}
				return nil, errors.New("fetch error")
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 10}, nil
			},
		}, &mockRolePermissionRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch error")
	})

	t.Run("RemoveByRoleAndPermission error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 10}, nil
			},
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(_, _ int64) (*model.RolePermission, error) {
				return &model.RolePermission{RolePermissionID: 1}, nil
			},
			removeByRoleAndPermissionFn: func(_, _ int64) error {
				return errors.New("remove error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "remove error")
	})

	t.Run("fetch after remove error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		callCount := 0
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, p ...string) (*model.Role, error) {
				callCount++
				if callCount == 1 {
					return newRole(1, "admin", tenantID), nil
				}
				return nil, errors.New("fetch error")
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 10}, nil
			},
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(_, _ int64) (*model.RolePermission, error) {
				return &model.RolePermission{RolePermissionID: 1}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch error")
	})

	t.Run("success → association removed", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		callCount := 0
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, p ...string) (*model.Role, error) {
				callCount++
				if callCount == 1 {
					return newRole(1, "admin", tenantID), nil
				}
				return newRole(1, "admin", tenantID), nil
			},
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Permission, error) {
				return &model.Permission{PermissionID: 10}, nil
			},
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(_, _ int64) (*model.RolePermission, error) {
				return &model.RolePermission{RolePermissionID: 1}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return roleActorUser(tenantID), nil
			},
		}, &mockTenantRepo{}, cache.NopInvalidator{})
		result, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("FindByUUID role error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return nil, errors.New("db error")
			},
		}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{}, cache.NopInvalidator{})
		_, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permUUID, actorUUID)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// toRoleServiceDataResult
// ---------------------------------------------------------------------------

func TestToRoleServiceDataResult(t *testing.T) {
	t.Run("nil → nil", func(t *testing.T) {
		assert.Nil(t, toRoleServiceDataResult(nil))
	})

	t.Run("role without permissions", func(t *testing.T) {
		r := &model.Role{RoleUUID: uuid.New(), Name: "admin", Status: model.StatusActive}
		result := toRoleServiceDataResult(r)
		require.NotNil(t, result)
		assert.Equal(t, "admin", result.Name)
		assert.Nil(t, result.Permissions)
	})

	t.Run("role with permissions", func(t *testing.T) {
		perms := []model.Permission{
			{PermissionUUID: uuid.New(), Name: "read"},
			{PermissionUUID: uuid.New(), Name: "write"},
		}
		r := &model.Role{RoleUUID: uuid.New(), Name: "admin", Permissions: perms}
		result := toRoleServiceDataResult(r)
		require.NotNil(t, result)
		require.NotNil(t, result.Permissions)
		assert.Len(t, *result.Permissions, 2)
	})
}
