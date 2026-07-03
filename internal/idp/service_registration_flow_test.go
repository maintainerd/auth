package idp

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func buildRegistrationFlow() *RegistrationFlow {
	var clientID int64 = 1
	return &RegistrationFlow{
		RegistrationFlowUUID: uuid.New(),
		TenantID:             1,
		Name:                 "test-flow",
		Description:          "desc",
		Identifier:           "abc123",
		Status:               shared.StatusActive,
		ClientID:             clientID,
		Client:               &Client{ClientUUID: uuid.New()},
	}
}

func defaultCR() *mockClientRepo {
	return &mockClientRepo{
		findSystemFn:                        func() (*Client, error) { return nil, nil },
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return nil, nil },
	}
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_GetByUUID(t *testing.T) {
	sf := buildRegistrationFlow()

	t.Run("not found (nil)", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetByUUID(context.Background(), sf.RegistrationFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) {
				return nil, errors.New("db error")
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetByUUID(context.Background(), sf.RegistrationFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetByUUID(context.Background(), sf.RegistrationFlowUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, sf.Name, res.Name)
	})
}

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_GetAll(t *testing.T) {
	sf := buildRegistrationFlow()

	t.Run("success without ClientUUID", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findPaginatedFn: func(_ RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
				return &PaginationResult[RegistrationFlow]{Data: []RegistrationFlow{*sf}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetAll(context.Background(), 1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
	})

	t.Run("with ClientUUID → client not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		cUUID := uuid.New()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, nil }
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.GetAll(context.Background(), 1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("with ClientUUID → client repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		cUUID := uuid.New()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db") }
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.GetAll(context.Background(), 1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("with ClientUUID → success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		cUUID := uuid.New()
		clientID := int64(5)
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: clientID, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findPaginatedFn: func(f RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
				assert.NotNil(t, f.ClientID)
				assert.Equal(t, clientID, *f.ClientID)
				return &PaginationResult[RegistrationFlow]{Data: []RegistrationFlow{*sf}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.GetAll(context.Background(), 1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("FindPaginated error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findPaginatedFn: func(_ RegistrationFlowRepositoryGetFilter) (*PaginationResult[RegistrationFlow], error) {
				return nil, errors.New("paginate error")
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetAll(context.Background(), 1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate error")
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_Create(t *testing.T) {
	sf := buildRegistrationFlow()
	clientUUID := uuid.New()

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, nil }
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("FindByName error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return nil, errors.New("name err") },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name already exists", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return &RegistrationFlow{Name: "test-flow"}, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name already exists")
	})

	t.Run("GenerateIdentifier failure", func(t *testing.T) {
		orig := crypto.GenerateIdentifier
		defer func() { crypto.GenerateIdentifier = orig }()
		crypto.GenerateIdentifier = func(int) (string, error) { return "", errors.New("rand failure") }

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure")
	})

	t.Run("FindByIdentifierAndClientID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return nil, nil },
			findByIdentifierAndClientIDFn: func(_ string, _ int64) (*RegistrationFlow, error) {
				return nil, errors.New("ident err")
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ident err")
	})

	t.Run("config marshal error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.Error(t, err)
	})

	t.Run("Create repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return nil, nil },
			createFn:                func(_ *RegistrationFlow) (*RegistrationFlow, error) { return nil, errors.New("create err") },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("success with nil config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return nil, nil },
			createFn:                func(e *RegistrationFlow) (*RegistrationFlow, error) { return sf, nil },
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) {
				return sf, nil
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("success with config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) {
			return &Client{ClientID: 1, TenantID: 1, Status: shared.StatusActive}, nil
		}
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return nil, nil },
			createFn:                func(e *RegistrationFlow) (*RegistrationFlow, error) { return sf, nil },
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) {
				return sf, nil
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, clientUUID, nil, nil, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_Update(t *testing.T) {
	sf := buildRegistrationFlow()

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, "new", "desc", shared.StatusActive, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("name change → FindByName error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) { return nil, errors.New("name err") },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, "different-name", "desc", shared.StatusActive, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name change → conflict with different flow", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) {
				return &RegistrationFlow{RegistrationFlowID: 999, Name: "different-name"}, nil
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, "different-name", "desc", shared.StatusActive, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name already exists")
	})

	t.Run("name change → same flow found (no conflict)", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
			findByNameAndTenantIDFn: func(_ string, _ int64) (*RegistrationFlow, error) {
				return &RegistrationFlow{RegistrationFlowID: sf.RegistrationFlowID}, nil // same ID → no conflict
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, "different-name", "desc", shared.StatusActive, nil, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("same name (no change) → skip name check", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("config marshal error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, false, "[]")
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
			createOrUpdateFn: func(_ *RegistrationFlow) (*RegistrationFlow, error) {
				return nil, errors.New("update err")
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("success with config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(context.Background(), sf.RegistrationFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_UpdateStatus(t *testing.T) {
	sf := buildRegistrationFlow()

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.UpdateStatus(context.Background(), sf.RegistrationFlowUUID, 1, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
			createOrUpdateFn: func(_ *RegistrationFlow) (*RegistrationFlow, error) {
				return nil, errors.New("save err")
			},
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.UpdateStatus(context.Background(), sf.RegistrationFlowUUID, 1, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.UpdateStatus(context.Background(), sf.RegistrationFlowUUID, 1, shared.StatusInactive)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_Delete(t *testing.T) {
	sf := buildRegistrationFlow()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(context.Background(), sf.RegistrationFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("referenced by pending invites", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "invites"`).
			WithArgs(sf.RegistrationFlowID, "pending").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(context.Background(), sf.RegistrationFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pending invites")
	})

	t.Run("DeleteByUUID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "invites"`).
			WithArgs(sf.RegistrationFlowID, "pending").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
			deleteByUUIDFn:          func(_ any) error { return errors.New("delete err") },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(context.Background(), sf.RegistrationFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "invites"`).
			WithArgs(sf.RegistrationFlowID, "pending").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Delete(context.Background(), sf.RegistrationFlowUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, sf.Name, res.Name)
	})
}

// ---------------------------------------------------------------------------
// toRegistrationFlowServiceDataResult
// ---------------------------------------------------------------------------

func TestToRegistrationFlowServiceDataResult(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, toRegistrationFlowServiceDataResult(nil))
	})

	t.Run("invalid config JSON", func(t *testing.T) {
		sf := &RegistrationFlow{
			RegistrationFlowUUID: uuid.New(),
			Name:                 "bad-config",
		}
		res := toRegistrationFlowServiceDataResult(sf)
		require.NotNil(t, res)
		// Config field removed
	})

	t.Run("valid config JSON", func(t *testing.T) {
		sf := &RegistrationFlow{
			RegistrationFlowUUID: uuid.New(),
			Name:                 "good-config",
		}
		res := toRegistrationFlowServiceDataResult(sf)
		require.NotNil(t, res)
		// Config field removed
	})

	t.Run("empty config", func(t *testing.T) {
		sf := &RegistrationFlow{
			RegistrationFlowUUID: uuid.New(),
			Name:                 "empty-config",
		}
		res := toRegistrationFlowServiceDataResult(sf)
		require.NotNil(t, res)
		// Config field removed
	})

	t.Run("with client", func(t *testing.T) {
		cUUID := uuid.New()
		sf := &RegistrationFlow{
			RegistrationFlowUUID: uuid.New(),
			Client:               &Client{ClientUUID: cUUID},
		}
		res := toRegistrationFlowServiceDataResult(sf)
		assert.Equal(t, cUUID, res.ClientUUID)
	})

	t.Run("without client", func(t *testing.T) {
		sf := &RegistrationFlow{
			RegistrationFlowUUID: uuid.New(),
			Client:               nil,
		}
		res := toRegistrationFlowServiceDataResult(sf)
		assert.Equal(t, uuid.Nil, res.ClientUUID)
	})
}

// ---------------------------------------------------------------------------
// AssignRoles
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_AssignRoles(t *testing.T) {
	sf := buildRegistrationFlow()
	role := &Role{RoleID: 10, RoleUUID: uuid.New(), TenantID: 1, Name: "editor", Status: shared.StatusActive}

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.RegistrationFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("role not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return nil, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.RegistrationFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByRegistrationFlowIDAndRoleID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			findByRegistrationFlowIDAndRoleIDFn: func(_, _ int64) (*RegistrationFlowRole, error) {
				return nil, errors.New("lookup err")
			},
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.RegistrationFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup err")
	})

	t.Run("role already assigned → skip", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			findByRegistrationFlowIDAndRoleIDFn: func(_, _ int64) (*RegistrationFlowRole, error) {
				return &RegistrationFlowRole{RegistrationFlowRoleID: 99}, nil // already exists
			},
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		res, err := svc.AssignRoles(context.Background(), sf.RegistrationFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Create registration flow role error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			createFn: func(_ *RegistrationFlowRole) (*RegistrationFlowRole, error) {
				return nil, errors.New("create sfr err")
			},
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.RegistrationFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create sfr err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		res, err := svc.AssignRoles(context.Background(), sf.RegistrationFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, role.Name, res[0].RoleName)
	})
}

// ---------------------------------------------------------------------------
// GetRoles
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_GetRoles(t *testing.T) {
	sf := buildRegistrationFlow()

	t.Run("flow not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetRoles(context.Background(), sf.RegistrationFlowUUID, 1, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("FindByRegistrationFlowIDPaginated error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			findByRegistrationFlowIDPaginatedFn: func(_ int64, _, _ int) ([]RegistrationFlowRole, int64, error) {
				return nil, 0, errors.New("paginate err")
			},
		}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetRoles(context.Background(), sf.RegistrationFlowUUID, 1, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate err")
	})

	t.Run("success with nil Role in result", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			findByRegistrationFlowIDPaginatedFn: func(_ int64, _, _ int) ([]RegistrationFlowRole, int64, error) {
				return []RegistrationFlowRole{{RegistrationFlowRoleUUID: uuid.New(), Role: nil}}, 1, nil
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.RegistrationFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
		// role is nil so RoleName should be zero value
		assert.Equal(t, "", res.Data[0].RoleName)
	})

	t.Run("success with Role populated", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		roleUUID := uuid.New()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			findByRegistrationFlowIDPaginatedFn: func(_ int64, _, _ int) ([]RegistrationFlowRole, int64, error) {
				return []RegistrationFlowRole{{
					RegistrationFlowRoleUUID: uuid.New(),
					Role:                     &Role{RoleUUID: roleUUID, Name: "viewer"},
				}}, 1, nil
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.RegistrationFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, "viewer", res.Data[0].RoleName)
		assert.Equal(t, 1, res.TotalPages)
	})

	t.Run("totalPages rounds up", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			findByRegistrationFlowIDPaginatedFn: func(_ int64, _, _ int) ([]RegistrationFlowRole, int64, error) {
				return nil, 11, nil // 11 items / 10 per page = 2 pages
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.RegistrationFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 2, res.TotalPages)
	})

	t.Run("exact page boundary", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			findByRegistrationFlowIDPaginatedFn: func(_ int64, _, _ int) ([]RegistrationFlowRole, int64, error) {
				return nil, 10, nil // 10 items / 10 per page = 1 page (no rounding)
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.RegistrationFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, res.TotalPages)
	})
}

// ---------------------------------------------------------------------------
// RemoveRole
// ---------------------------------------------------------------------------

func TestRegistrationFlowService_RemoveRole(t *testing.T) {
	sf := buildRegistrationFlow()
	roleUUID := uuid.New()

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return nil, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.RegistrationFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registration flow not found")
	})

	t.Run("role not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return nil, nil },
		}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.RegistrationFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("DeleteByRegistrationFlowIDAndRoleID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{
			deleteByRegistrationFlowIDAndRoleIDFn: func(_, _ int64) error { return errors.New("del err") },
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: 1}, nil
			},
		}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.RegistrationFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewRegistrationFlowService(db, &mockRegistrationFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*RegistrationFlow, error) { return sf, nil },
		}, &mockRegistrationFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: 1}, nil
			},
		}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.RegistrationFlowUUID, 1, roleUUID)
		require.NoError(t, err)
	})
}
