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

func newPolicyService(policyRepo *mockPolicyRepo, serviceRepo *mockServiceRepo, apiRepo *mockAPIRepo) PolicyService {
	return NewPolicyService(nil, policyRepo, serviceRepo, apiRepo)
}

func newPolicy(tenantID int64, name, version string) *model.Policy {
	return &model.Policy{
		PolicyID:   1,
		PolicyUUID: uuid.New(),
		TenantID:   tenantID,
		Name:       name,
		Version:    version,
		Status:     model.StatusActive,
	}
}

// ---------------------------------------------------------------------------
// PolicyService.Get
// ---------------------------------------------------------------------------

func TestPolicyService_Get(t *testing.T) {
	tenantID := int64(1)

	t.Run("repo error → propagated", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findPaginatedFn: func(_ repository.PolicyRepositoryGetFilter) (*repository.PaginationResult[model.Policy], error) {
				return nil, errors.New("db error")
			},
		}
		svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.Get(PolicyServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success → returns mapped results", func(t *testing.T) {
		p := newPolicy(tenantID, "read-only", "v1")
		policyRepo := &mockPolicyRepo{
			findPaginatedFn: func(_ repository.PolicyRepositoryGetFilter) (*repository.PaginationResult[model.Policy], error) {
				return &repository.PaginationResult[model.Policy]{
					Data: []model.Policy{*p}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		result, err := svc.Get(PolicyServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, p.Name, result.Data[0].Name)
	})
}

// ---------------------------------------------------------------------------
// PolicyService.GetServicesByPolicyUUID
// ---------------------------------------------------------------------------

func TestPolicyService_GetServicesByPolicyUUID(t *testing.T) {
	tenantID := int64(1)
	policyUUID := uuid.New()

	t.Run("policy lookup error → propagated", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.GetServicesByPolicyUUID(policyUUID, tenantID, PolicyServiceServicesFilter{Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("success → returns services", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return newPolicy(tenantID, "read-only", "v1"), nil
			},
		}
		svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		result, err := svc.GetServicesByPolicyUUID(policyUUID, tenantID, PolicyServiceServicesFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// ---------------------------------------------------------------------------
// PolicyService.GetByUUID
// ---------------------------------------------------------------------------

func TestPolicyService_GetByUUID(t *testing.T) {
	tenantID := int64(1)
	policyUUID := uuid.New()

	cases := []struct {
		name        string
		setupRepo   func(r *mockPolicyRepo)
		expectError bool
		errContains string
	}{
		{
			name: "repo error → propagated",
			setupRepo: func(r *mockPolicyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Policy, error) {
					return nil, errors.New("db error")
				}
			},
			expectError: true,
			errContains: "db error",
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockPolicyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Policy, error) { return nil, nil }
			},
			expectError: true,
			errContains: "policy not found",
		},
		{
			name: "found → success",
			setupRepo: func(r *mockPolicyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.Policy, error) {
					return newPolicy(tenantID, "read-only", "v1"), nil
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policyRepo := &mockPolicyRepo{}
			tc.setupRepo(policyRepo)
			svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
			result, err := svc.GetByUUID(policyUUID, tenantID)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "read-only", result.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// PolicyService.Create – transactional
// ---------------------------------------------------------------------------

func TestPolicyService_Create(t *testing.T) {
	tenantID := int64(1)

	t.Run("policy name+version already exists → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByNameAndVersionFn: func(_, _ string, _ int64) (*model.Policy, error) {
				return newPolicy(tenantID, "read-only", "v1"), nil
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.Create(tenantID, "read-only", nil, nil, "v1", model.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			createFn: func(_ *model.Policy) (*model.Policy, error) {
				return nil, errors.New("insert failed")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.Create(tenantID, "new-policy", nil, nil, "v1", model.StatusActive, false)
		require.Error(t, err)
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewPolicyService(db, &mockPolicyRepo{}, &mockServiceRepo{}, &mockAPIRepo{})
		result, err := svc.Create(tenantID, "new-policy", nil, nil, "v1", model.StatusActive, false)
		require.NoError(t, err)
		assert.Equal(t, "new-policy", result.Name)
	})
}

// ---------------------------------------------------------------------------
// PolicyService.SetStatusByUUID – transactional
// ---------------------------------------------------------------------------

func TestPolicyService_SetStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	policyUUID := uuid.New()

	t.Run("policy not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) { return nil, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.SetStatusByUUID(policyUUID, tenantID, model.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system policy → cannot update status → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "sys", "v1")
		p.IsSystem = true
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.SetStatusByUUID(policyUUID, tenantID, model.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system policy")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		p := newPolicy(tenantID, "read-only", "v1")
		callCount := 0
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				callCount++
				updated := *p
				updated.Status = model.StatusInactive
				return &updated, nil
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		result, err := svc.SetStatusByUUID(policyUUID, tenantID, model.StatusInactive)
		require.NoError(t, err)
		assert.Equal(t, model.StatusInactive, result.Status)
	})
}

// ---------------------------------------------------------------------------
// PolicyService.DeleteByUUID – transactional
// ---------------------------------------------------------------------------

func TestPolicyService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	policyUUID := uuid.New()

	t.Run("policy not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) { return nil, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.DeleteByUUID(policyUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system policy → cannot delete → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "sys", "v1")
		p.IsSystem = true
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.DeleteByUUID(policyUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		p := newPolicy(tenantID, "read-only", "v1")
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		result, err := svc.DeleteByUUID(policyUUID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, p.Name, result.Name)
	})
}
