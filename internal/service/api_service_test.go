package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
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
			result, err := svc.GetByUUID(apiUUID, tenantID)
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
		result, err := svc.Get(APIServiceGetFilter{TenantID: 1, Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
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
		id, err := svc.GetServiceIDByUUID(serviceUUID)
		require.NoError(t, err)
		assert.Equal(t, int64(42), id)
	})

	t.Run("service not found → error", func(t *testing.T) {
		svcRepo := &mockServiceRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Service, error) { return nil, nil },
		}
		svc := newAPIService(&mockAPIRepo{}, svcRepo, &mockTenantServiceRepo{})
		_, err := svc.GetServiceIDByUUID(serviceUUID)
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
		_, err := svc.Create(tenantID, "users-api", "", "", "rest", model.StatusActive, false, serviceUUID)
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
		_, err := svc.Create(tenantID, "users-api", "", "", "rest", model.StatusActive, false, serviceUUID)
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
		result, err := svc.Create(tenantID, "users-api", "", "", "rest", model.StatusActive, false, serviceUUID)
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
			result, err := svc.DeleteByUUID(apiUUID, tenantID)
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
