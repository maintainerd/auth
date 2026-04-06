package service

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
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

// ---------------------------------------------------------------------------
// TenantService.SetStatusByUUID – additional branches
// ---------------------------------------------------------------------------

func TestTenantService_SetStatusByUUID_Extra(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("SetStatusByUUID error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return newTenant(1, "acme"), nil
			},
			setStatusByUUIDFn: func(_ uuid.UUID, _ string) error {
				return errors.New("set status err")
			},
		}
		svc := NewTenantService(nil, repo)
		_, err := svc.SetStatusByUUID(tenantUUID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set status err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				calls++
				if calls == 1 {
					return newTenant(1, "acme"), nil
				}
				return nil, errors.New("fetch err")
			},
		}
		svc := NewTenantService(nil, repo)
		_, err := svc.SetStatusByUUID(tenantUUID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})
}

// ---------------------------------------------------------------------------
// TenantService.Get – success with data
// ---------------------------------------------------------------------------

func TestTenantService_Get_WithData(t *testing.T) {
	repo := &mockTenantRepo{
		findPaginatedFn: func(_ repository.TenantRepositoryGetFilter) (*repository.PaginationResult[model.Tenant], error) {
			return &repository.PaginationResult[model.Tenant]{
				Data:       []model.Tenant{*newTenant(1, "acme"), *newTenant(2, "beta")},
				Total:      2,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	svc := NewTenantService(nil, repo)
	res, err := svc.Get(TenantServiceGetFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, res.Data, 2)
	assert.Equal(t, int64(2), res.Total)
}

// ---------------------------------------------------------------------------
// TenantService.Create
// ---------------------------------------------------------------------------

func TestTenantService_Create(t *testing.T) {
	t.Run("FindByName error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByNameFn: func(_ string) (*model.Tenant, error) { return nil, errors.New("name err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Create("acme", "Acme Corp", "desc", "active", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("tenant already exists", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByNameFn: func(_ string) (*model.Tenant, error) { return newTenant(1, "acme"), nil },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Create("acme", "Acme Corp", "desc", "active", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		repo := &mockTenantRepo{
			createOrUpdateFn: func(_ *model.Tenant) (*model.Tenant, error) { return nil, errors.New("create err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Create("acme", "Acme Corp", "desc", "active", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, errors.New("fetch err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Create("acme", "Acme Corp", "desc", "active", true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return newTenant(1, "acme"), nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantService(db, repo)
		res, err := svc.Create("acme", "Acme Corp", "desc", "active", true, false)
		require.NoError(t, err)
		assert.Equal(t, "acme", res.Name)
	})
}

// ---------------------------------------------------------------------------
// TenantService.Update
// ---------------------------------------------------------------------------

func TestTenantService_Update(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("FindByUUID error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, errors.New("find err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Update(tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find err")
	})

	t.Run("tenant not found", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Update(tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("name conflict FindByName error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*model.Tenant, error) { return nil, errors.New("name err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Update(tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name already exists", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*model.Tenant, error) { return newTenant(999, "new"), nil },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Update(tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
			createOrUpdateFn: func(_ *model.Tenant) (*model.Tenant, error) { return nil, errors.New("save err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.Update(tenantUUID, "old", "New", "desc", "active", true) // same name → no conflict check
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success — same name", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme"}, nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantService(db, repo)
		res, err := svc.Update(tenantUUID, "acme", "Acme Corp", "desc", "active", true)
		require.NoError(t, err)
		assert.Equal(t, "Acme Corp", res.DisplayName)
	})

	t.Run("success — different name, no conflict", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantService(db, repo)
		res, err := svc.Update(tenantUUID, "new-name", "New Name", "desc", "active", false)
		require.NoError(t, err)
		assert.Equal(t, "new-name", res.Name)
	})
}

// ---------------------------------------------------------------------------
// TenantService.SetActivePublicByUUID
// ---------------------------------------------------------------------------

func TestTenantService_SetActivePublicByUUID(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("tenant not found", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil },
		}
		svc := NewTenantService(nil, repo)
		_, err := svc.SetActivePublicByUUID(tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("db update error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return newTenant(1, "acme"), nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE .tenants.`).
			WillReturnError(errors.New("update err"))
		mock.ExpectRollback()
		svc := NewTenantService(db, repo)
		_, err := svc.SetActivePublicByUUID(tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				calls++
				if calls == 1 {
					return newTenant(1, "acme"), nil
				}
				return nil, errors.New("fetch err")
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE .tenants.`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		svc := NewTenantService(db, repo)
		_, err := svc.SetActivePublicByUUID(tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				calls++
				if calls == 1 {
					return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme", IsPublic: false}, nil
				}
				return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme", IsPublic: true}, nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE .tenants.`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		svc := NewTenantService(db, repo)
		res, err := svc.SetActivePublicByUUID(tenantUUID)
		require.NoError(t, err)
		assert.True(t, res.IsPublic)
	})
}

// ---------------------------------------------------------------------------
// TenantService.SetDefaultStatusByUUID
// ---------------------------------------------------------------------------

func TestTenantService_SetDefaultStatusByUUID(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("tenant not found", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) { return nil, nil },
		}
		svc := NewTenantService(nil, repo)
		_, err := svc.SetDefaultStatusByUUID(tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("SetDefaultStatusByUUID error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				return newTenant(1, "acme"), nil
			},
			setDefaultStatusByUUIDFn: func(_ uuid.UUID, _ bool) error {
				return errors.New("default err")
			},
		}
		svc := NewTenantService(nil, repo)
		_, err := svc.SetDefaultStatusByUUID(tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				calls++
				if calls == 1 {
					return newTenant(1, "acme"), nil
				}
				return nil, errors.New("fetch err")
			},
		}
		svc := NewTenantService(nil, repo)
		_, err := svc.SetDefaultStatusByUUID(tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Tenant, error) {
				calls++
				if calls == 1 {
					return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme", IsDefault: false}, nil
				}
				return &model.Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme", IsDefault: true}, nil
			},
		}
		svc := NewTenantService(nil, repo)
		res, err := svc.SetDefaultStatusByUUID(tenantUUID)
		require.NoError(t, err)
		assert.True(t, res.IsDefault)
	})
}

// ---------------------------------------------------------------------------
// TenantService.DeleteByUUID – delete error
// ---------------------------------------------------------------------------

func TestTenantService_DeleteByUUID_DeleteError(t *testing.T) {
	tenantUUID := uuid.New()
	repo := &mockTenantRepo{
		findByUUIDFn:   func(_ any, _ ...string) (*model.Tenant, error) { return newTenant(1, "acme"), nil },
		deleteByUUIDFn: func(_ any) error { return errors.New("delete err") },
	}
	svc := NewTenantService(nil, repo)
	_, err := svc.DeleteByUUID(tenantUUID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete err")
}
