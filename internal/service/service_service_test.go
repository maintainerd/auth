package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServiceSvc(t *testing.T, serviceRepo *mockServiceRepo, tsRepo *mockTenantServiceRepo, apiRepo *mockAPIRepo, spRepo *mockServicePolicyRepo, policyRepo *mockPolicyRepo) ServiceService {
	db, _ := newMockGormDB(t)
	return NewServiceService(db, serviceRepo, tsRepo, apiRepo, spRepo, policyRepo)
}

func TestServiceService_GetByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("service not found", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.GetByUUID(id, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("access denied - not in tenant", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id}, nil
			},
		}, &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_ int64, _ int64) (*model.TenantService, error) { return nil, nil },
		}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.GetByUUID(id, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("success", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, Name: "auth"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.GetByUUID(id, tid)
		require.NoError(t, err)
		assert.Equal(t, "auth", res.Name)
	})
}

func TestServiceService_Create(t *testing.T) {
	t.Run("already exists → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Service, error) {
				return &model.Service{Name: "auth"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Create("auth", "Auth", "desc", "v1", false, "active", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.Create("payments", "Payments", "desc", "v1", false, "active", 1)
		require.NoError(t, err)
		assert.Equal(t, "payments", res.Name)
	})
}

func TestServiceService_SetStatusByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("service not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.SetStatusByUUID(id, tid, "inactive")
		require.Error(t, err)
	})

	t.Run("system service blocked → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, IsSystem: true}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.SetStatusByUUID(id, tid, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, Status: "active"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.SetStatusByUUID(id, tid, "inactive")
		require.NoError(t, err)
		assert.Equal(t, "inactive", res.Status)
	})
}

func TestServiceService_DeleteByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("not found", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.DeleteByUUID(id, tid)
		require.Error(t, err)
	})

	t.Run("system service blocked", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, IsSystem: true}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.DeleteByUUID(id, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("success", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, Name: "svc"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.DeleteByUUID(id, tid)
		require.NoError(t, err)
		assert.Equal(t, id, res.ServiceUUID)
	})
}

