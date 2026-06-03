package tenant

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTenant returns a minimal Tenant fixture for tests.
func newTenant(id int64, name string) *Tenant {
	return &Tenant{
		TenantID:   id,
		TenantUUID: uuid.New(),
		Name:       name,
		Status:     shared.StatusActive,
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
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) {
					return newTenant(1, "acme"), nil
				}
			},
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) {
					return nil, nil
				}
			},
			expectError: true,
		},
		{
			name: "repo error → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) {
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
			svc := NewTenantService(repo, nil)
			result, err := svc.GetByUUID(context.Background(), uuid.New())
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
// TenantService.GetSystem
// ---------------------------------------------------------------------------

func TestTenantService_GetSystem(t *testing.T) {
	cases := []struct {
		name        string
		setupRepo   func(r *mockTenantRepo)
		expectError bool
	}{
		{
			name: "found → success",
			setupRepo: func(r *mockTenantRepo) {
				r.findSystemFn = func() (*Tenant, error) {
					t := newTenant(1, "system")
					t.IsSystem = true
					return t, nil
				}
			},
		},
		{
			name: "not found → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findSystemFn = func() (*Tenant, error) { return nil, nil }
			},
			expectError: true,
		},
		{
			name: "repo error → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findSystemFn = func() (*Tenant, error) { return nil, errors.New("db error") }
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(repo, nil)
			result, err := svc.GetSystem(context.Background())
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
				r.findByIdentifierFn = func(id string) (*Tenant, error) {
					return newTenant(1, "Acme"), nil
				}
			},
		},
		{
			name:       "not found → error",
			identifier: "unknown",
			setupRepo: func(r *mockTenantRepo) {
				r.findByIdentifierFn = func(id string) (*Tenant, error) { return nil, nil }
			},
			expectError: true,
		},
		{
			name:       "repo error → error",
			identifier: "acme-corp",
			setupRepo: func(r *mockTenantRepo) {
				r.findByIdentifierFn = func(id string) (*Tenant, error) { return nil, errors.New("db error") }
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(repo, nil)
			result, err := svc.GetByIdentifier(context.Background(), tc.identifier)
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
		svc := NewTenantService(repo, nil)
		result, err := svc.Get(context.Background(), TenantServiceGetFilter{Page: 1, Limit: 10})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Data)
	})

	t.Run("repo error – propagated", func(t *testing.T) {
		repo := &mockTenantRepo{
			findPaginatedFn: func(_ TenantRepositoryGetFilter) (*PaginationResult[Tenant], error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewTenantService(repo, nil)
		result, err := svc.Get(context.Background(), TenantServiceGetFilter{Page: 1, Limit: 10})
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// ---------------------------------------------------------------------------
// TenantService.DeleteByUUID
// ---------------------------------------------------------------------------

func TestTenantService_DeleteByUUID(t *testing.T) {
	tenantUUID := uuid.New()

	// cascadeCount is the number of SQL statements the cascade loop executes
	// (30 models, each generating one UPDATE or DELETE via GORM).
	const cascadeCount = 30

	cases := []struct {
		name        string
		setupRepo   func(r *mockTenantRepo)
		setupSQL    func(mock sqlmock.Sqlmock)
		expectError bool
		errContains string
	}{
		{
			name: "not found → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return nil, nil }
			},
			setupSQL: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			expectError: true,
			errContains: "not found",
		},
		{
			name: "system tenant → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) {
					t := newTenant(1, "system")
					t.IsSystem = true
					return t, nil
				}
			},
			setupSQL: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			expectError: true,
			errContains: "system tenant",
		},
		{
			name: "success",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) {
					return newTenant(1, "acme"), nil
				}
			},
			setupSQL: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				for i := 0; i < cascadeCount; i++ {
					mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				}
				mock.ExpectCommit()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			tc.setupSQL(mock)
			repo := &mockTenantRepo{}
			tc.setupRepo(repo)
			svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
			result, err := svc.DeleteByUUID(context.Background(), tenantUUID)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
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
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) { return nil, nil }
			},
			expectError: true,
		},
		{
			name: "repo error → error",
			setupRepo: func(r *mockTenantRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) {
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
				r.findByUUIDFn = func(_ any, _ ...string) (*Tenant, error) {
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
			svc := NewTenantService(repo, nil)
			result, err := svc.SetStatusByUUID(context.Background(), tenantUUID, shared.StatusActive)
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
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return newTenant(1, "acme"), nil
			},
			setStatusByUUIDFn: func(_ uuid.UUID, _ string) error {
				return errors.New("set status err")
			},
		}
		svc := NewTenantService(repo, nil)
		_, err := svc.SetStatusByUUID(context.Background(), tenantUUID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set status err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				calls++
				if calls == 1 {
					return newTenant(1, "acme"), nil
				}
				return nil, errors.New("fetch err")
			},
		}
		svc := NewTenantService(repo, nil)
		_, err := svc.SetStatusByUUID(context.Background(), tenantUUID, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})
}

