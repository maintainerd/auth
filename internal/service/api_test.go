package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAPI returns a minimal API fixture.
func newAPI(id int64, name string, tenantID int64) *model.API {
	return &model.API{
		APIID:    id,
		APIUUID:  uuid.New(),
		Name:     name,
		TenantID: tenantID,
		Status:   model.StatusActive,
	}
}

func newAPIService(apiRepo *mockAPIRepo, svcRepo *mockServiceRepo, tsRepo *mockTenantServiceRepo) APIService {
	return NewAPIService(nil, apiRepo, svcRepo, tsRepo)
}

// ---------------------------------------------------------------------------
// APIService.GetByUUID
// ---------------------------------------------------------------------------

func TestAPIService_GetByUUID(t *testing.T) {
	tenantID := int64(1)
	apiUUID := uuid.New()

	cases := []struct {
		name        string
		setupRepo   func(r *mockAPIRepo)
		expectError bool
		errContains string
	}{
		{
			name: "found → success",
			setupRepo: func(r *mockAPIRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.API, error) {
					return newAPI(1, "users-api", tenantID), nil
				}
			},
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockAPIRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.API, error) {
					return nil, nil
				}
			},
			expectError: true,
			errContains: "api not found",
		},
		{
			name: "repo error → propagated",
			setupRepo: func(r *mockAPIRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64) (*model.API, error) {
					return nil, errors.New("db error")
				}
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiRepo := &mockAPIRepo{}
			tc.setupRepo(apiRepo)
			svc := newAPIService(apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
			result, err := svc.GetByUUID(context.Background(), apiUUID, tenantID)
			if tc.expectError {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// APIService.Get
// ---------------------------------------------------------------------------

func TestAPIService_Get(t *testing.T) {
	t.Run("success – empty result", func(t *testing.T) {
		svc := newAPIService(&mockAPIRepo{}, &mockServiceRepo{}, &mockTenantServiceRepo{})
		result, err := svc.Get(context.Background(), APIServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})

	t.Run("repo error → propagated", func(t *testing.T) {
		apiRepo := &mockAPIRepo{
			findPaginatedFn: func(_ repository.APIRepositoryGetFilter) (*repository.PaginationResult[model.API], error) {
				return nil, errors.New("db error")
			},
		}
		svc := newAPIService(apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.Get(context.Background(), APIServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.Error(t, err)
	})

	t.Run("success with service preloaded", func(t *testing.T) {
		api := newAPI(1, "users-api", 1)
		api.Service = &model.Service{ServiceUUID: uuid.New(), Name: "users-svc"}
		apiRepo := &mockAPIRepo{
			findPaginatedFn: func(_ repository.APIRepositoryGetFilter) (*repository.PaginationResult[model.API], error) {
				return &repository.PaginationResult[model.API]{
					Data: []model.API{*api}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		svc := newAPIService(apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		result, err := svc.Get(context.Background(), APIServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.NotNil(t, result.Data[0].Service)
		assert.Equal(t, "users-svc", result.Data[0].Service.Name)
	})
}

// ---------------------------------------------------------------------------
// APIService.GetServiceIDByUUID
// ---------------------------------------------------------------------------

func TestAPIService_GetServiceIDByUUID(t *testing.T) {
	serviceUUID := uuid.New()

	t.Run("service found → returns ID", func(t *testing.T) {
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 42}, nil
			},
		}
		svc := newAPIService(&mockAPIRepo{}, svcRepo, &mockTenantServiceRepo{})
		id, err := svc.GetServiceIDByUUID(context.Background(), serviceUUID)
		require.NoError(t, err)
		assert.Equal(t, int64(42), id)
	})

	t.Run("service not found → error", func(t *testing.T) {
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}
		svc := newAPIService(&mockAPIRepo{}, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.GetServiceIDByUUID(context.Background(), serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})
}

// ---------------------------------------------------------------------------
// APIService.Create – transactional
// ---------------------------------------------------------------------------

func TestAPIService_Create(t *testing.T) {
	tenantID := int64(1)
	serviceUUID := uuid.New().String()

	t.Run("api already exists → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByNameFn: func(_ string, _ int64) (*model.API, error) {
				return newAPI(1, "users-api", tenantID), nil // already exists
			},
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.Create(context.Background(), tenantID, "users-api", "", "", "rest", model.StatusActive, false, serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("service not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}
		svc := NewAPIService(db, &mockAPIRepo{}, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.Create(context.Background(), tenantID, "users-api", "", "", "rest", model.StatusActive, false, serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 1}, nil
			},
		}
		createdAPI := newAPI(1, "users-api", tenantID)
		apiRepo := &mockAPIRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return createdAPI, nil },
		}
		svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
		result, err := svc.Create(context.Background(), tenantID, "users-api", "", "", "rest", model.StatusActive, false, serviceUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// ---------------------------------------------------------------------------
// APIService.DeleteByUUID
// ---------------------------------------------------------------------------

func TestAPIService_DeleteByUUID(t *testing.T) {
	tenantID := int64(1)
	apiUUID := uuid.New()

	cases := []struct {
		name        string
		apiRepo     *mockAPIRepo
		tsRepo      *mockTenantServiceRepo
		expectError bool
		errContains string
	}{
		{
			name: "repo error → propagated",
			apiRepo: &mockAPIRepo{
				findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
					return nil, errors.New("db error")
				},
			},
			tsRepo:      &mockTenantServiceRepo{},
			expectError: true,
			errContains: "db error",
		},
		{
			name: "not found → error",
			apiRepo: &mockAPIRepo{
				findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return nil, nil },
			},
			tsRepo:      &mockTenantServiceRepo{},
			expectError: true,
			errContains: "api not found",
		},
		{
			name: "system api → cannot delete",
			apiRepo: &mockAPIRepo{
				findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
					a := newAPI(1, "system-api", tenantID)
					a.IsSystem = true
					return a, nil
				},
			},
			tsRepo:      &mockTenantServiceRepo{},
			expectError: true,
			errContains: "system API",
		},
		{
			name: "success",
			apiRepo: &mockAPIRepo{
				findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
					return newAPI(1, "users-api", tenantID), nil
				},
			},
			tsRepo: &mockTenantServiceRepo{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newAPIService(tc.apiRepo, &mockServiceRepo{}, tc.tsRepo)
			result, err := svc.DeleteByUUID(context.Background(), apiUUID, tenantID)
			if tc.expectError {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// APIService.DeleteByUUID – service tenant validation
// ---------------------------------------------------------------------------

func TestAPIService_DeleteByUUID_ServiceTenantValidation(t *testing.T) {
	tenantID := int64(1)
	apiUUID := uuid.New()

	t.Run("service belongs to tenant → success", func(t *testing.T) {
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				a := newAPI(1, "my-api", tenantID)
				a.ServiceID = 10
				return a, nil
			},
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) {
				return &model.TenantService{}, nil
			},
		}
		svc := newAPIService(apiRepo, &mockServiceRepo{}, tsRepo)
		result, err := svc.DeleteByUUID(context.Background(), apiUUID, tenantID)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("service NOT linked to tenant → access denied", func(t *testing.T) {
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				a := newAPI(1, "my-api", tenantID)
				a.ServiceID = 10
				return a, nil
			},
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) {
				return nil, nil
			},
		}
		svc := newAPIService(apiRepo, &mockServiceRepo{}, tsRepo)
		_, err := svc.DeleteByUUID(context.Background(), apiUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("delete error → propagated", func(t *testing.T) {
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "my-api", tenantID), nil
			},
			deleteByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) error {
				return errors.New("delete failed")
			},
		}
		svc := newAPIService(apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.DeleteByUUID(context.Background(), apiUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("service ID > 0 tenant-service repo error → access denied", func(t *testing.T) {
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				a := newAPI(1, "my-api", tenantID)
				a.ServiceID = 10
				return a, nil
			},
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) {
				return nil, errors.New("ts lookup err")
			},
		}
		svc := newAPIService(apiRepo, &mockServiceRepo{}, tsRepo)
		_, err := svc.DeleteByUUID(context.Background(), apiUUID, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})
}

// ---------------------------------------------------------------------------
// APIService.Update – transactional
// ---------------------------------------------------------------------------

func TestAPIService_Update(t *testing.T) {
	tenantID := int64(1)
	apiUUID := uuid.New()
	serviceUUID := uuid.New().String()

	t.Run("api not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return nil, nil },
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("api repo error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
	})

	t.Run("service ID > 0 tenant-service not linked → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		api := newAPI(1, "old", tenantID)
		api.ServiceID = 10
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return api, nil },
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) { return nil, nil },
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, tsRepo)
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("system api → cannot update", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		api := newAPI(1, "sys", tenantID)
		api.IsSystem = true
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return api, nil },
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system API")
	})

	t.Run("invalid service UUID → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old", tenantID), nil
			},
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", "bad-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid service UUID")
	})

	t.Run("service not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old", tenantID), nil
			},
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}
		svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("new service NOT linked to tenant → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old", tenantID), nil
			},
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 5}, nil
			},
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) { return nil, nil },
		}
		svc := NewAPIService(db, apiRepo, svcRepo, tsRepo)
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found or access denied")
	})

	t.Run("name conflict → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old-name", tenantID), nil
			},
			findByNameFn: func(_ string, _ int64) (*model.API, error) {
				other := newAPI(2, "new-name", tenantID)
				return other, nil
			},
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 5}, nil
			},
		}
		svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "new-name", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("update error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old", tenantID), nil
			},
			createOrUpdateFn: func(_ *model.API) (*model.API, error) {
				return nil, errors.New("save failed")
			},
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 5}, nil
			},
		}
		svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "old", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save failed")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		api := newAPI(1, "old", tenantID)
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return api, nil },
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 5}, nil
			},
		}
		svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
		result, err := svc.Update(context.Background(), apiUUID, tenantID, "old", "New Display", "desc", "rest", "active", serviceUUID)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("service ID > 0 tenant-service repo error → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		api := newAPI(1, "old", tenantID)
		api.ServiceID = 10
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return api, nil },
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) {
				return nil, errors.New("ts lookup err")
			},
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, tsRepo)
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("service repo FindByUUID error → service not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old", tenantID), nil
			},
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return nil, errors.New("svc lookup err")
			},
		}
		svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found")
	})

	t.Run("new service tenant-service repo error → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old", tenantID), nil
			},
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 5}, nil
			},
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) {
				return nil, errors.New("ts lookup err")
			},
		}
		svc := NewAPIService(db, apiRepo, svcRepo, tsRepo)
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "n", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service not found or access denied")
	})

	t.Run("FindByName error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "old-name", tenantID), nil
			},
			findByNameFn: func(_ string, _ int64) (*model.API, error) {
				return nil, errors.New("name lookup err")
			},
		}
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) {
				return &model.Service{ServiceID: 5}, nil
			},
		}
		svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.Update(context.Background(), apiUUID, tenantID, "new-name", "d", "desc", "rest", "active", serviceUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name lookup err")
	})
}

