package service

import (
	"errors"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantMemberService_GetByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*model.TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByUUID(context.Background(), id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*model.TenantMember, error) {
				return &model.TenantMember{TenantMemberUUID: i, Role: "member"}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.GetByUUID(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, id, res.TenantMemberUUID)
		assert.Equal(t, "member", res.Role)
	})
}

func TestTenantMemberService_GetByTenantAndUser(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByTenantAndUser(context.Background(), 1, 2)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		mid := uuid.New()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) {
				return &model.TenantMember{TenantMemberUUID: mid, TenantID: 1, UserID: 2, Role: "admin"}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.GetByTenantAndUser(context.Background(), 1, 2)
		require.NoError(t, err)
		assert.Equal(t, "admin", res.Role)
	})
}

func TestTenantMemberService_CreateByUserUUID(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("already a member", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) {
				return &model.TenantMember{}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 5}, nil
			},
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already a member")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) { return nil, nil },
			createFn: func(e *model.TenantMember) (*model.TenantMember, error) {
				e.TenantMemberUUID = uuid.New()
				return e, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 5}, nil
			},
		}, &mockTenantRepo{})
		res, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member")
		require.NoError(t, err)
		assert.Equal(t, int64(5), res.UserID)
	})
}

func TestTenantMemberService_DeleteByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*model.TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), id)
		require.Error(t, err)
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*model.TenantMember, error) {
				return &model.TenantMember{TenantMemberUUID: i}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), id)
		require.NoError(t, err)
	})
}

func TestTenantMemberService_IsUserInTenant(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("tenant not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, errors.New("not found") },
		})
		ok, err := svc.IsUserInTenant(context.Background(), 1, tenantUUID)
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("user is member", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) {
				return &model.TenantMember{}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 10}, nil
			},
		})
		ok, err := svc.IsUserInTenant(context.Background(), 1, tenantUUID)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("user is not member", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 10}, nil
			},
		})
		ok, err := svc.IsUserInTenant(context.Background(), 1, tenantUUID)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("FindByTenantAndUser error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 10}, nil
			},
		})
		ok, err := svc.IsUserInTenant(context.Background(), 1, tenantUUID)
		require.Error(t, err)
		assert.False(t, ok)
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestTenantMemberService_Create(t *testing.T) {
	t.Run("repo error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			createFn: func(_ *model.TenantMember) (*model.TenantMember, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.Create(context.Background(), 1, 2, "member")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		mid := uuid.New()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			createFn: func(e *model.TenantMember) (*model.TenantMember, error) {
				e.TenantMemberUUID = mid
				return e, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.Create(context.Background(), 1, 2, "admin")
		require.NoError(t, err)
		assert.Equal(t, mid, res.TenantMemberUUID)
		assert.Equal(t, "admin", res.Role)
	})
}

// ---------------------------------------------------------------------------
// CreateByUserUUID – extra paths
// ---------------------------------------------------------------------------

func TestTenantMemberService_CreateByUserUUID_Extra(t *testing.T) {
	userUUID := uuid.New()

	t.Run("Create error after user found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*model.TenantMember, error) { return nil, nil },
			createFn: func(_ *model.TenantMember) (*model.TenantMember, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 5, UserUUID: userUUID}, nil
			},
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("FindByUUID error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

// ---------------------------------------------------------------------------
// ListByTenant
// ---------------------------------------------------------------------------

func TestTenantMemberService_ListByTenant(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findAllByTenantFn: func(_ int64) ([]model.TenantMember, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.ListByTenant(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success with user lookup", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		mid := uuid.New()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findAllByTenantFn: func(_ int64) ([]model.TenantMember, error) {
				return []model.TenantMember{
					{TenantMemberUUID: mid, TenantID: 1, UserID: 42, Role: "admin"},
				}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 42, UserUUID: uuid.New(), Email: "a@b.com"}, nil
			},
		}, &mockTenantRepo{})
		res, err := svc.ListByTenant(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "admin", res[0].Role)
		require.NotNil(t, res[0].User)
		assert.Equal(t, "a@b.com", res[0].User.Email)
	})

	t.Run("success user lookup fails gracefully", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findAllByTenantFn: func(_ int64) ([]model.TenantMember, error) {
				return []model.TenantMember{
					{TenantMemberUUID: uuid.New(), TenantID: 1, UserID: 42, Role: "member"},
				}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return nil, errors.New("user not found")
			},
		}, &mockTenantRepo{})
		res, err := svc.ListByTenant(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Nil(t, res[0].User)
	})

	t.Run("empty list", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findAllByTenantFn: func(_ int64) ([]model.TenantMember, error) {
				return []model.TenantMember{}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.ListByTenant(context.Background(), 1)
		require.NoError(t, err)
		assert.Empty(t, res)
	})
}

