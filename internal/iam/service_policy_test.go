package iam

import (
	"context"
	"errors"
	"gorm.io/datatypes"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPolicyService(policyRepo *mockPolicyRepo, serviceRepo *mockServiceRepo, apiRepo *mockAPIRepo) PolicyService {
	return NewPolicyService(nil, policyRepo, serviceRepo, apiRepo, nil)
}

func newPolicy(tenantID int64, name, version string) *Policy {
	return &Policy{
		PolicyID:   1,
		PolicyUUID: uuid.New(),
		TenantID:   tenantID,
		Name:       name,
		Version:    version,
		Status:     shared.StatusActive,
	}
}

// ---------------------------------------------------------------------------
// PolicyService.Get
// ---------------------------------------------------------------------------

func TestPolicyService_Get(t *testing.T) {
	tenantID := int64(1)

	t.Run("repo error → propagated", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findPaginatedFn: func(_ PolicyRepositoryGetFilter) (*PaginationResult[Policy], error) {
				return nil, errors.New("db error")
			},
		}
		svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.Get(context.Background(), PolicyServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success → returns mapped results", func(t *testing.T) {
		p := newPolicy(tenantID, "read-only", "v1")
		policyRepo := &mockPolicyRepo{
			findPaginatedFn: func(_ PolicyRepositoryGetFilter) (*PaginationResult[Policy], error) {
				return &PaginationResult[Policy]{
					Data: []Policy{*p}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		result, err := svc.Get(context.Background(), PolicyServiceGetFilter{TenantID: tenantID, Page: 1, Limit: 10})
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
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newPolicyService(policyRepo, &mockServiceRepo{}, &mockAPIRepo{})
		_, err := svc.GetServicesByPolicyUUID(context.Background(), policyUUID, tenantID, PolicyServiceServicesFilter{Page: 1, Limit: 10})
		require.Error(t, err)
	})

	// FindByUUIDAndTenantID reports not-found as (nil, nil). The caller checked only
	// err, so another tenant's policy UUID fell through to the service listing and
	// returned that tenant's service names, UUIDs and descriptions.
	t.Run("foreign policy UUID → not found, no service lookup", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return nil, nil },
		}
		serviceRepo := &mockServiceRepo{
			findServicesByPolicyUUIDFn: func(_ uuid.UUID, _ ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
				t.Fatal("services must not be listed for a policy the tenant does not own")
				return nil, nil
			},
		}
		svc := newPolicyService(policyRepo, serviceRepo, &mockAPIRepo{})

		result, err := svc.GetServicesByPolicyUUID(context.Background(), policyUUID, tenantID, PolicyServiceServicesFilter{Page: 1, Limit: 10})

		assert.Nil(t, result)
		require.ErrorContains(t, err, "not found")
	})

	// Owning the POLICY does not prove every service linked to it belongs to the
	// caller, so the tenant is carried into the repository query too.
	t.Run("carries the tenant into the repository filter", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				return newPolicy(tenantID, "read-only", "v1"), nil
			},
		}
		var gotFilter ServiceRepositoryGetFilter
		serviceRepo := &mockServiceRepo{
			findServicesByPolicyUUIDFn: func(_ uuid.UUID, filter ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
				gotFilter = filter
				return &PaginationResult[Service]{Page: 1, Limit: 10}, nil
			},
		}
		svc := newPolicyService(policyRepo, serviceRepo, &mockAPIRepo{})

		_, err := svc.GetServicesByPolicyUUID(context.Background(), policyUUID, tenantID, PolicyServiceServicesFilter{Page: 1, Limit: 10})

		require.NoError(t, err)
		require.NotNil(t, gotFilter.TenantID)
		assert.Equal(t, tenantID, *gotFilter.TenantID)
	})

	t.Run("FindServicesByPolicyUUID error → propagated", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				return newPolicy(tenantID, "read-only", "v1"), nil
			},
		}
		serviceRepo := &mockServiceRepo{
			findServicesByPolicyUUIDFn: func(_ uuid.UUID, _ ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
				return nil, errors.New("svc repo error")
			},
		}
		svc := newPolicyService(policyRepo, serviceRepo, &mockAPIRepo{})
		_, err := svc.GetServicesByPolicyUUID(context.Background(), policyUUID, tenantID, PolicyServiceServicesFilter{Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "svc repo error")
	})

	t.Run("success → returns services with counts", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				return newPolicy(tenantID, "read-only", "v1"), nil
			},
		}
		svc1 := Service{ServiceID: 10, ServiceUUID: uuid.New(), Name: "svc-1", DisplayName: "Svc 1", Status: shared.StatusActive}
		serviceRepo := &mockServiceRepo{
			findServicesByPolicyUUIDFn: func(_ uuid.UUID, _ ServiceRepositoryGetFilter) (*PaginationResult[Service], error) {
				return &PaginationResult[Service]{
					Data: []Service{svc1}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
			countPoliciesByServiceIDFn: func(_ int64) (int64, error) { return 3, nil },
		}
		apiRepo := &mockAPIRepo{
			countByServiceIDFn: func(_ int64, _ int64) (int64, error) { return 5, nil },
		}
		svc := newPolicyService(policyRepo, serviceRepo, apiRepo)
		result, err := svc.GetServicesByPolicyUUID(context.Background(), policyUUID, tenantID, PolicyServiceServicesFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		require.Len(t, result.Data, 1)
		assert.Equal(t, svc1.Name, result.Data[0].Name)
		assert.Equal(t, int64(5), result.Data[0].APICount)
		assert.Equal(t, int64(3), result.Data[0].PolicyCount)
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
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*Policy, error) {
					return nil, errors.New("db error")
				}
			},
			expectError: true,
			errContains: "db error",
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockPolicyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*Policy, error) { return nil, nil }
			},
			expectError: true,
			errContains: "policy not found",
		},
		{
			name: "found → success",
			setupRepo: func(r *mockPolicyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*Policy, error) {
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
			result, err := svc.GetByUUID(context.Background(), policyUUID, tenantID)
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

	t.Run("FindByNameAndVersion error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByNameAndVersionFn: func(_, _ string, _ int64) (*Policy, error) {
				return nil, errors.New("lookup failed")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Create(context.Background(), tenantID, "p", nil, nil, "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup failed")
	})

	t.Run("policy name+version already exists → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByNameAndVersionFn: func(_, _ string, _ int64) (*Policy, error) {
				return newPolicy(tenantID, "read-only", "v1"), nil
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Create(context.Background(), tenantID, "read-only", nil, nil, "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			createFn: func(_ *Policy) (*Policy, error) {
				return nil, errors.New("insert failed")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Create(context.Background(), tenantID, "new-policy", nil, nil, "v1", shared.StatusActive, false)
		require.Error(t, err)
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewPolicyService(db, &mockPolicyRepo{}, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		result, err := svc.Create(context.Background(), tenantID, "new-policy", nil, nil, "v1", shared.StatusActive, false)
		require.NoError(t, err)
		assert.Equal(t, "new-policy", result.Name)
	})
}

// ---------------------------------------------------------------------------
// PolicyService.Update – transactional
// ---------------------------------------------------------------------------

func TestPolicyService_Update(t *testing.T) {
	tenantID := int64(1)
	policyUUID := uuid.New()

	t.Run("find error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Update(context.Background(), policyUUID, tenantID, "n", nil, nil, "v1", shared.StatusActive, PolicyChangeContext{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return nil, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Update(context.Background(), policyUUID, tenantID, "n", nil, nil, "v1", shared.StatusActive, PolicyChangeContext{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system policy → cannot update → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "sys", "v1")
		p.IsSystem = true
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Update(context.Background(), policyUUID, tenantID, "n", nil, nil, "v1", shared.StatusActive, PolicyChangeContext{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system policy")
	})

	t.Run("name changed → FindByNameAndVersion error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "old-name", "v1")
		p.PolicyUUID = policyUUID
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
			findByNameAndVersionFn: func(_, _ string, _ int64) (*Policy, error) {
				return nil, errors.New("lookup error")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Update(context.Background(), policyUUID, tenantID, "new-name", nil, nil, "v1", shared.StatusActive, PolicyChangeContext{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup error")
	})

	t.Run("name changed → duplicate exists → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "old-name", "v1")
		p.PolicyUUID = policyUUID
		otherUUID := uuid.New()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
			findByNameAndVersionFn: func(_, _ string, _ int64) (*Policy, error) {
				dup := newPolicy(tenantID, "new-name", "v1")
				dup.PolicyUUID = otherUUID
				return dup, nil
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Update(context.Background(), policyUUID, tenantID, "new-name", nil, nil, "v1", shared.StatusActive, PolicyChangeContext{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("UpdateByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "read-only", "v1")
		p.PolicyUUID = policyUUID
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
			updateByUUIDFn: func(_, _ any) (*Policy, error) {
				return nil, errors.New("update failed")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.Update(context.Background(), policyUUID, tenantID, "read-only", nil, nil, "v1", shared.StatusActive, PolicyChangeContext{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("success same name/version → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		p := newPolicy(tenantID, "read-only", "v1")
		p.PolicyUUID = policyUUID
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		result, err := svc.Update(context.Background(), policyUUID, tenantID, "read-only", nil, nil, "v1", shared.StatusActive, PolicyChangeContext{})
		require.NoError(t, err)
		assert.Equal(t, "read-only", result.Name)
	})

	t.Run("success name changed no conflict → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		p := newPolicy(tenantID, "old-name", "v1")
		p.PolicyUUID = policyUUID
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		result, err := svc.Update(context.Background(), policyUUID, tenantID, "new-name", nil, nil, "v2", shared.StatusActive, PolicyChangeContext{})
		require.NoError(t, err)
		assert.Equal(t, "new-name", result.Name)
	})
}

// ---------------------------------------------------------------------------
// PolicyService.SetStatusByUUID – transactional
// ---------------------------------------------------------------------------

func TestPolicyService_SetStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	policyUUID := uuid.New()

	t.Run("find error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("policy not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return nil, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, shared.StatusInactive)
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
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system policy")
	})

	t.Run("SetStatusByUUID repo error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "read-only", "v1")
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
			setStatusByUUIDFn: func(_ uuid.UUID, _ int64, _ string) error {
				return errors.New("status update failed")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status update failed")
	})

	t.Run("re-fetch error after status update → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "read-only", "v1")
		callCount := 0
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				callCount++
				if callCount == 1 {
					return p, nil
				}
				return nil, errors.New("re-fetch failed")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "re-fetch failed")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		p := newPolicy(tenantID, "read-only", "v1")
		callCount := 0
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				callCount++
				updated := *p
				updated.Status = shared.StatusInactive
				return &updated, nil
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		result, err := svc.SetStatusByUUID(context.Background(), policyUUID, tenantID, shared.StatusInactive)
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, result.Status)
	})
}

// ---------------------------------------------------------------------------
// PolicyService.DeleteByUUID – transactional
// ---------------------------------------------------------------------------

func TestPolicyService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	policyUUID := uuid.New()

	t.Run("find error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.DeleteByUUID(context.Background(), policyUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("policy not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return nil, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.DeleteByUUID(context.Background(), policyUUID, tenantID)
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
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.DeleteByUUID(context.Background(), policyUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("delete repo error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		p := newPolicy(tenantID, "read-only", "v1")
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
			deleteByUUIDAndTenantFn: func(_ uuid.UUID, _ int64) error {
				return errors.New("delete failed")
			},
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.DeleteByUUID(context.Background(), policyUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		p := newPolicy(tenantID, "read-only", "v1")
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*Policy, error) { return p, nil },
		}
		svc := NewPolicyService(db, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		result, err := svc.DeleteByUUID(context.Background(), policyUUID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, p.Name, result.Name)
	})
}

// "What did this policy look like" and "who changed it and why" are the same
// question for an IAM auditor. The columns existed and were never populated, so the
// history tab rendered a blank author — and the gRPC control plane, which writes no
// management_audit_log row either, left changes entirely unattributed.
func TestPolicyService_Update_RecordsChangeAttribution(t *testing.T) {
	policyUUID := uuid.New()
	tenantID := int64(1)
	actorUserID := int64(42)
	reason := "tightened billing scope after review"

	var captured *PolicyVersionHistory
	historyRepo := &mockHistoryRepo{
		createFn: func(h *PolicyVersionHistory) (*PolicyVersionHistory, error) {
			captured = h
			return h, nil
		},
		nextVersionFn: func(int64) (int, error) { return 2, nil },
	}

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := NewPolicyService(db, &mockPolicyRepo{
		findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) {
			return &Policy{
				PolicyID: 7, PolicyUUID: policyUUID, TenantID: tenantID,
				Name: "billing", Version: "v1",
				Document: datatypes.JSON(`{"version":"v1","statement":[]}`),
			}, nil
		},
	}, &mockServiceRepo{}, &mockAPIRepo{}, nil)
	SetPolicyVersionHistory(svc, historyRepo)

	_, err := svc.Update(context.Background(), policyUUID, tenantID, "billing", nil,
		datatypes.JSON(`{"version":"v1","statement":[]}`), "v1", "active",
		PolicyChangeContext{ActorUserID: &actorUserID, Reason: &reason})
	require.NoError(t, err)

	require.NotNil(t, captured, "an update must write a history row")
	require.NotNil(t, captured.ChangedByUserID)
	assert.Equal(t, actorUserID, *captured.ChangedByUserID)
	require.NotNil(t, captured.ChangeReason)
	assert.Equal(t, reason, *captured.ChangeReason)
	// The snapshot is the document BEFORE the change.
	assert.JSONEq(t, `{"version":"v1","statement":[]}`, string(captured.Document))
}

// ---------------------------------------------------------------------------
// PolicyService – MRN tenant boundary (Create/Update)
// ---------------------------------------------------------------------------

// stubTenantDirectory implements PolicyTenantDirectory for the boundary tests.
type stubTenantDirectory struct {
	system *tenant.TenantServiceDataResult
	byName map[string]*tenant.TenantServiceDataResult
}

func (d *stubTenantDirectory) GetSystem(context.Context) (*tenant.TenantServiceDataResult, error) {
	if d.system == nil {
		return nil, errNotFound
	}
	return d.system, nil
}

func (d *stubTenantDirectory) GetByName(_ context.Context, name string) (*tenant.TenantServiceDataResult, error) {
	if t, ok := d.byName[name]; ok {
		return t, nil
	}
	return nil, errNotFound
}

// The policy row is tenant-scoped but the evaluator matches on the resource
// STRING alone, so the write path is the only place that can stop tenant A
// granting itself mrn:storage:tenant-b:... — these tests pin that boundary.
func TestPolicyService_Create_MRNTenantBoundary(t *testing.T) {
	const (
		systemTenantID = int64(1)
		acmeTenantID   = int64(2)
		otherTenantID  = int64(3)
	)
	directory := &stubTenantDirectory{
		system: &tenant.TenantServiceDataResult{TenantID: systemTenantID, Name: "system", IsSystem: true},
		byName: map[string]*tenant.TenantServiceDataResult{
			"system":   {TenantID: systemTenantID, Name: "system", IsSystem: true},
			"acme":     {TenantID: acmeTenantID, Name: "acme"},
			"tenant-b": {TenantID: otherTenantID, Name: "tenant-b"},
		},
	}
	mrnDoc := func(resource string) datatypes.JSON {
		return datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["storage:read"],"resource":["` + resource + `"]}]}`)
	}

	newService := func(t *testing.T, expectWrite bool, dir PolicyTenantDirectory) PolicyService {
		t.Helper()
		db, mock := newMockGormDB(t)
		if expectWrite {
			mock.ExpectBegin()
			mock.ExpectCommit()
		}
		svc := NewPolicyService(db, &mockPolicyRepo{}, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		if dir != nil {
			SetPolicyTenantDirectory(svc, dir)
		}
		return svc
	}

	t.Run("regular tenant may reference its own tenant segment", func(t *testing.T) {
		svc := newService(t, true, directory)
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, mrnDoc("mrn:storage:acme:billing:bucket/*"), "v1", shared.StatusActive, false)
		require.NoError(t, err)
	})

	t.Run("regular tenant cross-tenant literal refused", func(t *testing.T) {
		svc := newService(t, false, directory)
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, mrnDoc("mrn:storage:tenant-b:billing:bucket/*"), "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not this policy's own tenant")
	})

	t.Run("regular tenant wildcard tenant segment refused", func(t *testing.T) {
		svc := newService(t, false, directory)
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, mrnDoc("mrn:storage:*:billing:bucket/*"), "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "wildcard tenant segment")
	})

	t.Run("regular tenant platform-scoped (empty tenant) refused", func(t *testing.T) {
		svc := newService(t, false, directory)
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, mrnDoc("mrn:core:::agent/*"), "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "platform-scoped")
	})

	t.Run("system tenant may use wildcard, empty, and other tenants' literals", func(t *testing.T) {
		for _, resource := range []string{
			"mrn:storage:*:billing:bucket/*",
			"mrn:core:::agent/*",
			"mrn:storage:tenant-b:billing:bucket/*",
		} {
			svc := newService(t, true, directory)
			_, err := svc.Create(context.Background(), systemTenantID, "p", nil, mrnDoc(resource), "v1", shared.StatusActive, false)
			require.NoError(t, err, "system tenant must be allowed %q", resource)
		}
	})

	t.Run("malformed MRN resource refused with the entry named", func(t *testing.T) {
		svc := newService(t, false, directory)
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, mrnDoc("mrn:storage:acme"), "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"mrn:storage:acme"`)
	})

	// Fail closed: an unverifiable tenant boundary is indistinguishable from a
	// crossed one, so no directory means no MRN-bearing policies — even for a
	// resource that names the owner's own tenant.
	t.Run("directory not wired → MRN resources refused", func(t *testing.T) {
		svc := newService(t, false, nil)
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, mrnDoc("mrn:storage:acme:billing:bucket/*"), "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be verified")
	})

	t.Run("system tenant unresolvable → MRN resources refused", func(t *testing.T) {
		svc := newService(t, false, &stubTenantDirectory{byName: directory.byName})
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, mrnDoc("mrn:storage:acme:billing:bucket/*"), "v1", shared.StatusActive, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system tenant could not be resolved")
	})

	// Legacy flat resources never touch the tenant directory at all — an
	// un-wired directory must not break the policies that exist today.
	t.Run("legacy flat resources untouched even without a directory", func(t *testing.T) {
		svc := newService(t, true, nil)
		doc := datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["user:*"],"resource":["auth:*","account:profile","*"]}]}`)
		_, err := svc.Create(context.Background(), acmeTenantID, "p", nil, doc, "v1", shared.StatusActive, false)
		require.NoError(t, err)
	})
}

// Update is just another write path for smuggling a cross-tenant MRN into an
// already-accepted policy, so it enforces the same boundary.
func TestPolicyService_Update_MRNTenantBoundary(t *testing.T) {
	const acmeTenantID = int64(2)
	directory := &stubTenantDirectory{
		system: &tenant.TenantServiceDataResult{TenantID: 1, Name: "system", IsSystem: true},
		byName: map[string]*tenant.TenantServiceDataResult{
			"acme": {TenantID: acmeTenantID, Name: "acme"},
		},
	}

	db, _ := newMockGormDB(t)
	svc := NewPolicyService(db, &mockPolicyRepo{
		findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) {
			t.Fatal("the boundary must be enforced before the policy is even loaded")
			return nil, nil
		},
	}, &mockServiceRepo{}, &mockAPIRepo{}, nil)
	SetPolicyTenantDirectory(svc, directory)

	doc := datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["storage:read"],"resource":["mrn:storage:tenant-b:billing:bucket/*"]}]}`)
	_, err := svc.Update(context.Background(), uuid.New(), acmeTenantID, "p", nil, doc, "v1", shared.StatusActive, PolicyChangeContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not this policy's own tenant")
}
