package iam

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// mockHistoryRepo is a function-field mock of PolicyVersionHistoryRepository.
type mockHistoryRepo struct {
	createFn        func(*PolicyVersionHistory) (*PolicyVersionHistory, error)
	nextVersionFn   func(int64) (int, error)
	findPaginatedFn func(int64, int, int) (*PaginationResult[PolicyVersionHistory], error)
	findByVersionFn func(int64, int) (*PolicyVersionHistory, error)
}

func (m *mockHistoryRepo) WithTx(_ *gorm.DB) PolicyVersionHistoryRepository { return m }
func (m *mockHistoryRepo) Create(e *PolicyVersionHistory) (*PolicyVersionHistory, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockHistoryRepo) NextVersionNumber(policyID int64) (int, error) {
	if m.nextVersionFn != nil {
		return m.nextVersionFn(policyID)
	}
	return 1, nil
}
func (m *mockHistoryRepo) FindByPolicyIDPaginated(policyID int64, page, limit int) (*PaginationResult[PolicyVersionHistory], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(policyID, page, limit)
	}
	return &PaginationResult[PolicyVersionHistory]{}, nil
}
func (m *mockHistoryRepo) FindByPolicyIDAndVersion(policyID int64, versionNumber int) (*PolicyVersionHistory, error) {
	if m.findByVersionFn != nil {
		return m.findByVersionFn(policyID, versionNumber)
	}
	return nil, nil
}

func TestPolicyService_GetHistory(t *testing.T) {
	t.Run("resolves policy, calls history repo, maps results", func(t *testing.T) {
		policyUUID := uuid.New()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(id uuid.UUID, tenantID int64) (*Policy, error) {
				assert.Equal(t, policyUUID, id)
				assert.Equal(t, int64(1), tenantID)
				return &Policy{PolicyID: 42}, nil
			},
		}
		historyRepo := &mockHistoryRepo{
			findPaginatedFn: func(policyID int64, page, limit int) (*PaginationResult[PolicyVersionHistory], error) {
				assert.Equal(t, int64(42), policyID)
				assert.Equal(t, 1, page)
				assert.Equal(t, 10, limit)
				return &PaginationResult[PolicyVersionHistory]{
					Data:       []PolicyVersionHistory{{Name: "v1-snapshot"}},
					Total:      1,
					Page:       1,
					Limit:      10,
					TotalPages: 1,
				}, nil
			},
		}
		svc := NewPolicyService(nil, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		SetPolicyVersionHistory(svc, historyRepo)
		res, err := svc.GetHistory(context.Background(), policyUUID, 1, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Equal(t, "v1-snapshot", res.Data[0].Name)
	})

	t.Run("returns error when historyRepo is nil", func(t *testing.T) {
		svc := NewPolicyService(nil, &mockPolicyRepo{}, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		_, err := svc.GetHistory(context.Background(), uuid.New(), 1, 1, 10)
		assert.Error(t, err)
	})

	t.Run("policy not found returns error", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) { return nil, nil }}
		svc := NewPolicyService(nil, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		SetPolicyVersionHistory(svc, &mockHistoryRepo{})
		_, err := svc.GetHistory(context.Background(), uuid.New(), 1, 1, 10)
		assert.Error(t, err)
	})
}

func TestPolicyService_GetHistoryVersion(t *testing.T) {
	t.Run("resolves policy, calls history repo", func(t *testing.T) {
		policyUUID := uuid.New()
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(id uuid.UUID, tenantID int64) (*Policy, error) {
				return &Policy{PolicyID: 42}, nil
			},
		}
		historyRepo := &mockHistoryRepo{
			findByVersionFn: func(policyID int64, versionNumber int) (*PolicyVersionHistory, error) {
				assert.Equal(t, int64(42), policyID)
				assert.Equal(t, 3, versionNumber)
				return &PolicyVersionHistory{Name: "v3-snapshot", VersionNumber: 3}, nil
			},
		}
		svc := NewPolicyService(nil, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		SetPolicyVersionHistory(svc, historyRepo)
		res, err := svc.GetHistoryVersion(context.Background(), policyUUID, 1, 3)
		assert.NoError(t, err)
		assert.Equal(t, 3, res.VersionNumber)
		assert.Equal(t, "v3-snapshot", res.Name)
	})

	t.Run("version not found returns error", func(t *testing.T) {
		policyRepo := &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(uuid.UUID, int64) (*Policy, error) { return &Policy{PolicyID: 42}, nil },
		}
		historyRepo := &mockHistoryRepo{findByVersionFn: func(int64, int) (*PolicyVersionHistory, error) { return nil, nil }}
		svc := NewPolicyService(nil, policyRepo, &mockServiceRepo{}, &mockAPIRepo{}, nil)
		SetPolicyVersionHistory(svc, historyRepo)
		_, err := svc.GetHistoryVersion(context.Background(), uuid.New(), 1, 99)
		assert.Error(t, err)
	})
}