// ---------------------------------------------------------------------------
// TenantService.Get – success with data
// ---------------------------------------------------------------------------

func TestTenantService_Get_WithData(t *testing.T) {
	repo := &mockTenantRepo{
		findPaginatedFn: func(_ TenantRepositoryGetFilter) (*PaginationResult[Tenant], error) {
			return &PaginationResult[Tenant]{
				Data:       []Tenant{*newTenant(1, "acme"), *newTenant(2, "beta")},
				Total:      2,
				Page:       1,
				Limit:      10,
				TotalPages: 1,
			}, nil
		},
	}
	svc := NewTenantService(repo, nil)
	res, err := svc.Get(context.Background(), TenantServiceGetFilter{Page: 1, Limit: 10})
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
			findByNameFn: func(_ string) (*Tenant, error) { return nil, errors.New("name err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Create(context.Background(), "acme", "Acme Corp", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("tenant already exists", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByNameFn: func(_ string) (*Tenant, error) { return newTenant(1, "acme"), nil },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Create(context.Background(), "acme", "Acme Corp", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GenerateIdentifier failure", func(t *testing.T) {
		orig := crypto.GenerateIdentifier
		defer func() { crypto.GenerateIdentifier = orig }()
		crypto.GenerateIdentifier = func(int) (string, error) { return "", errors.New("rand failure") }

		repo := &mockTenantRepo{}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Create(context.Background(), "acme", "Acme Corp", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		repo := &mockTenantRepo{
			createOrUpdateFn: func(_ *Tenant) (*Tenant, error) { return nil, errors.New("create err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Create(context.Background(), "acme", "Acme Corp", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, errors.New("fetch err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Create(context.Background(), "acme", "Acme Corp", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return newTenant(1, "acme"), nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		res, err := svc.Create(context.Background(), "acme", "Acme Corp", "desc", "active", true)
		require.NoError(t, err)
		assert.Equal(t, "acme", res.Name)
	})
}

// ---------------------------------------------------------------------------
// TenantService.Update
// ---------------------------------------------------------------------------

func TestTenantService_Update(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("authevent.FindByUUID error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, errors.New("find err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Update(context.Background(), tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find err")
	})

	t.Run("tenant not found", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, nil },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Update(context.Background(), tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("initial fetch error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, errors.New("fetch err") },
		}
		svc := NewTenantService(repo, nil)
		_, err := svc.SetActivePublicByUUID(context.Background(), tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("name conflict FindByName error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*Tenant, error) { return nil, errors.New("name err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Update(context.Background(), tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name already exists", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
			findByNameFn: func(_ string) (*Tenant, error) { return newTenant(999, "new"), nil },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Update(context.Background(), tenantUUID, "new", "New", "desc", "active", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
			createOrUpdateFn: func(_ *Tenant) (*Tenant, error) { return nil, errors.New("save err") },
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.Update(context.Background(), tenantUUID, "old", "New", "desc", "active", true) // same name → no conflict check
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success — same name", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme"}, nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		res, err := svc.Update(context.Background(), tenantUUID, "acme", "Acme Corp", "desc", "active", true)
		require.NoError(t, err)
		assert.Equal(t, "Acme Corp", res.DisplayName)
	})

	t.Run("success — different name, no conflict", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return &Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "old"}, nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		res, err := svc.Update(context.Background(), tenantUUID, "new-name", "New Name", "desc", "active", false)
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
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) { return nil, nil },
		}
		svc := NewTenantService(repo, nil)
		_, err := svc.SetActivePublicByUUID(context.Background(), tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant not found")
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return newTenant(1, "acme"), nil
			},
			createOrUpdateFn: func(_ *Tenant) (*Tenant, error) {
				return nil, errors.New("update err")
			},
		}
		svc := NewTenantService(repo, nil)
		_, err := svc.SetActivePublicByUUID(context.Background(), tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("final fetch error", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
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
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		_, err := svc.SetActivePublicByUUID(context.Background(), tenantUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
	})

	t.Run("success", func(t *testing.T) {
		calls := 0
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				calls++
				if calls == 1 {
					return &Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme", IsPublic: false}, nil
				}
				return &Tenant{TenantID: 1, TenantUUID: tenantUUID, Name: "acme", IsPublic: true}, nil
			},
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE .tenants.`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
		res, err := svc.SetActivePublicByUUID(context.Background(), tenantUUID)
		require.NoError(t, err)
		assert.True(t, res.IsPublic)
	})
}

// ---------------------------------------------------------------------------
// TenantService.DeleteByUUID – delete error
// ---------------------------------------------------------------------------

func TestTenantService_DeleteByUUID_DeleteError(t *testing.T) {
	tenantUUID := uuid.New()
	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	for i := 0; i < 30; i++ {
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectRollback()

	repo := &mockTenantRepo{
		findByUUIDFn:   func(_ any, _ ...string) (*Tenant, error) { return newTenant(1, "acme"), nil },
		deleteByUUIDFn: func(_ any) error { return errors.New("delete err") },
	}
	svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))
	_, err := svc.DeleteByUUID(context.Background(), tenantUUID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete err")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantService_DeleteByUUID_AdditionalErrors(t *testing.T) {
	tenantUUID := uuid.New()

	t.Run("initial fetch error rolls back", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return nil, errors.New("fetch err")
			},
		}
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))

		_, err := svc.DeleteByUUID(context.Background(), tenantUUID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cascade error rolls back", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(".*").WillReturnError(errors.New("cascade err"))
		mock.ExpectRollback()
		repo := &mockTenantRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
				return newTenant(1, "acme"), nil
			},
		}
		svc := NewTenantService(repo, NewGormUnitOfWork(db, repo, nil, testCascadeModels()))

		_, err := svc.DeleteByUUID(context.Background(), tenantUUID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cascade err")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestValidateTenantAccess(t *testing.T) {
	cases := []struct {
		name        string
		user        AccessActor
		target      *Tenant
		expectError bool
		errContains string
	}{
		{
			name:        "nil actor → error",
			user:        nil,
			target:      buildTenant(10, false),
			expectError: true,
			errContains: "actor user is nil",
		},
		{
			name:        "nil target → error",
			user:        buildUserWithIdentities([]AccessIdentity{buildIdentity(10, false)}),
			target:      nil,
			expectError: true,
			errContains: "target tenant is nil",
		},
		{
			name:        "no identities → error",
			user:        buildUserWithIdentities(nil),
			target:      buildTenant(10, false),
			expectError: true,
			errContains: "no identities",
		},
		{
			name: "user from default tenant → allowed on any tenant",
			user: buildUserWithIdentities([]AccessIdentity{
				buildIdentity(1, true),
			}),
			target:      buildTenant(99, false),
			expectError: false,
		},
		{
			name: "user from same tenant → allowed",
			user: buildUserWithIdentities([]AccessIdentity{
				buildIdentity(10, false),
			}),
			target:      buildTenant(10, false),
			expectError: false,
		},
		{
			name: "user from different non-default tenant → denied",
			user: buildUserWithIdentities([]AccessIdentity{
				buildIdentity(20, false),
			}),
			target:      buildTenant(10, false),
			expectError: true,
			errContains: "access denied",
		},
		{
			name: "multiple identities; one matches target → allowed",
			user: buildUserWithIdentities([]AccessIdentity{
				buildIdentity(20, false),
				buildIdentity(10, false),
			}),
			target:      buildTenant(10, false),
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTenantAccess(tc.user, tc.target)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTenantAccessByID(t *testing.T) {
	cases := []struct {
		name           string
		user           AccessActor
		targetTenantID int64
		expectError    bool
		errContains    string
	}{
		{
			name:           "nil actor → error",
			user:           nil,
			targetTenantID: 10,
			expectError:    true,
			errContains:    "actor user is nil",
		},
		{
			name:           "no identities → error",
			user:           buildUserWithIdentities(nil),
			targetTenantID: 10,
			expectError:    true,
			errContains:    "no identities",
		},
		{
			name: "default tenant user → allowed on any tenant",
			user: buildUserWithIdentities([]AccessIdentity{
				buildIdentity(1, true),
			}),
			targetTenantID: 99,
			expectError:    false,
		},
		{
			name: "matching tenant ID → allowed",
			user: buildUserWithIdentities([]AccessIdentity{
				buildIdentity(10, false),
			}),
			targetTenantID: 10,
			expectError:    false,
		},
		{
			name: "non-matching non-default → denied",
			user: buildUserWithIdentities([]AccessIdentity{
				buildIdentity(20, false),
			}),
			targetTenantID: 10,
			expectError:    true,
			errContains:    "access denied",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTenantAccessByID(tc.user, tc.targetTenantID)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
