package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPermission returns a minimal Permission fixture.
func newPermission(id int64, name string, tenantID int64) *model.Permission {
	return &model.Permission{
		PermissionID:   id,
		PermissionUUID: uuid.New(),
		Name:           name,
		TenantID:       tenantID,
		Status:         model.StatusActive,
	}
}

func newPermissionService(permRepo *mockPermissionRepo, apiRepo *mockAPIRepo, roleRepo *mockRoleRepo, clientRepo *mockClientRepo) PermissionService {
	return NewPermissionService(nil, permRepo, apiRepo, roleRepo, clientRepo)
}

// ---------------------------------------------------------------------------
// PermissionService.GetByUUID
// ---------------------------------------------------------------------------

func TestPermissionService_GetByUUID(t *testing.T) {
	tenantID := int64(1)
	permUUID := uuid.New()

	cases := []struct {
		name        string
		setupRepo   func(r *mockPermissionRepo)
		expectError bool
		errContains string
	}{
		{
			name: "found → success",
			setupRepo: func(r *mockPermissionRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Permission, error) {
					return newPermission(1, "read:users", tenantID), nil
				}
			},
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockPermissionRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Permission, error) {
					return nil, nil
				}
			},
			expectError: true,
			errContains: "permission not found",
		},
		{
			name: "repo error → propagated",
			setupRepo: func(r *mockPermissionRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Permission, error) {
					return nil, errors.New("db error")
				}
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			permRepo := &mockPermissionRepo{}
			tc.setupRepo(permRepo)
			svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
			result, err := svc.GetByUUID(permUUID, tenantID)
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
// PermissionService.Get – filters
// ---------------------------------------------------------------------------

func TestPermissionService_Get(t *testing.T) {
	t.Run("client uuid filter → error", func(t *testing.T) {
		clientUUID := "some-client-uuid"
		svc := newPermissionService(&mockPermissionRepo{}, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Get(PermissionServiceGetFilter{ClientUUID: &clientUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer supported")
	})

	t.Run("api uuid not found → error", func(t *testing.T) {
		apiUUID := uuid.New().String()
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return nil, nil },
		}
		svc := newPermissionService(&mockPermissionRepo{}, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Get(PermissionServiceGetFilter{APIUUID: &apiUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api not found")
	})

	t.Run("role uuid not found → error", func(t *testing.T) {
		roleUUID := uuid.New().String()
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}
		svc := newPermissionService(&mockPermissionRepo{}, &mockAPIRepo{}, roleRepo, &mockClientRepo{})
		_, err := svc.Get(PermissionServiceGetFilter{RoleUUID: &roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("success – empty result", func(t *testing.T) {
		svc := newPermissionService(&mockPermissionRepo{}, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.Get(PermissionServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})
}

// ---------------------------------------------------------------------------
// PermissionService.DeleteByUUID
// ---------------------------------------------------------------------------

func TestPermissionService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	permUUID := uuid.New()

	cases := []struct {
		name        string
		setupRepo   func(r *mockPermissionRepo)
		expectError bool
		errContains string
	}{
		{
			name: "not found → error",
			setupRepo: func(r *mockPermissionRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Permission, error) {
					return nil, nil
				}
			},
			expectError: true,
			errContains: "permission not found",
		},
		{
			name: "default permission → cannot delete",
			setupRepo: func(r *mockPermissionRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Permission, error) {
					p := newPermission(1, "read:users", tenantID)
					p.IsDefault = true
					return p, nil
				}
			},
			expectError: true,
			errContains: "default permission",
		},
		{
			name: "success",
			setupRepo: func(r *mockPermissionRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Permission, error) {
					return newPermission(1, "read:users", tenantID), nil
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			permRepo := &mockPermissionRepo{}
			tc.setupRepo(permRepo)
			svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
			result, err := svc.DeleteByUUID(permUUID, tenantID)
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
// PermissionService.SetStatus / SetActiveStatusByUUID – transactional
// ---------------------------------------------------------------------------

func TestPermissionService_SetStatus(t *testing.T) {
	tenantID := int64(1)
	permUUID := uuid.New()

	t.Run("not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, nil
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		svcWithDB := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_ = svc
		result, err := svcWithDB.SetStatus(permUUID, tenantID, model.StatusInactive)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("success → status updated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		perm := newPermission(1, "read:users", tenantID)
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.SetStatus(permUUID, tenantID, model.StatusInactive)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.StatusInactive, result.Status)
	})
}

func TestPermissionService_SetActiveStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	permUUID := uuid.New()

	t.Run("default permission → cannot toggle", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		perm := newPermission(1, "read:users", tenantID)
		perm.IsDefault = true
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.SetActiveStatusByUUID(permUUID, tenantID)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("active → inactive", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		perm := newPermission(1, "read:users", tenantID)
		perm.Status = model.StatusActive
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.SetActiveStatusByUUID(permUUID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, model.StatusInactive, result.Status)
	})
}
