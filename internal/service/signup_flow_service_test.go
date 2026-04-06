package service

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func buildSignupFlow() *model.SignupFlow {
	return &model.SignupFlow{
		SignupFlowID:   1,
		SignupFlowUUID: uuid.New(),
		TenantID:       1,
		Name:           "test-flow",
		Description:    "desc",
		Identifier:     "abc123",
		Status:         model.StatusActive,
		ClientID:       1,
		Client:         &model.Client{ClientUUID: uuid.New()},
	}
}

func defaultCR() *mockClientRepo {
	return &mockClientRepo{
		findDefaultFn:                       func() (*model.Client, error) { return nil, nil },
		findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) { return nil, nil },
	}
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestSignupFlowService_GetByUUID(t *testing.T) {
	sf := buildSignupFlow()

	t.Run("not found (nil)", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetByUUID(sf.SignupFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
				return nil, errors.New("db error")
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetByUUID(sf.SignupFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetByUUID(sf.SignupFlowUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, sf.Name, res.Name)
	})
}

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestSignupFlowService_GetAll(t *testing.T) {
	sf := buildSignupFlow()

	t.Run("success without ClientUUID", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findPaginatedFn: func(_ repository.SignupFlowRepositoryGetFilter) (*repository.PaginationResult[model.SignupFlow], error) {
				return &repository.PaginationResult[model.SignupFlow]{Data: []model.SignupFlow{*sf}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetAll(1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
	})

	t.Run("with ClientUUID → client not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		cUUID := uuid.New()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return nil, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.GetAll(1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("with ClientUUID → client repo error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		cUUID := uuid.New()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return nil, errors.New("db") }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.GetAll(1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("with ClientUUID → success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		cUUID := uuid.New()
		clientID := int64(5)
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) {
			return &model.Client{ClientID: clientID}, nil
		}
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findPaginatedFn: func(f repository.SignupFlowRepositoryGetFilter) (*repository.PaginationResult[model.SignupFlow], error) {
				assert.NotNil(t, f.ClientID)
				assert.Equal(t, clientID, *f.ClientID)
				return &repository.PaginationResult[model.SignupFlow]{Data: []model.SignupFlow{*sf}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.GetAll(1, nil, nil, nil, &cUUID, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("FindPaginated error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findPaginatedFn: func(_ repository.SignupFlowRepositoryGetFilter) (*repository.PaginationResult[model.SignupFlow], error) {
				return nil, errors.New("paginate error")
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetAll(1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate error")
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestSignupFlowService_Create(t *testing.T) {
	sf := buildSignupFlow()
	clientUUID := uuid.New()

	t.Run("client not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return nil, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(1, "test-flow", "desc", nil, model.StatusActive, clientUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth client not found")
	})

	t.Run("FindByName error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByNameFn: func(_ string) (*model.SignupFlow, error) { return nil, errors.New("name err") },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(1, "test-flow", "desc", nil, model.StatusActive, clientUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name already exists", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByNameFn: func(_ string) (*model.SignupFlow, error) { return &model.SignupFlow{Name: "test-flow"}, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(1, "test-flow", "desc", nil, model.StatusActive, clientUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name already exists")
	})

	t.Run("FindByIdentifierAndClientID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByNameFn: func(_ string) (*model.SignupFlow, error) { return nil, nil },
			findByIdentifierAndClientIDFn: func(_ string, _ int64) (*model.SignupFlow, error) {
				return nil, errors.New("ident err")
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(1, "test-flow", "desc", nil, model.StatusActive, clientUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ident err")
	})

	t.Run("config marshal error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByNameFn: func(_ string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(1, "test-flow", "desc", map[string]any{"bad": math.Inf(1)}, model.StatusActive, clientUUID)
		require.Error(t, err)
	})

	t.Run("Create repo error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByNameFn: func(_ string) (*model.SignupFlow, error) { return nil, nil },
			createFn:     func(_ *model.SignupFlow) (*model.SignupFlow, error) { return nil, errors.New("create err") },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		_, err := svc.Create(1, "test-flow", "desc", nil, model.StatusActive, clientUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create err")
	})

	t.Run("success with nil config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByNameFn: func(_ string) (*model.SignupFlow, error) { return nil, nil },
			createFn:     func(e *model.SignupFlow) (*model.SignupFlow, error) { return sf, nil },
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
				return sf, nil
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.Create(1, "test-flow", "desc", nil, model.StatusActive, clientUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("success with config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		cr := defaultCR()
		cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return &model.Client{ClientID: 1}, nil }
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByNameFn: func(_ string) (*model.SignupFlow, error) { return nil, nil },
			createFn:     func(e *model.SignupFlow) (*model.SignupFlow, error) { return sf, nil },
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
				return sf, nil
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
		res, err := svc.Create(1, "test-flow", "desc", map[string]any{"key": "val"}, model.StatusActive, clientUUID)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestSignupFlowService_Update(t *testing.T) {
	sf := buildSignupFlow()

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(sf.SignupFlowUUID, 1, "new", "desc", nil, model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("name change → FindByName error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
			findByNameFn:            func(_ string) (*model.SignupFlow, error) { return nil, errors.New("name err") },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(sf.SignupFlowUUID, 1, "different-name", "desc", nil, model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name err")
	})

	t.Run("name change → conflict with different flow", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
			findByNameFn: func(_ string) (*model.SignupFlow, error) {
				return &model.SignupFlow{SignupFlowID: 999, Name: "different-name"}, nil
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(sf.SignupFlowUUID, 1, "different-name", "desc", nil, model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name already exists")
	})

	t.Run("name change → same flow found (no conflict)", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
			findByNameFn: func(_ string) (*model.SignupFlow, error) {
				return &model.SignupFlow{SignupFlowID: sf.SignupFlowID}, nil // same ID → no conflict
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(sf.SignupFlowUUID, 1, "different-name", "desc", nil, model.StatusActive)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("same name (no change) → skip name check", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(sf.SignupFlowUUID, 1, sf.Name, "desc", nil, model.StatusActive)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("config marshal error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(sf.SignupFlowUUID, 1, sf.Name, "desc", map[string]any{"bad": math.Inf(1)}, model.StatusActive)
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
			createOrUpdateFn: func(_ *model.SignupFlow) (*model.SignupFlow, error) {
				return nil, errors.New("update err")
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Update(sf.SignupFlowUUID, 1, sf.Name, "desc", nil, model.StatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update err")
	})

	t.Run("success with config", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Update(sf.SignupFlowUUID, 1, sf.Name, "desc", map[string]any{"k": "v"}, model.StatusActive)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestSignupFlowService_UpdateStatus(t *testing.T) {
	sf := buildSignupFlow()

	t.Run("not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.UpdateStatus(sf.SignupFlowUUID, 1, model.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
			createOrUpdateFn: func(_ *model.SignupFlow) (*model.SignupFlow, error) {
				return nil, errors.New("save err")
			},
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.UpdateStatus(sf.SignupFlowUUID, 1, model.StatusInactive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.UpdateStatus(sf.SignupFlowUUID, 1, model.StatusInactive)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestSignupFlowService_Delete(t *testing.T) {
	sf := buildSignupFlow()

	t.Run("not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(sf.SignupFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("DeleteByUUID error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
			deleteByUUIDFn:          func(_ any) error { return errors.New("delete err") },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.Delete(sf.SignupFlowUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete err")
	})

	t.Run("success", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		res, err := svc.Delete(sf.SignupFlowUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, sf.Name, res.Name)
	})
}

// ---------------------------------------------------------------------------
// toSignupFlowServiceDataResult
// ---------------------------------------------------------------------------

func TestToSignupFlowServiceDataResult(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, toSignupFlowServiceDataResult(nil))
	})

	t.Run("invalid config JSON", func(t *testing.T) {
		sf := &model.SignupFlow{
			SignupFlowUUID: uuid.New(),
			Name:           "bad-config",
			Config:         []byte("not-json"),
		}
		res := toSignupFlowServiceDataResult(sf)
		require.NotNil(t, res)
		assert.Nil(t, res.Config)
	})

	t.Run("valid config JSON", func(t *testing.T) {
		sf := &model.SignupFlow{
			SignupFlowUUID: uuid.New(),
			Name:           "good-config",
			Config:         []byte(`{"key":"val"}`),
		}
		res := toSignupFlowServiceDataResult(sf)
		require.NotNil(t, res)
		assert.Equal(t, "val", res.Config["key"])
	})

	t.Run("empty config", func(t *testing.T) {
		sf := &model.SignupFlow{
			SignupFlowUUID: uuid.New(),
			Name:           "empty-config",
			Config:         []byte{},
		}
		res := toSignupFlowServiceDataResult(sf)
		require.NotNil(t, res)
		assert.Nil(t, res.Config)
	})

	t.Run("with client", func(t *testing.T) {
		cUUID := uuid.New()
		sf := &model.SignupFlow{
			SignupFlowUUID: uuid.New(),
			Client:         &model.Client{ClientUUID: cUUID},
			Config:         []byte(`{}`),
		}
		res := toSignupFlowServiceDataResult(sf)
		assert.Equal(t, cUUID, res.ClientUUID)
	})

	t.Run("without client", func(t *testing.T) {
		sf := &model.SignupFlow{
			SignupFlowUUID: uuid.New(),
			Client:         nil,
			Config:         []byte(`{}`),
		}
		res := toSignupFlowServiceDataResult(sf)
		assert.Equal(t, uuid.Nil, res.ClientUUID)
	})
}

// ---------------------------------------------------------------------------
// AssignRoles
// ---------------------------------------------------------------------------

func TestSignupFlowService_AssignRoles(t *testing.T) {
	sf := buildSignupFlow()
	role := &model.Role{RoleID: 10, RoleUUID: uuid.New(), Name: "editor", Status: model.StatusActive}

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.AssignRoles(sf.SignupFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("role not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(sf.SignupFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("FindBySignupFlowIDAndRoleID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			findBySignupFlowIDAndRoleIDFn: func(_, _ int64) (*model.SignupFlowRole, error) {
				return nil, errors.New("lookup err")
			},
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return role, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(sf.SignupFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lookup err")
	})

	t.Run("role already assigned → skip", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			findBySignupFlowIDAndRoleIDFn: func(_, _ int64) (*model.SignupFlowRole, error) {
				return &model.SignupFlowRole{SignupFlowRoleID: 99}, nil // already exists
			},
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return role, nil },
		}, defaultCR())
		res, err := svc.AssignRoles(sf.SignupFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Create signup flow role error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			createFn: func(_ *model.SignupFlowRole) (*model.SignupFlowRole, error) {
				return nil, errors.New("create sfr err")
			},
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return role, nil },
		}, defaultCR())
		_, err := svc.AssignRoles(sf.SignupFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create sfr err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return role, nil },
		}, defaultCR())
		res, err := svc.AssignRoles(sf.SignupFlowUUID, 1, []uuid.UUID{role.RoleUUID})
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, role.Name, res[0].RoleName)
	})
}

// ---------------------------------------------------------------------------
// GetRoles
// ---------------------------------------------------------------------------

func TestSignupFlowService_GetRoles(t *testing.T) {
	sf := buildSignupFlow()

	t.Run("flow not found", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetRoles(sf.SignupFlowUUID, 1, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("FindBySignupFlowIDPaginated error", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			findBySignupFlowIDPaginatedFn: func(_ int64, _, _ int) ([]model.SignupFlowRole, int64, error) {
				return nil, 0, errors.New("paginate err")
			},
		}, &mockRoleRepo{}, defaultCR())
		_, err := svc.GetRoles(sf.SignupFlowUUID, 1, 1, 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate err")
	})

	t.Run("success with nil Role in result", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			findBySignupFlowIDPaginatedFn: func(_ int64, _, _ int) ([]model.SignupFlowRole, int64, error) {
				return []model.SignupFlowRole{{SignupFlowRoleUUID: uuid.New(), Role: nil}}, 1, nil
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(sf.SignupFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
		// role is nil so RoleName should be zero value
		assert.Equal(t, "", res.Data[0].RoleName)
	})

	t.Run("success with Role populated", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		roleUUID := uuid.New()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			findBySignupFlowIDPaginatedFn: func(_ int64, _, _ int) ([]model.SignupFlowRole, int64, error) {
				return []model.SignupFlowRole{{
					SignupFlowRoleUUID: uuid.New(),
					Role:               &model.Role{RoleUUID: roleUUID, Name: "viewer"},
				}}, 1, nil
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(sf.SignupFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, "viewer", res.Data[0].RoleName)
		assert.Equal(t, 1, res.TotalPages)
	})

	t.Run("totalPages rounds up", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			findBySignupFlowIDPaginatedFn: func(_ int64, _, _ int) ([]model.SignupFlowRole, int64, error) {
				return nil, 11, nil // 11 items / 10 per page = 2 pages
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(sf.SignupFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 2, res.TotalPages)
	})

	t.Run("exact page boundary", func(t *testing.T) {
		db, _ := newMockGormDB(t)
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			findBySignupFlowIDPaginatedFn: func(_ int64, _, _ int) ([]model.SignupFlowRole, int64, error) {
				return nil, 10, nil // 10 items / 10 per page = 1 page (no rounding)
			},
		}, &mockRoleRepo{}, defaultCR())
		res, err := svc.GetRoles(sf.SignupFlowUUID, 1, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, res.TotalPages)
	})
}

// ---------------------------------------------------------------------------
// RemoveRole
// ---------------------------------------------------------------------------

func TestSignupFlowService_RemoveRole(t *testing.T) {
	sf := buildSignupFlow()
	roleUUID := uuid.New()

	t.Run("flow not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return nil, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, defaultCR())
		err := svc.RemoveRole(sf.SignupFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "signup flow not found")
	})

	t.Run("role not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) { return nil, nil },
		}, defaultCR())
		err := svc.RemoveRole(sf.SignupFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("DeleteBySignupFlowIDAndRoleID error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{
			deleteBySignupFlowIDAndRoleIDFn: func(_, _ int64) error { return errors.New("del err") },
		}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return &model.Role{RoleID: 10, RoleUUID: roleUUID}, nil
			},
		}, defaultCR())
		err := svc.RemoveRole(sf.SignupFlowUUID, 1, roleUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "del err")
	})

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		svc := NewSignupFlowService(db, &mockSignupFlowRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) { return sf, nil },
		}, &mockSignupFlowRoleRepo{}, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.Role, error) {
				return &model.Role{RoleID: 10, RoleUUID: roleUUID}, nil
			},
		}, defaultCR())
		err := svc.RemoveRole(sf.SignupFlowUUID, 1, roleUUID)
		require.NoError(t, err)
	})
}
