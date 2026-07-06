package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTenantMemberServiceForTest(db *gorm.DB, memberRepo TenantMemberRepository, userRepo UserReader, tenantRepo TenantRepository) TenantMemberService {
	return NewTenantMemberService(memberRepo, userRepo, tenantRepo, NewGormUnitOfWork(db, tenantRepo, memberRepo, nil), nil)
}

type testUserProvisioner struct {
	grantFn  func(context.Context, *gorm.DB, int64, int64, string) error
	revokeFn func(context.Context, *gorm.DB, int64, int64, string) error
}

func (p *testUserProvisioner) EnsureUserInTenant(_ context.Context, _ uuid.UUID, _ int64) (int64, error) {
	return 0, nil
}

func (p *testUserProvisioner) GrantRoleByName(ctx context.Context, tx *gorm.DB, userID, tenantID int64, role string) error {
	if p.grantFn != nil {
		return p.grantFn(ctx, tx, userID, tenantID, role)
	}
	return nil
}

func (p *testUserProvisioner) RevokeRoleByName(ctx context.Context, tx *gorm.DB, userID, tenantID int64, role string) error {
	if p.revokeFn != nil {
		return p.revokeFn(ctx, tx, userID, tenantID, role)
	}
	return nil
}

func TestTenantMemberService_GetByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByUUID(context.Background(), id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error is mapped to not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*TenantMember, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByUUID(context.Background(), id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: i, Role: "member"}, nil
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByTenantAndUser(context.Background(), 1, 2)
		require.Error(t, err)
	})

	t.Run("repo error is mapped to not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.GetByTenantAndUser(context.Background(), 1, 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		mid := uuid.New()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: mid, TenantID: 1, UserID: 2, Role: "admin"}, nil
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ uuid.UUID) (*MemberUser, error) { return nil, nil },
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("already a member", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) {
				return &TenantMember{}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ uuid.UUID) (*MemberUser, error) {
				return &MemberUser{UserID: 5}, nil
			},
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already a member")
	})

	t.Run("duplicate check error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) {
				return nil, errors.New("lookup failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ uuid.UUID) (*MemberUser, error) {
				return &MemberUser{UserID: 5}, nil
			},
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify tenant membership")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) { return nil, nil },
			createFn: func(e *TenantMember) (*TenantMember, error) {
				e.TenantMemberUUID = uuid.New()
				return e, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ uuid.UUID) (*MemberUser, error) {
				return &MemberUser{UserID: 5}, nil
			},
		}, &mockTenantRepo{})
		res, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member", 99)
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), 1, id, 99)
		require.Error(t, err)
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: i, TenantID: 1}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), 1, id, 99)
		require.NoError(t, err)
	})
}

