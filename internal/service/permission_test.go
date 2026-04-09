package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
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
			result, err := svc.GetByUUID(context.Background(), permUUID, tenantID)
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
		_, err := svc.Get(context.Background(), PermissionServiceGetFilter{ClientUUID: &clientUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer supported")
	})

	t.Run("api uuid not found → error", func(t *testing.T) {
		apiUUID := uuid.New().String()
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return nil, nil },
		}
		svc := newPermissionService(&mockPermissionRepo{}, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Get(context.Background(), PermissionServiceGetFilter{APIUUID: &apiUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api not found")
	})

	t.Run("api uuid found → apiID set", func(t *testing.T) {
		apiUUID := uuid.New().String()
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) {
				return &model.API{APIID: 42}, nil
			},
		}
		svc := newPermissionService(&mockPermissionRepo{}, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.Get(context.Background(), PermissionServiceGetFilter{APIUUID: &apiUUID, TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("role uuid not found → error", func(t *testing.T) {
		roleUUID := uuid.New().String()
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}
		svc := newPermissionService(&mockPermissionRepo{}, &mockAPIRepo{}, roleRepo, &mockClientRepo{})
		_, err := svc.Get(context.Background(), PermissionServiceGetFilter{RoleUUID: &roleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("role uuid found → roleID set", func(t *testing.T) {
		roleUUID := uuid.New().String()
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return &model.Role{RoleID: 7}, nil
			},
		}
		svc := newPermissionService(&mockPermissionRepo{}, &mockAPIRepo{}, roleRepo, &mockClientRepo{})
		result, err := svc.Get(context.Background(), PermissionServiceGetFilter{RoleUUID: &roleUUID, TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("paginated repo error → propagated", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findPaginatedFn: func(_ repository.PermissionRepositoryGetFilter) (*repository.PaginationResult[model.Permission], error) {
				return nil, errors.New("db error")
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Get(context.Background(), PermissionServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success with data and API preloaded", func(t *testing.T) {
		perm := newPermission(1, "read:users", 1)
		perm.API = &model.API{APIUUID: uuid.New(), Name: "Users API"}
		permRepo := &mockPermissionRepo{
			findPaginatedFn: func(_ repository.PermissionRepositoryGetFilter) (*repository.PaginationResult[model.Permission], error) {
				return &repository.PaginationResult[model.Permission]{
					Data: []model.Permission{*perm}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.Get(context.Background(), PermissionServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		assert.NotNil(t, result.Data[0].API)
	})

	t.Run("success – empty result", func(t *testing.T) {
		svc := newPermissionService(&mockPermissionRepo{}, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.Get(context.Background(), PermissionServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})
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
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.SetStatus(context.Background(), permUUID, tenantID, model.StatusInactive)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "permission not found")
	})

	t.Run("find error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.SetStatus(context.Background(), permUUID, tenantID, model.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("createOrUpdate error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		perm := newPermission(1, "read:users", tenantID)
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
			createOrUpdateFn: func(_ *model.Permission) (*model.Permission, error) {
				return nil, errors.New("save err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.SetStatus(context.Background(), permUUID, tenantID, model.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
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
		result, err := svc.SetStatus(context.Background(), permUUID, tenantID, model.StatusInactive)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, model.StatusInactive, result.Status)
	})
}

func TestPermissionService_SetActiveStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	permUUID := uuid.New()

	t.Run("find error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.SetActiveStatusByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.SetActiveStatusByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission not found")
	})

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
		result, err := svc.SetActiveStatusByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("createOrUpdate error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		perm := newPermission(1, "read:users", tenantID)
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
			createOrUpdateFn: func(_ *model.Permission) (*model.Permission, error) {
				return nil, errors.New("save err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.SetActiveStatusByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
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
		result, err := svc.SetActiveStatusByUUID(context.Background(), permUUID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, model.StatusInactive, result.Status)
	})

	t.Run("inactive → active", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		perm := newPermission(1, "read:users", tenantID)
		perm.Status = model.StatusInactive
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.SetActiveStatusByUUID(context.Background(), permUUID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, model.StatusActive, result.Status)
	})
}

// ---------------------------------------------------------------------------
// PermissionService.DeleteByUUID – expanded
// ---------------------------------------------------------------------------

func TestPermissionService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	permUUID := uuid.New()

	t.Run("find error → propagated", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, errors.New("db err")
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.DeleteByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found → error", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, nil
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.DeleteByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission not found")
	})

	t.Run("default permission → cannot delete", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				p := newPermission(1, "read:users", tenantID)
				p.IsDefault = true
				return p, nil
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.DeleteByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default permission")
	})

	t.Run("delete repo error → propagated", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return newPermission(1, "read:users", tenantID), nil
			},
			deleteByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) error {
				return errors.New("delete fail")
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.DeleteByUUID(context.Background(), permUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete fail")
	})

	t.Run("success", func(t *testing.T) {
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return newPermission(1, "read:users", tenantID), nil
			},
		}
		svc := newPermissionService(permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.DeleteByUUID(context.Background(), permUUID, tenantID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// ---------------------------------------------------------------------------
// PermissionService.Create – transactional
// ---------------------------------------------------------------------------

func TestPermissionService_Create(t *testing.T) {
	tenantID := int64(1)
	apiUUID := uuid.New()

	t.Run("findByName error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByNameFn: func(_ string, _ int64) (*model.Permission, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, apiUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("permission already exists → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByNameFn: func(_ string, _ int64) (*model.Permission, error) {
				return newPermission(1, "read:users", tenantID), nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, apiUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("invalid api UUID → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, "not-a-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid api uuid")
	})

	t.Run("api FindByUUIDAndTenantID error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return nil, errors.New("api db err")
			},
		}
		svc := NewPermissionService(db, &mockPermissionRepo{}, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, apiUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api db err")
	})

	t.Run("api not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return nil, nil
			},
		}
		svc := NewPermissionService(db, &mockPermissionRepo{}, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, apiUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api not found")
	})

	t.Run("createOrUpdate error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return &model.API{APIID: 42}, nil
			},
		}
		permRepo := &mockPermissionRepo{
			createOrUpdateFn: func(_ *model.Permission) (*model.Permission, error) {
				return nil, errors.New("create err")
			},
		}
		svc := NewPermissionService(db, permRepo, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, apiUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("findByUUIDAndTenantID after create error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return &model.API{APIID: 42}, nil
			},
		}
		callCount := 0
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				callCount++
				// The second call (after create) fails
				return nil, errors.New("fetch err")
			},
		}
		svc := NewPermissionService(db, permRepo, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, apiUUID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		api := &model.API{APIID: 42, APIUUID: apiUUID, Name: "Users API"}
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return api, nil
			},
		}
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				p := newPermission(1, "read:users", tenantID)
				p.API = api
				return p, nil
			},
		}
		svc := NewPermissionService(db, permRepo, apiRepo, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.Create(context.Background(), tenantID, "read:users", "desc", model.StatusActive, false, apiUUID.String())
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "read:users", result.Name)
		assert.NotNil(t, result.API)
	})
}

