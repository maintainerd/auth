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

func buildAPIKey() *model.APIKey {
	return &model.APIKey{
		APIKeyUUID:  uuid.New(),
		TenantID:    1,
		Name:        "test-key",
		Description: "desc",
		KeyPrefix:   "ak_abcdefgh",
		KeyHash:     "abc123hash",
		Status:      model.StatusActive,
	}
}

func newAPIKeySvc(t *testing.T, akRepo *mockAPIKeyRepo, userRepo *mockUserRepo) APIKeyService {
	t.Helper()
	gormDB, _ := newMockGormDB(t)
	return NewAPIKeyService(gormDB, akRepo, &mockAPIKeyApiRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, userRepo, &mockPermissionRepo{})
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestAPIKeyService_GetByUUID(t *testing.T) {
	ak := buildAPIKey()
	requesterUUID := uuid.New()

	cases := []struct {
		name    string
		setup   func(*mockAPIKeyRepo)
		wantErr string
	}{
		{
			name: "not found",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*model.APIKey, error) { return nil, nil }
			},
			wantErr: "API key not found",
		},
		{
			name: "repo error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*model.APIKey, error) { return nil, errors.New("db error") }
			},
			wantErr: "db error",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*model.APIKey, error) { return ak, nil }
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akRepo := &mockAPIKeyRepo{}
			tc.setup(akRepo)
			mock.ExpectBegin()
			if tc.wantErr != "" {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyApiRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{})
			res, err := svc.GetByUUID(ak.APIKeyUUID, 1, requesterUUID)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, ak.Name, res.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAPIKeyService_Create(t *testing.T) {
	ak := buildAPIKey()

	cases := []struct {
		name    string
		setup   func(*mockAPIKeyRepo)
		wantErr string
	}{
		{
			name: "repo error",
			setup: func(r *mockAPIKeyRepo) {
				r.createFn = func(_ *model.APIKey) (*model.APIKey, error) { return nil, errors.New("db error") }
			},
			wantErr: "db error",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.createFn = func(e *model.APIKey) (*model.APIKey, error) { return ak, nil }
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akRepo := &mockAPIKeyRepo{}
			tc.setup(akRepo)
			mock.ExpectBegin()
			if tc.wantErr != "" {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyApiRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{})
			res, plainKey, err := svc.Create(1, "test-key", "desc", nil, nil, nil, model.StatusActive)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
				assert.NotEmpty(t, plainKey)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SetStatusByUUID
// ---------------------------------------------------------------------------

func TestAPIKeyService_SetStatusByUUID(t *testing.T) {
	ak := buildAPIKey()

	cases := []struct {
		name    string
		setup   func(*mockAPIKeyRepo)
		wantErr string
	}{
		{
			name: "not found",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*model.APIKey, error) { return nil, nil }
			},
			wantErr: "API key not found",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*model.APIKey, error) { return ak, nil }
				r.updateByUUIDFn = func(_, _ any) (*model.APIKey, error) { return ak, nil }
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akRepo := &mockAPIKeyRepo{}
			tc.setup(akRepo)
			mock.ExpectBegin()
			if tc.wantErr != "" {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyApiRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{})
			res, err := svc.SetStatusByUUID(ak.APIKeyUUID, 1, model.StatusActive)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, ak.Name, res.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAPIKeyService_Delete(t *testing.T) {
	ak := buildAPIKey()
	deleterUUID := uuid.New()

	cases := []struct {
		name    string
		setup   func(*mockAPIKeyRepo)
		wantErr string
	}{
		{
			name: "not found",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*model.APIKey, error) { return nil, nil }
			},
			wantErr: "API key not found",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*model.APIKey, error) { return ak, nil }
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akRepo := &mockAPIKeyRepo{}
			tc.setup(akRepo)
			mock.ExpectBegin()
			if tc.wantErr != "" {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyApiRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{})
			res, err := svc.Delete(ak.APIKeyUUID, 1, deleterUUID)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, ak.Name, res.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestAPIKeyService_Get(t *testing.T) {
	ak := buildAPIKey()
	requesterUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{}
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil }}
		svc := newAPIKeySvc(t, akRepo, userRepo)
		res, err := svc.Get(APIKeyServiceGetFilter{TenantID: 1, Page: 1, Limit: 10}, requesterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requesting user not found")
		assert.Nil(t, res)
	})

	t.Run("success", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{
			findPaginatedFn: func(_ repository.APIKeyRepositoryGetFilter) (*repository.PaginationResult[model.APIKey], error) {
				return &repository.PaginationResult[model.APIKey]{Data: []model.APIKey{*ak}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
			return &model.User{UserUUID: requesterUUID}, nil
		}}
		svc := newAPIKeySvc(t, akRepo, userRepo)
		res, err := svc.Get(APIKeyServiceGetFilter{TenantID: 1, Page: 1, Limit: 10}, requesterUUID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
	})
}

// ---------------------------------------------------------------------------
// ValidateAPIKey
// ---------------------------------------------------------------------------

func TestAPIKeyService_ValidateAPIKey(t *testing.T) {
	t.Run("not implemented", func(t *testing.T) {
		svc := newAPIKeySvc(t, &mockAPIKeyRepo{}, &mockUserRepo{})
		res, err := svc.ValidateAPIKey("somehash")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
		assert.Nil(t, res)
	})
}