func TestTenantMemberService_IsUserInTenant(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("tenant not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, errors.New("not found") },
		})
		ok, err := svc.IsUserInTenant(context.Background(), 1, tenantUUID)
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("user is member", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) {
				return &TenantMember{}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 10}, nil
			},
		})
		ok, err := svc.IsUserInTenant(context.Background(), 1, tenantUUID)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("user is not member", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 10}, nil
			},
		})
		ok, err := svc.IsUserInTenant(context.Background(), 1, tenantUUID)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("FindByTenantAndUser error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 10}, nil
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
	t.Run("regular tenant member cannot assign owner", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		memberRepo := &mockTenantMemberRepo{findByTenantAndUserFn: func(tenantID, userID int64) (*TenantMember, error) {
			if tenantID == 1 && userID == 10 {
				return &TenantMember{TenantID: tenantID, UserID: userID}, nil
			}
			return nil, nil
		}}
		svc := newTenantMemberServiceForTest(db, memberRepo, &mockUserRepo{}, &mockTenantRepo{})

		_, err := svc.Create(context.Background(), 1, 2, "owner", 10)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "only system tenant administrators")
	})

	t.Run("first owner completes tenant and receives super-admin atomically", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tenantRecord := &Tenant{TenantID: 1, Status: "pending"}
		completed := false
		tenantRepo := &mockTenantRepo{
			findByIDFn: func(any, ...string) (*Tenant, error) { return tenantRecord, nil },
			createOrUpdateFn: func(record *Tenant) (*Tenant, error) {
				completed = record.Status == "active"
				return record, nil
			},
		}
		memberRepo := &mockTenantMemberRepo{createFn: func(member *TenantMember) (*TenantMember, error) {
			member.TenantMemberUUID = uuid.New()
			return member, nil
		}}
		granted := false
		provisioner := &testUserProvisioner{grantFn: func(_ context.Context, _ *gorm.DB, userID, tenantID int64, role string) error {
			granted = true
			assert.Equal(t, int64(2), userID)
			assert.Equal(t, int64(1), tenantID)
			assert.Equal(t, "super-admin", role)
			return nil
		}}
		svc := NewTenantMemberService(memberRepo, &mockUserRepo{}, tenantRepo, NewGormUnitOfWork(db, tenantRepo, memberRepo, nil), nil, provisioner)

		result, err := svc.Create(context.Background(), 1, 2, "owner", 99)

		require.NoError(t, err)
		assert.Equal(t, "owner", result.Role)
		assert.True(t, granted)
		assert.True(t, completed)
	})

	t.Run("repo error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			createFn: func(_ *TenantMember) (*TenantMember, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.Create(context.Background(), 1, 2, "member", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		mid := uuid.New()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			createFn: func(e *TenantMember) (*TenantMember, error) {
				e.TenantMemberUUID = mid
				return e, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.Create(context.Background(), 1, 2, "admin", 99)
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantAndUserFn: func(_ int64, _ int64) (*TenantMember, error) { return nil, nil },
			createFn: func(_ *TenantMember) (*TenantMember, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ uuid.UUID) (*MemberUser, error) {
				return &MemberUser{UserID: 5, UserUUID: userUUID}, nil
			},
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("authevent.FindByUUID error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ uuid.UUID) (*MemberUser, error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantRepo{})
		_, err := svc.CreateByUserUUID(context.Background(), 1, userUUID, "member", 99)
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantFn: func(_ TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.ListByTenant(context.Background(), TenantMemberServiceListFilter{TenantID: 1, Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success with user lookup", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		mid := uuid.New()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantFn: func(_ TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error) {
				return &PaginationResult[TenantMember]{
					Data:  []TenantMember{{TenantMemberUUID: mid, TenantID: 1, UserID: 42, Role: "owner"}},
					Total: 1,
					Page:  1,
					Limit: 10,
				}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ int64) (*MemberUser, error) {
				return &MemberUser{UserID: 42, UserUUID: uuid.New(), Email: "a@b.com"}, nil
			},
		}, &mockTenantRepo{})
		res, err := svc.ListByTenant(context.Background(), TenantMemberServiceListFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		require.Len(t, res.Data, 1)
		assert.Equal(t, "owner", res.Data[0].Role)
		require.NotNil(t, res.Data[0].User)
		assert.Equal(t, "a@b.com", res.Data[0].User.Email)
	})

	t.Run("success user lookup fails gracefully", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantFn: func(_ TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error) {
				return &PaginationResult[TenantMember]{
					Data:  []TenantMember{{TenantMemberUUID: uuid.New(), TenantID: 1, UserID: 42, Role: "member"}},
					Total: 1,
					Page:  1,
					Limit: 10,
				}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ int64) (*MemberUser, error) {
				return nil, errors.New("user not found")
			},
		}, &mockTenantRepo{})
		res, err := svc.ListByTenant(context.Background(), TenantMemberServiceListFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		require.Len(t, res.Data, 1)
		assert.Nil(t, res.Data[0].User)
	})

	t.Run("empty list", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantFn: func(_ TenantMemberRepositoryListFilter) (*PaginationResult[TenantMember], error) {
				return &PaginationResult[TenantMember]{Data: []TenantMember{}, Page: 1, Limit: 10}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		res, err := svc.ListByTenant(context.Background(), TenantMemberServiceListFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Empty(t, res.Data)
	})
}

// ---------------------------------------------------------------------------
// ListByUser
// ---------------------------------------------------------------------------

