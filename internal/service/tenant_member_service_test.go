package service

import (
	"errors"
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
		_, err := svc.GetByUUID(id)
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
		res, err := svc.GetByUUID(id)
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
		_, err := svc.GetByTenantAndUser(1, 2)
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
		res, err := svc.GetByTenantAndUser(1, 2)
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
		_, err := svc.CreateByUserUUID(1, userUUID, "member")
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
		_, err := svc.CreateByUserUUID(1, userUUID, "member")
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
		res, err := svc.CreateByUserUUID(1, userUUID, "member")
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
		err := svc.DeleteByUUID(id)
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
		err := svc.DeleteByUUID(id)
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
		ok, err := svc.IsUserInTenant(1, tenantUUID)
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
		ok, err := svc.IsUserInTenant(1, tenantUUID)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}