// ---------------------------------------------------------------------------
// PermissionService.Update – transactional
// ---------------------------------------------------------------------------

func TestPermissionService_Update(t *testing.T) {
	tenantID := int64(1)
	permUUID := uuid.New()

	t.Run("find error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, errors.New("db err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Update(context.Background(), permUUID, tenantID, "new-name", "desc", model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return nil, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Update(context.Background(), permUUID, tenantID, "new-name", "desc", model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission not found")
	})

	t.Run("default permission → cannot update", func(t *testing.T) {
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
		_, err := svc.Update(context.Background(), permUUID, tenantID, "new-name", "desc", model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default permission")
	})

	t.Run("name changed findByName error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		perm := newPermission(1, "read:users", tenantID)
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
			findByNameFn: func(_ string, _ int64) (*model.Permission, error) {
				return nil, errors.New("name lookup err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Update(context.Background(), permUUID, tenantID, "write:users", "desc", model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name lookup err")
	})

	t.Run("name changed duplicate name → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		perm := newPermission(1, "read:users", tenantID)
		otherPerm := newPermission(2, "write:users", tenantID)
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
			findByNameFn: func(_ string, _ int64) (*model.Permission, error) {
				return otherPerm, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		_, err := svc.Update(context.Background(), permUUID, tenantID, "write:users", "desc", model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("createOrUpdate error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		perm := newPermission(1, "read:users", tenantID)
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
			createOrUpdateFn: func(_ *model.Permission) (*model.Permission, error) {
				return nil, errors.New("save err")
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		// Same name so no findByName path
		_, err := svc.Update(context.Background(), permUUID, tenantID, "read:users", "new desc", model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success – same name", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		perm := newPermission(1, "read:users", tenantID)
		perm.API = &model.API{APIUUID: uuid.New(), Name: "Users API"}
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.Update(context.Background(), permUUID, tenantID, "read:users", "updated desc", model.StatusActive)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "updated desc", result.Description)
	})

	t.Run("success – name changed no conflict", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		perm := newPermission(1, "read:users", tenantID)
		permRepo := &mockPermissionRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Permission, error) {
				return perm, nil
			},
			findByNameFn: func(_ string, _ int64) (*model.Permission, error) {
				return nil, nil // no conflict
			},
		}
		svc := NewPermissionService(db, permRepo, &mockAPIRepo{}, &mockRoleRepo{}, &mockClientRepo{})
		result, err := svc.Update(context.Background(), permUUID, tenantID, "write:users", "desc", model.StatusActive)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "write:users", result.Name)
	})
}

// ---------------------------------------------------------------------------
// toPermissionServiceDataResult – nil permission
// ---------------------------------------------------------------------------

func TestToPermissionServiceDataResult_Nil(t *testing.T) {
	result := toPermissionServiceDataResult(nil)
	assert.Nil(t, result)
}