func TestTenantMemberService_ListByUser(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findAllByUserFn: func(_ int64) ([]TenantMember, error) {
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findAllByUserFn: func(_ int64) ([]TenantMember, error) {
				return []TenantMember{
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findAllByUserFn: func(_ int64) ([]TenantMember, error) {
				return []TenantMember{}, nil
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*TenantMember, error) { return nil, nil },
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*TenantMember, error) {
				return nil, errors.New("find error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find error")
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(id uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: id, TenantID: 1, UserID: 5, Role: "member"}, nil
			},
			createOrUpdateFn: func(_ *TenantMember) (*TenantMember, error) {
				return nil, errors.New("update error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update error")
	})

	t.Run("cross-tenant member → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(id uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: id, TenantID: 99, UserID: 5, Role: "member"}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		_, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success with user populated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(id uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: id, TenantID: 1, UserID: 5, Role: "member"}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ int64) (*MemberUser, error) {
				return &MemberUser{UserID: 5, Email: "test@test.com"}, nil
			},
		}, &mockTenantRepo{})
		res, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)
		require.NoError(t, err)
		assert.Equal(t, "owner", res.Role)
		require.NotNil(t, res.User)
		assert.Equal(t, "test@test.com", res.User.Email)
	})

	t.Run("success user lookup fails gracefully", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(id uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: id, TenantID: 1, UserID: 5, Role: "member"}, nil
			},
		}, &mockUserRepo{
			findByIDFn: func(_ int64) (*MemberUser, error) {
				return nil, errors.New("user gone")
			},
		}, &mockTenantRepo{})
		res, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)
		require.NoError(t, err)
		assert.Equal(t, "owner", res.Role)
		assert.Nil(t, res.User)
	})

	t.Run("ownership transfer updates membership and IAM role atomically", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		oldOwner := &TenantMember{TenantMemberUUID: uuid.New(), TenantID: 1, UserID: 4, Role: "owner"}
		newOwner := &TenantMember{TenantMemberUUID: tmUUID, TenantID: 1, UserID: 5, Role: "member"}
		var updatedRoles []string
		memberRepo := &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(uuid.UUID) (*TenantMember, error) { return newOwner, nil },
			findOwnerByTenantIDFn:    func(int64) (*TenantMember, error) { return oldOwner, nil },
			createOrUpdateFn: func(member *TenantMember) (*TenantMember, error) {
				updatedRoles = append(updatedRoles, member.Role)
				return member, nil
			},
		}
		var revoked, granted int64
		provisioner := &testUserProvisioner{
			revokeFn: func(_ context.Context, _ *gorm.DB, userID, tenantID int64, role string) error {
				revoked = userID
				assert.Equal(t, int64(1), tenantID)
				assert.Equal(t, "super-admin", role)
				return nil
			},
			grantFn: func(_ context.Context, _ *gorm.DB, userID, tenantID int64, role string) error {
				granted = userID
				assert.Equal(t, int64(1), tenantID)
				assert.Equal(t, "super-admin", role)
				return nil
			},
		}
		svc := NewTenantMemberService(memberRepo, &mockUserRepo{
			findByIDFn: func(id int64) (*MemberUser, error) {
				return &MemberUser{UserID: id, UserUUID: uuid.New()}, nil
			},
		}, &mockTenantRepo{}, NewGormUnitOfWork(db, &mockTenantRepo{}, memberRepo, nil), nil, provisioner)

		result, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)

		require.NoError(t, err)
		assert.Equal(t, "owner", result.Role)
		assert.Equal(t, []string{"member", "owner"}, updatedRoles)
		assert.Equal(t, int64(4), revoked)
		assert.Equal(t, int64(5), granted)
	})

	t.Run("system tenant ownership cannot be transferred", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		memberRepo := &mockTenantMemberRepo{findByTenantMemberUUIDFn: func(uuid.UUID) (*TenantMember, error) {
			return &TenantMember{TenantMemberUUID: tmUUID, TenantID: 1, UserID: 5, Role: "member"}, nil
		}}
		tenantRepo := &mockTenantRepo{findByIDFn: func(any, ...string) (*Tenant, error) {
			return &Tenant{TenantID: 1, IsSystem: true}, nil
		}}
		svc := NewTenantMemberService(memberRepo, &mockUserRepo{}, tenantRepo, NewGormUnitOfWork(db, tenantRepo, memberRepo, nil), nil)

		_, err := svc.UpdateRole(context.Background(), 1, tmUUID, "owner", 99)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot transfer ownership of the system tenant")
	})

	t.Run("owner cannot be directly demoted", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		memberRepo := &mockTenantMemberRepo{findByTenantMemberUUIDFn: func(uuid.UUID) (*TenantMember, error) {
			return &TenantMember{TenantMemberUUID: tmUUID, TenantID: 1, UserID: 5, Role: "owner"}, nil
		}}
		svc := newTenantMemberServiceForTest(db, memberRepo, &mockUserRepo{}, &mockTenantRepo{})

		_, err := svc.UpdateRole(context.Background(), 1, tmUUID, "member", 99)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "transfer ownership")
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
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(_ uuid.UUID) (*TenantMember, error) {
				return nil, errors.New("find error")
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), 1, id, 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find error")
	})

	t.Run("delete error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: i, TenantID: 1}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), 1, id, 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("cross-tenant member → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: i, TenantID: 99}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})
		err := svc.DeleteByUUID(context.Background(), 1, id, 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("owner cannot be removed directly", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := newTenantMemberServiceForTest(db, &mockTenantMemberRepo{
			findByTenantMemberUUIDFn: func(i uuid.UUID) (*TenantMember, error) {
				return &TenantMember{TenantMemberUUID: i, TenantID: 1, Role: "owner"}, nil
			},
		}, &mockUserRepo{}, &mockTenantRepo{})

		err := svc.DeleteByUUID(context.Background(), 1, id, 99)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "transfer ownership")
	})
}

func TestTenantMemberService_NewWithNilUnitOfWork(t *testing.T) {
	svc := NewTenantMemberService(&mockTenantMemberRepo{}, &mockUserRepo{}, &mockTenantRepo{}, nil, nil)

	result, err := svc.Create(context.Background(), 1, 2, "member", 99)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.TenantID)
}