// ---------------------------------------------------------------------------
// ListByUser
// ---------------------------------------------------------------------------

func TestTenantMemberService_ListByUser(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findAllByUserFn: func(_ int64) ([]model.TenantMember, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.ListByUser(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		mid := uuid.New()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findAllByUserFn: func(_ int64) ([]model.TenantMember, error) {
				return []model.TenantMember{
					{TenantMemberUUID: mid, TenantID: 1, UserID: 5, Role: "member"},
					{TenantMemberUUID: uuid.New(), TenantID: 2, UserID: 5, Role: "admin"},
				}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.ListByUser(context.Background(), 5)
		require.NoError(t, err)
		require.Len(t, res, 2)
		assert.Equal(t, mid, res[0].TenantMemberUUID)
		assert.Equal(t, "admin", res[1].Role)
	})

	t.Run("empty list", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findAllByUserFn: func(_ int64) ([]model.TenantMember, error) {
				return []model.TenantMember{}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.ListByUser(context.Background(), 1)
		require.NoError(t, err)
		assert.Empty(t, res)
	})
}

// ---------------------------------------------------------------------------
// UpdateRole
// ---------------------------------------------------------------------------

func TestTenantMemberService_UpdateRole(t *testing.T) {
	tmUUID := uuid.New()

	t.Run("not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*model.TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateRole(context.Background(), tmUUID, "admin")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*model.TenantMember, error) {
				return nil, errors.New("find error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateRole(context.Background(), tmUUID, "admin")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find error")
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(id uuid.UUID) (*model.TenantMember, error) {
				return &model.TenantMember{TenantMemberUUID: id, UserID: 5, Role: "member"}, nil
			},
			createOrUpdateFn: func(_ *model.TenantMember) (*model.TenantMember, error) {
				return nil, errors.New("update error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateRole(context.Background(), tmUUID, "admin")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update error")
	})

	t.Run("success with user populated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(id uuid.UUID) (*model.TenantMember, error) {
				return &model.TenantMember{TenantMemberUUID: id, UserID: 5, Role: "member"}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 5, Email: "test@test.com"}, nil
			},
		}, &mockTenantRepo{})
		res, err := svc.UpdateRole(context.Background(), tmUUID, "admin")
		require.NoError(t, err)
		assert.Equal(t, "admin", res.Role)
		require.NotNil(t, res.User)
		assert.Equal(t, "test@test.com", res.User.Email)
	})

	t.Run("success user lookup fails gracefully", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(id uuid.UUID) (*model.TenantMember, error) {
				return &model.TenantMember{TenantMemberUUID: id, UserID: 5, Role: "member"}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return nil, errors.New("user gone")
			},
		}, &mockTenantRepo{})
		res, err := svc.UpdateRole(context.Background(), tmUUID, "admin")
		require.NoError(t, err)
		assert.Equal(t, "admin", res.Role)
		assert.Nil(t, res.User)
	})
}

// ---------------------------------------------------------------------------
// DeleteByUUID – extra paths
// ---------------------------------------------------------------------------

func TestTenantMemberService_DeleteByUUID_Extra(t *testing.T) {
	id := uuid.New()

	t.Run("find error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*model.TenantMember, error) {
				return nil, errors.New("find error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find error")
	})

	t.Run("delete error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantMemberService(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*model.TenantMember, error) {
				return &model.TenantMember{TenantMemberUUID: i}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})
}
