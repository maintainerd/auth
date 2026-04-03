package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildUserService(t *testing.T, userRepo *mockUserRepo, roleRepo *mockRoleRepo, userRoleRepo *mockUserRoleRepo) UserService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewUserService(db, userRepo, &mockUserIdentityRepo{}, userRoleRepo, roleRepo,
		&mockTenantRepo{}, &mockIdentityProviderRepo{}, &mockClientRepo{}, &mockTenantUserRepo{})
}

func TestUserService_Get(t *testing.T) {
	t.Run("invalid role UUID returns error", func(t *testing.T) {
		svc := buildUserService(t, &mockUserRepo{}, &mockRoleRepo{}, &mockUserRoleRepo{})
		bad := "not-a-uuid"
		_, err := svc.Get(UserServiceGetFilter{RoleUUID: &bad, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role UUID")
	})

	t.Run("role not found returns error", func(t *testing.T) {
		roleRepo := &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}
		svc := buildUserService(t, &mockUserRepo{}, roleRepo, &mockUserRoleRepo{})
		rid := uuid.New().String()
		_, err := svc.Get(UserServiceGetFilter{RoleUUID: &rid, TenantID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("paginate error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findPaginatedFn: func(_ repository.UserRepositoryGetFilter) (*repository.PaginationResult[model.User], error) {
				return nil, errors.New("db error")
			},
		}
		svc := buildUserService(t, userRepo, &mockRoleRepo{}, &mockUserRoleRepo{})
		_, err := svc.Get(UserServiceGetFilter{TenantID: 1})
		require.Error(t, err)
	})

	t.Run("success returns empty list", func(t *testing.T) {
		svc := buildUserService(t, &mockUserRepo{}, &mockRoleRepo{}, &mockUserRoleRepo{})
		res, err := svc.Get(UserServiceGetFilter{TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestUserService_GetByUUID(t *testing.T) {
	uid := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := buildUserService(t, userRepo, &mockRoleRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetByUUID(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no tenant access", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{
					UserID:         1,
					UserIdentities: []model.UserIdentity{{TenantID: 99}},
				}, nil
			},
		}
		svc := buildUserService(t, userRepo, &mockRoleRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetByUUID(uid, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("success", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{
					UserID:         1,
					UserIdentities: []model.UserIdentity{{TenantID: 1}},
				}, nil
			},
		}
		svc := buildUserService(t, userRepo, &mockRoleRepo{}, &mockUserRoleRepo{})
		res, err := svc.GetByUUID(uid, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestUserService_GetUserRoles(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		userRepo := &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}
		svc := buildUserService(t, userRepo, &mockRoleRepo{}, &mockUserRoleRepo{})
		_, err := svc.GetUserRoles(uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