// ---------------------------------------------------------------------------
// APIService.SetStatusByUUID – transactional
// ---------------------------------------------------------------------------

func TestAPIService_SetStatusByUUID(t *testing.T) {
	tenantID := int64(1)
	apiUUID := uuid.New()

	t.Run("api repo error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), apiUUID, tenantID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("api not found → error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return nil, nil },
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), apiUUID, tenantID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("service ID > 0 – tenant-service not linked → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		api := newAPI(1, "a", tenantID)
		api.ServiceID = 10
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return api, nil },
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) { return nil, nil },
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, tsRepo)
		_, err := svc.SetStatusByUUID(context.Background(), apiUUID, tenantID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("system api → cannot update status", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		api := newAPI(1, "sys", tenantID)
		api.IsSystem = true
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return api, nil },
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), apiUUID, tenantID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system API")
	})

	t.Run("save error → propagated", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "a", tenantID), nil
			},
			createOrUpdateFn: func(_ *model.API) (*model.API, error) { return nil, errors.New("save err") },
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		_, err := svc.SetStatusByUUID(context.Background(), apiUUID, tenantID, "inactive")
		require.Error(t, err)
	})

	t.Run("service ID > 0 – tenant-service repo error → access denied", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		api := newAPI(1, "a", tenantID)
		api.ServiceID = 10
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) { return api, nil },
		}
		tsRepo := &mockTenantServiceRepo{
			findByTenantAndServiceFn: func(_, _ int64) (*model.TenantService, error) {
				return nil, errors.New("ts lookup err")
			},
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, tsRepo)
		_, err := svc.SetStatusByUUID(context.Background(), apiUUID, tenantID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		apiRepo := &mockAPIRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.API, error) {
				return newAPI(1, "a", tenantID), nil
			},
		}
		svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
		result, err := svc.SetStatusByUUID(context.Background(), apiUUID, tenantID, "inactive")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// APIService.Create – additional edge cases
