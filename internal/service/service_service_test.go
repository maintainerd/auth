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

func newServiceSvc(t *testing.T, serviceRepo *mockServiceRepo, tsRepo *mockTenantServiceRepo, apiRepo *mockAPIRepo, spRepo *mockServicePolicyRepo, policyRepo *mockPolicyRepo) ServiceService {
	db, _ := newMockGormDB(t)
	return NewServiceService(db, serviceRepo, tsRepo, apiRepo, spRepo, policyRepo)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestServiceService_Get(t *testing.T) {
	t.Run("FindPaginated error", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findPaginatedFn: func(_ repository.ServiceRepositoryGetFilter) (*repository.PaginationResult[model.Service], error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Get(ServiceServiceGetFilter{Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("success without tenantID", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findPaginatedFn: func(_ repository.ServiceRepositoryGetFilter) (*repository.PaginationResult[model.Service], error) {
				return &repository.PaginationResult[model.Service]{
					Data:  []model.Service{{ServiceUUID: uuid.New(), Name: "svc1"}},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.Get(ServiceServiceGetFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Equal(t, "svc1", res.Data[0].Name)
	})

	t.Run("success with tenantID", func(t *testing.T) {
		tid := int64(5)
		svc := newServiceSvc(t, &mockServiceRepo{
			findPaginatedFn: func(_ repository.ServiceRepositoryGetFilter) (*repository.PaginationResult[model.Service], error) {
				return &repository.PaginationResult[model.Service]{
					Data:  []model.Service{{ServiceUUID: uuid.New(), Name: "svc2"}},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.Get(ServiceServiceGetFilter{TenantID: &tid, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("empty results", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.Get(ServiceServiceGetFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Empty(t, res.Data)
	})
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

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

	t.Run("FindByUUID error", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return nil, errors.New("db error")
			},
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

	t.Run("FindByTenantAndService error", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id}, nil
			},
		}, &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_ int64, _ int64) (*model.TenantService, error) {
				return nil, errors.New("db error")
			},
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

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestServiceService_Create(t *testing.T) {
	t.Run("FindByNameAndTenantID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*model.Service, error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Create("auth", "Auth", "desc", "v1", false, "active", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

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

	t.Run("CreateOrUpdate service error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			createOrUpdateFn: func(_ *model.Service) (*model.Service, error) {
				return nil, errors.New("create failed")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Create("payments", "Payments", "desc", "v1", false, "active", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("CreateOrUpdate tenant-service error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{}, &mockTenantServiceRepo{
			createOrUpdateFn: func(_ *model.TenantService) (*model.TenantService, error) {
				return nil, errors.New("tenant link failed")
			},
		}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Create("payments", "Payments", "desc", "v1", false, "active", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant link failed")
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

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestServiceService_Update(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("service not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("FindByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("access denied → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id}, nil
			},
		}, &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_ int64, _ int64) (*model.TenantService, error) { return nil, nil },
		}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("system service cannot be updated → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id, IsSystem: true, Name: "core"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system service cannot be updated")
	})

	t.Run("name change → FindByName error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*model.Service, error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("name change → duplicate name different UUID → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		otherUUID := uuid.New()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*model.Service, error) {
				return &model.Service{ServiceUUID: otherUUID, Name: "new"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("name change → same UUID → allowed", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, Name: "new"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.NoError(t, err)
		assert.Equal(t, "new", res.Name)
	})

	t.Run("name change → FindByName nil → allowed", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*model.Service, error) {
				return nil, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.Update(id, tid, "new", "New", "d", "v1", false, "active")
		require.NoError(t, err)
		assert.Equal(t, "new", res.Name)
	})

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id, Name: "svc"}, nil
			},
			createOrUpdateFn: func(_ *model.Service) (*model.Service, error) {
				return nil, errors.New("save failed")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.Update(id, tid, "svc", "Svc", "d", "v2", false, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save failed")
	})

	t.Run("success same name → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id, Name: "svc"}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		res, err := svc.Update(id, tid, "svc", "Service", "updated desc", "v2", false, "active")
		require.NoError(t, err)
		assert.Equal(t, "Service", res.DisplayName)
		assert.Equal(t, "v2", res.Version)
	})
}

// ---------------------------------------------------------------------------
// SetStatusByUUID
// ---------------------------------------------------------------------------

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

	t.Run("access denied → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id}, nil
			},
		}, &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_ int64, _ int64) (*model.TenantService, error) { return nil, nil },
		}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.SetStatusByUUID(id, tid, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
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

	t.Run("CreateOrUpdate error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, Status: "active"}, nil
			},
			createOrUpdateFn: func(_ *model.Service) (*model.Service, error) {
				return nil, errors.New("save failed")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.SetStatusByUUID(id, tid, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save failed")
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

// ---------------------------------------------------------------------------
// DeleteByUUID
// ---------------------------------------------------------------------------

func TestServiceService_DeleteByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("not found", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.DeleteByUUID(id, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("access denied", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: id}, nil
			},
		}, &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_ int64, _ int64) (*model.TenantService, error) { return nil, nil },
		}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.DeleteByUUID(id, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
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

	t.Run("DeleteByUUID repo error", func(t *testing.T) {
		svc := newServiceSvc(t, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceUUID: id, Name: "svc"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		_, err := svc.DeleteByUUID(id, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
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

// ---------------------------------------------------------------------------
// AssignPolicy
// ---------------------------------------------------------------------------

func TestServiceService_AssignPolicy(t *testing.T) {
	svcUUID := uuid.New()
	polUUID := uuid.New()
	tid := int64(1)

	t.Run("FindByUUID service error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("service not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("FindByUUIDAndTenantID policy error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return nil, errors.New("db error")
			},
		})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("policy not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) { return nil, nil },
		})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "policy not found")
	})

	t.Run("FindByServiceAndPolicy error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{
			findByServiceAndPolicyFn: func(_ int64, _ int64) (*model.ServicePolicy, error) {
				return nil, errors.New("db error")
			},
		}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("already assigned → idempotent success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{
			findByServiceAndPolicyFn: func(_ int64, _ int64) (*model.ServicePolicy, error) {
				return &model.ServicePolicy{ServicePolicyUUID: uuid.New()}, nil
			},
		}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.NoError(t, err)
	})

	t.Run("Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{
			createFn: func(_ *model.ServicePolicy) (*model.ServicePolicy, error) {
				return nil, errors.New("create failed")
			},
		}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.AssignPolicy(svcUUID, polUUID, tid)
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// RemovePolicy
// ---------------------------------------------------------------------------

func TestServiceService_RemovePolicy(t *testing.T) {
	svcUUID := uuid.New()
	polUUID := uuid.New()
	tid := int64(1)

	t.Run("FindByUUID service error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return nil, errors.New("db error")
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("service not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("FindByUUIDAndTenantID policy error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return nil, errors.New("db error")
			},
		})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("policy not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) { return nil, nil },
		})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "policy not found")
	})

	t.Run("FindByServiceAndPolicy error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{
			findByServiceAndPolicyFn: func(_ int64, _ int64) (*model.ServicePolicy, error) {
				return nil, errors.New("db error")
			},
		}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("not assigned → idempotent success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.NoError(t, err)
	})

	t.Run("DeleteByServiceAndPolicy error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{
			findByServiceAndPolicyFn: func(_ int64, _ int64) (*model.ServicePolicy, error) {
				return &model.ServicePolicy{ServicePolicyUUID: uuid.New()}, nil
			},
			deleteByServiceAndPolicy: func(_ int64, _ int64) error {
				return errors.New("delete failed")
			},
		}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewServiceService(db, &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1, ServiceUUID: svcUUID}, nil
			},
		}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{
			findByServiceAndPolicyFn: func(_ int64, _ int64) (*model.ServicePolicy, error) {
				return &model.ServicePolicy{ServicePolicyUUID: uuid.New()}, nil
			},
		}, &mockPolicyRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Policy, error) {
				return &model.Policy{PolicyID: 2, PolicyUUID: polUUID}, nil
			},
		})
		err := svc.RemovePolicy(svcUUID, polUUID, tid)
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// toServiceServiceDataResult (indirectly tested, but verify nil-safe)
// ---------------------------------------------------------------------------

func TestServiceService_toServiceServiceDataResult(t *testing.T) {
	svc := newServiceSvc(t, &mockServiceRepo{}, &mockTenantServiceRepo{}, &mockAPIRepo{}, &mockServicePolicyRepo{}, &mockPolicyRepo{})
	// Access private method via exported method (GetByUUID success path)
	ss := svc.(*serviceService)

	t.Run("maps all fields", func(t *testing.T) {
		s := &model.Service{
			ServiceUUID: uuid.New(),
			Name:        "auth",
			DisplayName: "Auth Service",
			Description: "Authentication",
			Version:     "v1",
			IsSystem:    true,
			Status:      "active",
		}
		res := ss.toServiceServiceDataResult(s, 1)
		assert.Equal(t, s.ServiceUUID, res.ServiceUUID)
		assert.Equal(t, "auth", res.Name)
		assert.Equal(t, "Auth Service", res.DisplayName)
		assert.Equal(t, "Authentication", res.Description)
		assert.Equal(t, "v1", res.Version)
		assert.True(t, res.IsSystem)
		assert.Equal(t, "active", res.Status)
	})
}
