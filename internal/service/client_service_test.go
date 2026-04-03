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

func buildClientService(t *testing.T, clientRepo *mockClientRepo, idpRepo *mockIdentityProviderRepo, userRepo *mockUserRepo) ClientService {
	t.Helper()
	db, _ := newMockGormDB(t)
	return NewClientService(db, clientRepo, &mockClientURIRepo{}, idpRepo,
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientApiRepo{},
		&mockAPIRepo{}, userRepo, &mockTenantRepo{})
}

func TestClientService_Get(t *testing.T) {
	idpUUID := uuid.New().String()

	t.Run("idp not found - returns empty result", func(t *testing.T) {
		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := buildClientService(t, &mockClientRepo{}, idpRepo, &mockUserRepo{})
		res, err := svc.Get(ClientServiceGetFilter{IdentityProviderUUID: &idpUUID, TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Empty(t, res.Data)
	})

	t.Run("paginate error", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findPaginatedFn: func(_ repository.ClientRepositoryGetFilter) (*repository.PaginationResult[model.Client], error) {
				return nil, errors.New("db error")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.Get(ClientServiceGetFilter{TenantID: 1})
		require.Error(t, err)
	})

	t.Run("success with no filter - returns empty list", func(t *testing.T) {
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		res, err := svc.Get(ClientServiceGetFilter{TenantID: 1})
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestClientService_GetByUUID(t *testing.T) {
	cUUID := uuid.New()

	t.Run("client not found returns error", func(t *testing.T) {
		// default findByUUIDAndTenantIDFn returns nil,nil → "auth client not found"
		svc := buildClientService(t, &mockClientRepo{}, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(cUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error returns error", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return nil, errors.New("db error")
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		_, err := svc.GetByUUID(cUUID, 1)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		clientRepo := &mockClientRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64) (*model.Client, error) {
				return &model.Client{ClientUUID: cUUID}, nil
			},
		}
		svc := buildClientService(t, clientRepo, &mockIdentityProviderRepo{}, &mockUserRepo{})
		res, err := svc.GetByUUID(cUUID, 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestClientService_Create(t *testing.T) {
	actorUUID := uuid.New()

	t.Run("invalid identity provider UUID", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, &mockIdentityProviderRepo{},
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientApiRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})

		_, err := svc.Create(1, "test", "Test", "public", "example.com", nil, "active", false, "not-a-valid-uuid", actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid identity provider UUID")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider not found", func(t *testing.T) {
		gormDB, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idpRepo := &mockIdentityProviderRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.IdentityProvider, error) { return nil, nil },
		}
		svc := NewClientService(gormDB, &mockClientRepo{}, &mockClientURIRepo{}, idpRepo,
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientApiRepo{},
			&mockAPIRepo{}, &mockUserRepo{}, &mockTenantRepo{})

		_, err := svc.Create(1, "test", "Test", "public", "example.com", nil, "active", false, uuid.New().String(), actorUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identity provider not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