// ---------------------------------------------------------------------------

func TestAPIService_Create_FindByNameError(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	apiRepo := &mockAPIRepo{
		findByNameFn: func(_ string, _ int64) (*model.API, error) { return nil, errors.New("name lookup err") },
	}
	svc := NewAPIService(db, apiRepo, &mockServiceRepo{}, &mockTenantServiceRepo{})
	_, err := svc.Create(context.Background(), 1, "api", "", "", "rest", "active", false, uuid.New().String())
	require.Error(t, err)
}

func TestAPIService_Create_SaveError(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	apiRepo := &mockAPIRepo{
		createOrUpdateFn: func(_ *model.API) (*model.API, error) { return nil, errors.New("save err") },
	}
	svcRepo := &mockServiceRepo{
		findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return &model.Service{ServiceID: 1}, nil },
	}
	svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
	_, err := svc.Create(context.Background(), 1, "api", "", "", "rest", "active", false, uuid.New().String())
	require.Error(t, err)
}

func TestAPIService_Create_FetchAfterSaveError(t *testing.T) {
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()
	apiRepo := &mockAPIRepo{
		findByUUIDFn: func(_ any, _ ...string) (*model.API, error) { return nil, errors.New("fetch err") },
	}
	svcRepo := &mockServiceRepo{
		findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return &model.Service{ServiceID: 1}, nil },
	}
	svc := NewAPIService(db, apiRepo, svcRepo, &mockTenantServiceRepo{})
	_, err := svc.Create(context.Background(), 1, "api", "", "", "rest", "active", false, uuid.New().String())
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// toAPIServiceDataResult – nil input
// ---------------------------------------------------------------------------

func TestToAPIServiceDataResult_Nil(t *testing.T) {
	result := toAPIServiceDataResult(nil)
	assert.Nil(t, result)
}
