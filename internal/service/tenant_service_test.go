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

// newTenant returns a minimal Tenant fixture for tests.
func newTenant(id int64, name string) *model.Tenant {
	return &model.Tenant{
		TenantID:   id,
		TenantUUID: uuid.New(),
		Name:       name,
		Status:     model.StatusActive,
	}
}

// ---------------------------------------------------------------------------
// TenantService.GetByUUID
// ---------------------------------------------------------------------------

func TestTenantService_GetByUUID(t *testing.T) {
	cases := []struct {
		name        string
		setupRepo   func(r *mockTenantRepo)
		expectError bool
	}{
		{
			name: "found → success",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					return newTenant(1, "acme"), nil
				}
			},
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					return nil, nil
				}
			},
			expectError: true,
		},
		{
			name: "repo error → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					return nil, errors.New("db error")
				}
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(nil, repo)
			result, err := svc.GetByUUID(uuid.New())
			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TenantService.GetDefault
// ---------------------------------------------------------------------------

func TestTenantService_GetDefault(t *testing.T) {
	cases := []struct {
		name        string
		setupRepo   func(r *mockTenantRepo)
		expectError bool
	}{
		{
			name: "found → success",
			setupRepo: func(r *mockTenantRepo) {
				r.findDefaultFn = func() (*model.Tenant, error) {
					t := newTenant(1, "default")
					t.IsDefault = true
					return t, nil
				}
			},
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findDefaultFn = func() (*model.Tenant, error) { return nil, nil }
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(nil, repo)
			result, err := svc.GetDefault()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TenantService.GetByIdentifier
// ---------------------------------------------------------------------------

func TestTenantService_GetByIdentifier(t *testing.T) {
	cases := []struct {
		name        string
		identifier  string
		setupRepo   func(r *mockTenantRepo)
		expectError bool
	}{
		{
			name:       "found → success",
			identifier: "acme-corp",
			setupRepo: func(r *mockTenantRepo) {
				r.findByIdentifierFn = func(id string) (*model.Tenant, error) {
					return newTenant(1, "Acme"), nil
				}
			},
		},
		{
			name:       "not found → error",
			identifier: "unknown",
			setupRepo: func(r *mockTenantRepo) {
				r.findByIdentifierFn = func(id string) (*model.Tenant, error) { return nil, nil }
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(nil, repo)
			result, err := svc.GetByIdentifier(tc.identifier)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TenantService.Get (paginated)
// ---------------------------------------------------------------------------

func TestTenantService_Get(t *testing.T) {
	t.Run("success – empty result", func(t *testing.T) {
		repo := &mockTenantRepo{}
		svc := NewTenantService(nil, repo)
		result, err := svc.Get(TenantServiceGetFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})

	t.Run("repo error – propagated", func(t *testing.T) {
		repo := &mockTenantRepo{
			findPaginatedFn: func(_ repository.TenantRepositoryGetFilter) (*repository.PaginationResult[model.Tenant], error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewTenantService(nil, repo)
		result, err := svc.Get(TenantServiceGetFilter{Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TenantService.DeleteByUUID
// ---------------------------------------------------------------------------

func TestTenantService_DeleteByUUID(t *testing.T) {
	tenantUUID := uuid.New()

	cases := []struct {
		name        string
		setupRepo   func(r *mockTenantRepo)
		expectError bool
		errContains string
	}{
		{
			name: "not found → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil }
			},
			expectError: true,
			errContains: "not found",
		},
		{
			name: "system tenant → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					t := newTenant(1, "system")
					t.IsSystem = true
					return t, nil
				}
			},
			expectError: true,
			errContains: "system tenant",
		},
		{
			name: "default tenant → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					t := newTenant(1, "default")
					t.IsDefault = true
					return t, nil
				}
			},
			expectError: true,
			errContains: "default tenant",
		},
		{
			name: "success",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					return newTenant(1, "acme"), nil
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(nil, repo)
			result, err := svc.DeleteByUUID(tenantUUID)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TenantService.SetStatusByUUID
// ---------------------------------------------------------------------------

func TestTenantService_SetStatusByUUID(t *testing.T) {
	tenantUUID := uuid.New()

	cases := []struct {
		name        string
		setupRepo   func(r *mockTenantRepo)
		expectError bool
	}{
		{
			name: "tenant not found → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil }
			},
			expectError: true,
		},
		{
			name: "repo error → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					return nil, errors.New("db error")
				}
			},
			expectError: true,
		},
		{
			name: "success",
			setupRepo: func(r *mockTenantRepo) {
				tenant := newTenant(1, "acme")
				calls := 0
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Tenant, error) {
					calls++
					return tenant, nil
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(nil, repo)
			result, err := svc.SetStatusByUUID(tenantUUID, model.StatusActive)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
