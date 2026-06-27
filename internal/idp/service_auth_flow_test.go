package idp

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

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func buildAuthFlow() *AuthFlow {
	var clientID int64 = 1
	return &AuthFlow{
		AuthFlowUUID: uuid.New(),
		TenantID:     1,
		Name:         "test-flow",
		Description:  "desc",
		Identifier:   "abc123",
		Status:       shared.StatusActive,
		ClientID:     &clientID,
		Client:       &Client{ClientUUID: uuid.New()},
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

func TestAuthFlowService_GetByUUID(t *testing.T) {
	sf := buildAuthFlow()

	t.Run("not found (nil)", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetByUUID(context.Background(), sf.AuthFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) {
				return nil, errors.New("db error")
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetByUUID(context.Background(), sf.AuthFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetByUUID(context.Background(), sf.AuthFlowUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, sf.Name, res.Name)
	})
}

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestAuthFlowService_GetAll(t *testing.T) {
	sf := buildAuthFlow()

	t.Run("success without ClientUUID", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findPaginatedFn: func(_ AuthFlowRepositoryGetFilter) (*PaginationResult[AuthFlow], error) {
				return &PaginationResult[AuthFlow]{Data: []AuthFlow{*sf}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
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
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.GetAll(context.Background(), 1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("with ClientUUID → client repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		cUUID := uuid.New()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, errors.New("db") }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
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
			return &Client{ClientID: clientID, TenantID: 1}, nil
		}
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findPaginatedFn: func(f AuthFlowRepositoryGetFilter) (*PaginationResult[AuthFlow], error) {
				assert.NotNil(t, f.ClientID)
				assert.Equal(t, clientID, *f.ClientID)
				return &PaginationResult[AuthFlow]{Data: []AuthFlow{*sf}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.GetAll(context.Background(), 1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("FindPaginated error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findPaginatedFn: func(_ AuthFlowRepositoryGetFilter) (*PaginationResult[AuthFlow], error) {
				return nil, errors.New("paginate error")
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetAll(context.Background(), 1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate error")
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAuthFlowService_Create(t *testing.T) {
	sf := buildAuthFlow()
	clientUUID := uuid.New()

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return nil, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("FindByName error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) { return nil, errors.New("name err") },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name already exists", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) { return &AuthFlow{Name: "test-flow"}, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
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
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rand failure")
	})

	t.Run("FindByIdentifierAndClientID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) { return nil, nil },
			findByIdentifierAndClientIDFn: func(_ string, _ int64) (*AuthFlow, error) {
				return nil, errors.New("ident err")
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ident err")
	})

	t.Run("config marshal error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
	})

	t.Run("Create repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) { return nil, nil },
			createFn:     func(_ *AuthFlow) (*AuthFlow, error) { return nil, errors.New("create err") },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("success with nil config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) { return nil, nil },
			createFn:     func(e *AuthFlow) (*AuthFlow, error) { return sf, nil },
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) {
				return sf, nil
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("success with config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*Client, error) { return &Client{ClientID: 1, TenantID: 1}, nil }
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) { return nil, nil },
			createFn:     func(e *AuthFlow) (*AuthFlow, error) { return sf, nil },
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) {
				return sf, nil
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.Create(context.Background(), 1, "test-flow", "desc", shared.StatusActive, shared.DestinationIdentity, clientUUID, nil, nil, nil, false, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestAuthFlowService_Update(t *testing.T) {
	sf := buildAuthFlow()

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, "new", "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("name change → FindByName error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
			findByNameAndTenantIDFn:            func(_ string, _ int64) (*AuthFlow, error) { return nil, errors.New("name err") },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, "different-name", "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name change → conflict with different flow", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) {
				return &AuthFlow{AuthFlowID: 999, Name: "different-name"}, nil
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, "different-name", "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name already exists")
	})

	t.Run("name change → same flow found (no conflict)", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
			findByNameAndTenantIDFn: func(_ string, _ int64) (*AuthFlow, error) {
				return &AuthFlow{AuthFlowID: sf.AuthFlowID}, nil // same ID → no conflict
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, "different-name", "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("same name (no change) → skip name check", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("config marshal error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
			createOrUpdateFn: func(_ *AuthFlow) (*AuthFlow, error) {
				return nil, errors.New("update err")
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("success with config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(context.Background(), sf.AuthFlowUUID, 1, sf.Name, "desc", shared.StatusActive, nil, nil, nil, false, false, "[]")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestAuthFlowService_UpdateStatus(t *testing.T) {
	sf := buildAuthFlow()

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.UpdateStatus(context.Background(), sf.AuthFlowUUID, 1, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
			createOrUpdateFn: func(_ *AuthFlow) (*AuthFlow, error) {
				return nil, errors.New("save err")
			},
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.UpdateStatus(context.Background(), sf.AuthFlowUUID, 1, shared.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.UpdateStatus(context.Background(), sf.AuthFlowUUID, 1, shared.StatusInactive)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAuthFlowService_Delete(t *testing.T) {
	sf := buildAuthFlow()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(context.Background(), sf.AuthFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth flow not found")
	})

	t.Run("referenced by pending invites", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "invites"`).
			WithArgs(sf.AuthFlowID, "pending").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(context.Background(), sf.AuthFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pending invites")
	})

	t.Run("DeleteByUUID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "invites"`).
			WithArgs(sf.AuthFlowID, "pending").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
			deleteByUUIDFn:          func(_ any) error { return errors.New("delete err") },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(context.Background(), sf.AuthFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "invites"`).
			WithArgs(sf.AuthFlowID, "pending").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Delete(context.Background(), sf.AuthFlowUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, sf.Name, res.Name)
	})
}

// ---------------------------------------------------------------------------
// toAuthFlowServiceDataResult
// ---------------------------------------------------------------------------

func TestToAuthFlowServiceDataResult(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, toAuthFlowServiceDataResult(nil))
	})

	t.Run("invalid config JSON", func(t *testing.T) {
		sf := &AuthFlow{
			AuthFlowUUID: uuid.New(),
			Name:         "bad-config",
		}
		res := toAuthFlowServiceDataResult(sf)
		require.NotNil(t, res)
		// Config field removed
	})

	t.Run("valid config JSON", func(t *testing.T) {
		sf := &AuthFlow{
			AuthFlowUUID: uuid.New(),
			Name:         "good-config",
		}
		res := toAuthFlowServiceDataResult(sf)
		require.NotNil(t, res)
		// Config field removed
	})

	t.Run("empty config", func(t *testing.T) {
		sf := &AuthFlow{
			AuthFlowUUID: uuid.New(),
			Name:         "empty-config",
		}
		res := toAuthFlowServiceDataResult(sf)
		require.NotNil(t, res)
		// Config field removed
	})

	t.Run("with client", func(t *testing.T) {
		cUUID := uuid.New()
		sf := &AuthFlow{
			AuthFlowUUID: uuid.New(),
			Client:       &Client{ClientUUID: cUUID},
		}
		res := toAuthFlowServiceDataResult(sf)
		assert.Equal(t, cUUID, res.ClientUUID)
	})

	t.Run("without client", func(t *testing.T) {
		sf := &AuthFlow{
			AuthFlowUUID: uuid.New(),
			Client:       nil,
		}
		res := toAuthFlowServiceDataResult(sf)
		assert.Equal(t, uuid.Nil, res.ClientUUID)
	})
}

// ---------------------------------------------------------------------------
// AssignRoles
// ---------------------------------------------------------------------------

func TestAuthFlowService_AssignRoles(t *testing.T) {
	sf := buildAuthFlow()
	role := &Role{RoleID: 10, RoleUUID: uuid.New(), TenantID: 1, Name: "editor", Status: shared.StatusActive}

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.AuthFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("role not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return nil, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.AuthFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindByAuthFlowIDAndRoleID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			findByAuthFlowIDAndRoleIDFn: func(_, _ int64) (*AuthFlowRole, error) {
				return nil, errors.New("lookup err")
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.AuthFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup err")
	})

	t.Run("role already assigned → skip", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			findByAuthFlowIDAndRoleIDFn: func(_, _ int64) (*AuthFlowRole, error) {
				return &AuthFlowRole{AuthFlowRoleID: 99}, nil // already exists
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		res, err := svc.AssignRoles(context.Background(), sf.AuthFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Create signup flow role error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			createFn: func(_ *AuthFlowRole) (*AuthFlowRole, error) {
				return nil, errors.New("create sfr err")
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(context.Background(), sf.AuthFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create sfr err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return role, nil },
		}, defaultCR())
		res, err := svc.AssignRoles(context.Background(), sf.AuthFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, role.Name, res[0].RoleName)
	})
}

// ---------------------------------------------------------------------------
// GetRoles
// ---------------------------------------------------------------------------

func TestAuthFlowService_GetRoles(t *testing.T) {
	sf := buildAuthFlow()

	t.Run("flow not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetRoles(context.Background(), sf.AuthFlowUUID, 1, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("FindByAuthFlowIDPaginated error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			findByAuthFlowIDPaginatedFn: func(_ int64, _, _ int) ([]AuthFlowRole, int64, error) {
				return nil, 0, errors.New("paginate err")
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetRoles(context.Background(), sf.AuthFlowUUID, 1, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate err")
	})

	t.Run("success with nil Role in result", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			findByAuthFlowIDPaginatedFn: func(_ int64, _, _ int) ([]AuthFlowRole, int64, error) {
				return []AuthFlowRole{{AuthFlowRoleUUID: uuid.New(), Role: nil}}, 1, nil
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.AuthFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
		// role is nil so RoleName should be zero value
		assert.Equal(t, "", res.Data[0].RoleName)
	})

	t.Run("success with Role populated", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		roleUUID := uuid.New()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			findByAuthFlowIDPaginatedFn: func(_ int64, _, _ int) ([]AuthFlowRole, int64, error) {
				return []AuthFlowRole{{
					AuthFlowRoleUUID: uuid.New(),
					Role:             &Role{RoleUUID: roleUUID, Name: "viewer"},
				}}, 1, nil
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.AuthFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, "viewer", res.Data[0].RoleName)
		assert.Equal(t, 1, res.TotalPages)
	})

	t.Run("totalPages rounds up", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			findByAuthFlowIDPaginatedFn: func(_ int64, _, _ int) ([]AuthFlowRole, int64, error) {
				return nil, 11, nil // 11 items / 10 per page = 2 pages
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.AuthFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 2, res.TotalPages)
	})

	t.Run("exact page boundary", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			findByAuthFlowIDPaginatedFn: func(_ int64, _, _ int) ([]AuthFlowRole, int64, error) {
				return nil, 10, nil // 10 items / 10 per page = 1 page (no rounding)
			},
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(context.Background(), sf.AuthFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, res.TotalPages)
	})
}

// ---------------------------------------------------------------------------
// RemoveRole
// ---------------------------------------------------------------------------

func TestAuthFlowService_RemoveRole(t *testing.T) {
	sf := buildAuthFlow()
	roleUUID := uuid.New()

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return nil, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.AuthFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("role not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return nil, nil },
		}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.AuthFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("DeleteByAuthFlowIDAndRoleID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{
			deleteByAuthFlowIDAndRoleIDFn: func(_, _ int64) error { return errors.New("del err") },
		}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: 1}, nil
			},
		}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.AuthFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewAuthFlowService(db, &mockAuthFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*AuthFlow, error) { return sf, nil },
		}, &mockAuthFlowRoleRepo{}, &mockAuthFlowCallbackURIRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) {
				return &Role{RoleID: 10, RoleUUID: roleUUID, TenantID: 1}, nil
			},
		}, defaultCR())
		err := svc.RemoveRole(context.Background(), sf.AuthFlowUUID, 1, roleUUID)
		require.NoError(t, err)
	})
}
