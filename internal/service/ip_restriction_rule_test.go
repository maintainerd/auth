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

func newIPRuleSvc(repo *mockIPRestrictionRuleRepo) IPRestrictionRuleService {
	return NewIPRestrictionRuleService(nil, repo)
}

func TestIPRestrictionRuleService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findPaginatedFn: func(_ repository.IPRestrictionRuleRepositoryGetFilter) (*repository.PaginationResult[model.IPRestrictionRule], error) {
				return &repository.PaginationResult[model.IPRestrictionRule]{
					Data:  []model.IPRestrictionRule{{IPAddress: "192.168.1.1", TenantID: 1}},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		})
		res, err := svc.GetAll(1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findPaginatedFn: func(_ repository.IPRestrictionRuleRepositoryGetFilter) (*repository.PaginationResult[model.IPRestrictionRule], error) {
				return nil, errors.New("db err")
			},
		})
		_, err := svc.GetAll(1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})
}

func TestIPRestrictionRuleService_GetByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	cases := []struct {
		name    string
		repoFn  func(any, ...string) (*model.IPRestrictionRule, error)
		wantErr string
	}{
		{
			name:    "not found",
			repoFn:  func(_ any, _ ...string) (*model.IPRestrictionRule, error) { return nil, nil },
			wantErr: "not found",
		},
		{
			name: "wrong tenant",
			repoFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{TenantID: 999, IPAddress: "1.1.1.1"}, nil
			},
			wantErr: "not found",
		},
		{
			name:    "repo error",
			repoFn:  func(_ any, _ ...string) (*model.IPRestrictionRule, error) { return nil, errors.New("db") },
			wantErr: "db",
		},
		{
			name: "success",
			repoFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{
					IPRestrictionRuleUUID: id, TenantID: tid, IPAddress: "192.168.1.1",
				}, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{findByUUIDFn: tc.repoFn})
			res, err := svc.GetByUUID(tid, id)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, id, res.IPRestrictionRuleUUID)
			}
		})
	}
}

func TestIPRestrictionRuleService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			createFn: func(e *model.IPRestrictionRule) (*model.IPRestrictionRule, error) { return e, nil },
		})
		res, err := svc.Create(1, "block malicious", "blacklist", "10.0.0.1", "active", 42)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.1", res.IPAddress)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			createFn: func(_ *model.IPRestrictionRule) (*model.IPRestrictionRule, error) { return nil, errors.New("fail") },
		})
		_, err := svc.Create(1, "d", "blacklist", "10.0.0.1", "active", 42)
		require.Error(t, err)
	})
}

func TestIPRestrictionRuleService_Update(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("not found", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) { return nil, nil },
		})
		_, err := svc.Update(tid, id, "d", "blacklist", "10.0.0.1", "active", 1)
		require.Error(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return nil, errors.New("db err")
			},
		})
		_, err := svc.Update(tid, id, "d", "blacklist", "10.0.0.1", "active", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("wrong tenant", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{TenantID: 999}, nil
			},
		})
		_, err := svc.Update(tid, id, "d", "blacklist", "10.0.0.1", "active", 1)
		require.Error(t, err)
	})

	t.Run("update error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPRestrictionRuleUUID: id, TenantID: tid}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.IPRestrictionRule, error) {
				return nil, errors.New("update err")
			},
		})
		_, err := svc.Update(tid, id, "d", "blacklist", "10.0.0.1", "active", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("success", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPRestrictionRuleUUID: id, TenantID: tid}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPAddress: "10.0.0.1", TenantID: tid}, nil
			},
		})
		res, err := svc.Update(tid, id, "d", "blacklist", "10.0.0.1", "active", 1)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.1", res.IPAddress)
	})
}

func TestIPRestrictionRuleService_UpdateStatus(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("repo error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return nil, errors.New("db err")
			},
		})
		_, err := svc.UpdateStatus(tid, id, "inactive", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) { return nil, nil },
		})
		_, err := svc.UpdateStatus(tid, id, "inactive", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("wrong tenant", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{TenantID: 999}, nil
			},
		})
		_, err := svc.UpdateStatus(tid, id, "inactive", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("update error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPRestrictionRuleUUID: id, TenantID: tid}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.IPRestrictionRule, error) {
				return nil, errors.New("update err")
			},
		})
		_, err := svc.UpdateStatus(tid, id, "inactive", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("success", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPRestrictionRuleUUID: id, TenantID: tid}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPRestrictionRuleUUID: id, TenantID: tid, Status: "inactive"}, nil
			},
		})
		res, err := svc.UpdateStatus(tid, id, "inactive", 1)
		require.NoError(t, err)
		assert.Equal(t, "inactive", res.Status)
	})
}

func TestIPRestrictionRuleService_Delete(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("repo error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return nil, errors.New("db err")
			},
		})
		_, err := svc.Delete(tid, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) { return nil, nil },
		})
		_, err := svc.Delete(tid, id)
		require.Error(t, err)
	})

	t.Run("wrong tenant", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{TenantID: 999}, nil
			},
		})
		_, err := svc.Delete(tid, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete error", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPRestrictionRuleUUID: id, TenantID: tid, IPAddress: "1.2.3.4"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete err") },
		})
		_, err := svc.Delete(tid, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete err")
	})

	t.Run("success", func(t *testing.T) {
		svc := newIPRuleSvc(&mockIPRestrictionRuleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IPRestrictionRule, error) {
				return &model.IPRestrictionRule{IPRestrictionRuleUUID: id, TenantID: tid, IPAddress: "1.2.3.4"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		})
		res, err := svc.Delete(tid, id)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3.4", res.IPAddress)
	})
}
