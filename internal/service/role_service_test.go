package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
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
	return NewRoleService(nil, roleRepo, permRepo, rpRepo, userRepo, tenantRepo)
}

// ---------------------------------------------------------------------------
// RoleService.GetByUUID
// ---------------------------------------------------------------------------

func TestRoleService_GetByUUID(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()

	cases := []struct {
		name        string
		setupRepo   func(r *mockRoleRepo)
		expectError bool
		errContains string
	}{
		{
			name: "found, correct tenant → success",
			setupRepo: func(r *mockRoleRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) {
					return newRole(1, "admin", tenantID), nil
				}
			},
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockRoleRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return nil, nil }
			},
			expectError: true,
			errContains: "role not found",
		},
		{
			name: "wrong tenant → access denied",
			setupRepo: func(r *mockRoleRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) {
					return newRole(1, "admin", 99), nil // different tenant
				}
			},
			expectError: true,
			errContains: "access denied",
		},
		{
			name: "repo error → propagated",
			setupRepo: func(r *mockRoleRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) {
					return nil, errors.New("db error")
				}
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roleRepo := &mockRoleRepo{}
			tc.setupRepo(roleRepo)
			svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
			result, err := svc.GetByUUID(roleUUID, tenantID)
			if tc.expectError {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RoleService.Get (paginated)
// ---------------------------------------------------------------------------

func TestRoleService_Get(t *testing.T) {
	t.Run("success – empty result", func(t *testing.T) {
		svc := newRoleService(&mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		result, err := svc.Get(RoleServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})

}

// ---------------------------------------------------------------------------
// RoleService.DeleteByUUID
// ---------------------------------------------------------------------------

func TestRoleService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	actorUUID := uuid.New()

	// helper: actor user with default-tenant identity → can access any tenant
	actorUser := func() *model.User {
		return &model.User{
			UserID: 1,
			UserIdentities: []model.UserIdentity{
				{TenantID: tenantID, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}},
			},
		}
	}

	cases := []struct {
		name        string
		roleRepo    *mockRoleRepo
		userRepo    *mockUserRepo
		expectError bool
		errContains string
	}{
		{
			name: "role not found → error",
			roleRepo: &mockRoleRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
			},
			userRepo:    &mockUserRepo{},
			expectError: true,
			errContains: "role not found",
		},
		{
			name: "wrong tenant → access denied",
			roleRepo: &mockRoleRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
					return newRole(1, "admin", 99), nil
				},
			},
			userRepo:    &mockUserRepo{},
			expectError: true,
			errContains: "access denied",
		},
		{
			name: "actor user not found → error",
			roleRepo: &mockRoleRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
					return newRole(1, "admin", tenantID), nil
				},
			},
			userRepo: &mockUserRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
			},
			expectError: true,
			errContains: "actor user not found",
		},
		{
			name: "system role → cannot delete",
			roleRepo: &mockRoleRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
					r := newRole(1, "system", tenantID)
					r.IsSystem = true
					return r, nil
				},
			},
			userRepo: &mockUserRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(), nil },
			},
			expectError: true,
			errContains: "system role",
		},
		{
			name: "success",
			roleRepo: &mockRoleRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
					return newRole(1, "admin", tenantID), nil
				},
			},
			userRepo: &mockUserRepo{
				findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(), nil },
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newRoleService(tc.roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, tc.userRepo, &mockTenantRepo{})
			result, err := svc.DeleteByUUID(roleUUID, tenantID, actorUUID)
			if tc.expectError {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RoleService.GetRolePermissions
// ---------------------------------------------------------------------------

func TestRoleService_GetRolePermissions(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()

	t.Run("role not found → error", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetRolePermissions(RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("wrong tenant → error", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", 99), nil // different tenant
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetRolePermissions(RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("success – empty permissions", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}
		svc := newRoleService(roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		result, err := svc.GetRolePermissions(RoleServiceGetPermissionsFilter{RoleUUID: roleUUID, TenantID: tenantID})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})
}

// ---------------------------------------------------------------------------
// RoleService.SetStatusByUUID – transactional
// ---------------------------------------------------------------------------

func TestRoleService_SetStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	actorUUID := uuid.New()

	actorUser := func() *model.User {
		return &model.User{
			UserID: 1,
			UserIdentities: []model.UserIdentity{
				{TenantID: tenantID, Tenant: &model.Tenant{TenantID: tenantID, IsDefault: true}},
			},
		}
	}

	t.Run("role not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}
		svc := NewRoleService(db, roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.SetStatusByUUID(roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("system role → cannot update", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				r := newRole(1, "system-role", tenantID)
				r.IsSystem = true
				return r, nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(), nil },
		}
		svc := NewRoleService(db, roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, userRepo, &mockTenantRepo{})
		_, err := svc.SetStatusByUUID(roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system role")
	})

	t.Run("success → status updated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return newRole(1, "admin", tenantID), nil
			},
		}
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return actorUser(), nil },
		}
		svc := NewRoleService(db, roleRepo, &mockPermissionRepo{}, &mockRolePermissionRepo{}, userRepo, &mockTenantRepo{})
		result, err := svc.SetStatusByUUID(roleUUID, tenantID, model.StatusInactive, actorUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.StatusInactive, result.Status)
	})
}
