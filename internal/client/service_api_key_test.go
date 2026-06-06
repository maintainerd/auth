package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func buildAPIKey() *APIKey {
	return &APIKey{
		APIKeyID:    1,
		APIKeyUUID:  uuid.New(),
		TenantID:    1,
		Name:        "test-key",
		Description: "desc",
		KeyPrefix:   "ak_abcdefgh",
		KeyHash:     "abc123hash",
		Status:      shared.StatusActive,
	}
}

func newAPIKeySvc(t *testing.T, akRepo *mockAPIKeyRepo, userRepo *mockUserRepo) APIKeyService {
	t.Helper()
	gormDB, _ := newMockGormDB(t)
	return NewAPIKeyService(gormDB, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, userRepo, &mockPermissionRepo{}, nil)
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
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, nil }
			},
			wantErr: "API key not found",
		},
		{
			name: "repo error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, errors.New("db error") }
			},
			wantErr: "db error",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
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
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
			res, err := svc.GetByUUID(context.Background(), ak.APIKeyUUID, 1, requesterUUID)
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
				r.createFn = func(_ *APIKey) (*APIKey, error) { return nil, errors.New("db error") }
			},
			wantErr: "db error",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.createFn = func(e *APIKey) (*APIKey, error) { return ak, nil }
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
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
			res, plainKey, err := svc.Create(context.Background(), 1, "test-key", "desc", nil, nil, nil, shared.StatusActive)
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
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, nil }
			},
			wantErr: "API key not found",
		},
		{
			name: "find error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, errors.New("find err") }
			},
			wantErr: "find err",
		},
		{
			name: "update error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
				r.updateByUUIDFn = func(_, _ any) (*APIKey, error) { return nil, errors.New("update err") }
			},
			wantErr: "update err",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
				r.updateByUUIDFn = func(_, _ any) (*APIKey, error) { return ak, nil }
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
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
			res, err := svc.SetStatusByUUID(context.Background(), ak.APIKeyUUID, 1, shared.StatusActive)
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
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, nil }
			},
			wantErr: "API key not found",
		},
		{
			name: "find error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, errors.New("find err") }
			},
			wantErr: "find err",
		},
		{
			name: "delete error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
				r.deleteByUUIDFn = func(_ any) error { return errors.New("delete err") }
			},
			wantErr: "delete err",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
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
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
			res, err := svc.Delete(context.Background(), ak.APIKeyUUID, 1, deleterUUID)
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

	t.Run("user repo error", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{}
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, errors.New("user err") }}
		svc := newAPIKeySvc(t, akRepo, userRepo)
		res, err := svc.Get(context.Background(), APIKeyServiceGetFilter{TenantID: 1, Page: 1, Limit: 10}, requesterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user err")
		assert.Nil(t, res)
	})

	t.Run("user not found", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{}
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil }}
		svc := newAPIKeySvc(t, akRepo, userRepo)
		res, err := svc.Get(context.Background(), APIKeyServiceGetFilter{TenantID: 1, Page: 1, Limit: 10}, requesterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requesting user not found")
		assert.Nil(t, res)
	})

	t.Run("find paginated error", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{
			findPaginatedFn: func(_ APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error) {
				return nil, errors.New("paginate err")
			},
		}
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserUUID: requesterUUID}, nil
		}}
		svc := newAPIKeySvc(t, akRepo, userRepo)
		res, err := svc.Get(context.Background(), APIKeyServiceGetFilter{TenantID: 1, Page: 1, Limit: 10}, requesterUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate err")
		assert.Nil(t, res)
	})

	t.Run("success", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{
			findPaginatedFn: func(_ APIKeyRepositoryGetFilter) (*PaginationResult[APIKey], error) {
				return &PaginationResult[APIKey]{Data: []APIKey{*ak}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) {
			return &User{UserUUID: requesterUUID}, nil
		}}
		svc := newAPIKeySvc(t, akRepo, userRepo)
		res, err := svc.Get(context.Background(), APIKeyServiceGetFilter{TenantID: 1, Page: 1, Limit: 10}, requesterUUID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
	})
}

// ---------------------------------------------------------------------------
// GetConfigByUUID
// ---------------------------------------------------------------------------

func TestAPIKeyService_GetConfigByUUID(t *testing.T) {
	ak := buildAPIKey()
	ak.Config = datatypes.JSON(`{"rate_limit":100}`)

	t.Run("repo error", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) { return nil, errors.New("db err") },
		}
		svc := newAPIKeySvc(t, akRepo, &mockUserRepo{})
		cfg, err := svc.GetConfigByUUID(context.Background(), ak.APIKeyUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key not found")
		assert.Nil(t, cfg)
	})

	t.Run("not found", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) { return nil, nil },
		}
		svc := newAPIKeySvc(t, akRepo, &mockUserRepo{})
		cfg, err := svc.GetConfigByUUID(context.Background(), ak.APIKeyUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key not found")
		assert.Nil(t, cfg)
	})

	t.Run("success", func(t *testing.T) {
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) { return ak, nil },
		}
		svc := newAPIKeySvc(t, akRepo, &mockUserRepo{})
		cfg, err := svc.GetConfigByUUID(context.Background(), ak.APIKeyUUID, 1)
		require.NoError(t, err)
		assert.JSONEq(t, `{"rate_limit":100}`, string(cfg))
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestAPIKeyService_Update(t *testing.T) {
	ak := buildAPIKey()
	updaterUUID := uuid.New()
	nameStr := "updated-name"
	descStr := "updated-desc"
	statusStr := "inactive"
	now := time.Now()
	rl := 100

	cases := []struct {
		name         string
		setup        func(*mockAPIKeyRepo)
		nameArg      *string
		descArg      *string
		configArg    datatypes.JSON
		expiresArg   *time.Time
		rateLimArg   *int
		statusArg    *string
		wantErr      string
		expectCommit bool
	}{
		{
			name: "find error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, errors.New("find err") }
			},
			wantErr: "find err",
		},
		{
			name: "not found",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, nil }
			},
			wantErr: "API key not found",
		},
		{
			name: "update error",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
				r.updateByUUIDFn = func(_, _ any) (*APIKey, error) { return nil, errors.New("update err") }
			},
			nameArg: &nameStr,
			wantErr: "update err",
		},
		{
			name: "success with all fields",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
				r.updateByUUIDFn = func(_, _ any) (*APIKey, error) { return ak, nil }
			},
			nameArg:      &nameStr,
			descArg:      &descStr,
			configArg:    datatypes.JSON(`{"key":"val"}`),
			expiresArg:   &now,
			rateLimArg:   &rl,
			statusArg:    &statusStr,
			expectCommit: true,
		},
		{
			name: "success with no optional fields",
			setup: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return ak, nil }
				r.updateByUUIDFn = func(_, _ any) (*APIKey, error) { return ak, nil }
			},
			expectCommit: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akRepo := &mockAPIKeyRepo{}
			tc.setup(akRepo)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			svc := NewAPIKeyService(gormDB, akRepo, &mockAPIKeyAPIRepo{}, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
			res, err := svc.Update(context.Background(), ak.APIKeyUUID, 1, tc.nameArg, tc.descArg, tc.configArg, tc.expiresArg, tc.rateLimArg, tc.statusArg, updaterUUID)
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
// GetAPIKeyAPIs
// ---------------------------------------------------------------------------

func TestAPIKeyService_GetAPIKeyAPIs(t *testing.T) {
	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()

	t.Run("defaults and success with permissions", func(t *testing.T) {
		akaRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDPaginatedFn: func(_ uuid.UUID, page, limit int, _, _ string) (*PaginationResult[APIKeyAPI], error) {
				assert.Equal(t, 1, page)
				assert.Equal(t, pagination.DefaultPageSize, limit)
				return &PaginationResult[APIKeyAPI]{
					Data: []APIKeyAPI{
						{
							APIKeyAPIUUID: uuid.New(),
							API: API{
								APIUUID:     apiUUID,
								Name:        "test-api",
								DisplayName: "Test API",
								Description: "desc",
								APIType:     "rest",
								Identifier:  "test",
								Status:      "active",
								IsSystem:    false,
							},
							Permissions: []APIKeyPermission{
								{
									Permission: &Permission{
										PermissionUUID: permUUID,
										Name:           "read",
										Description:    "Read access",
										Status:         "active",
										IsDefault:      true,
										IsSystem:       false,
									},
								},
							},
						},
					},
					Total: 1, Page: 1, Limit: pagination.DefaultPageSize, TotalPages: 1,
				}, nil
			},
		}
		gormDB, _ := newMockGormDB(t)
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
				return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewAPIKeyService(gormDB, akRepo, akaRepo, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
		// Pass 0 for page/limit to test defaults
		res, err := svc.GetAPIKeyAPIs(context.Background(), tenantID, akUUID, 0, 0, "", "")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
		assert.Equal(t, "test-api", res.Data[0].Api.Name)
		assert.Len(t, res.Data[0].Permissions, 1)
		assert.Equal(t, "read", res.Data[0].Permissions[0].Name)
	})

	t.Run("repo error", func(t *testing.T) {
		akaRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDPaginatedFn: func(_ uuid.UUID, _, _ int, _, _ string) (*PaginationResult[APIKeyAPI], error) {
				return nil, errors.New("paginate err")
			},
		}
		gormDB, _ := newMockGormDB(t)
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
				return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewAPIKeyService(gormDB, akRepo, akaRepo, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
		res, err := svc.GetAPIKeyAPIs(context.Background(), tenantID, akUUID, 1, 10, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "paginate err")
		assert.Nil(t, res)
	})
}

// ---------------------------------------------------------------------------
// AddAPIKeyAPIs
// ---------------------------------------------------------------------------

func TestAPIKeyService_AddAPIKeyAPIs(t *testing.T) {
	akUUID := uuid.New()
	apiUUID1 := uuid.New()
	apiUUID2 := uuid.New()

	buildAK := func() *APIKey {
		return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}
	}
	buildAPI := func(id int64, u uuid.UUID) *API {
		return &API{APIID: id, APIUUID: u, TenantID: tenantID}
	}

	cases := []struct {
		name         string
		apiUUIDs     []uuid.UUID
		setupAK      func(*mockAPIKeyRepo)
		setupAKA     func(*mockAPIKeyAPIRepo)
		setupAPI     func(*mockAPIRepo)
		wantErr      string
		expectCommit bool
	}{
		{
			name:     "api key find error",
			apiUUIDs: []uuid.UUID{apiUUID1},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, errors.New("ak find err") }
			},
			setupAKA: func(_ *mockAPIKeyAPIRepo) {},
			setupAPI: func(_ *mockAPIRepo) {},
			wantErr:  "ak find err",
		},
		{
			name:     "api key not found",
			apiUUIDs: []uuid.UUID{apiUUID1},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return nil, nil }
			},
			setupAKA: func(_ *mockAPIKeyAPIRepo) {},
			setupAPI: func(_ *mockAPIRepo) {},
			wantErr:  "API key not found",
		},
		{
			name:     "api find error",
			apiUUIDs: []uuid.UUID{apiUUID1},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return buildAK(), nil }
			},
			setupAKA: func(_ *mockAPIKeyAPIRepo) {},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return nil, errors.New("api find err") }
			},
			wantErr: "api find err",
		},
		{
			name:     "api not found",
			apiUUIDs: []uuid.UUID{apiUUID1},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return buildAK(), nil }
			},
			setupAKA: func(_ *mockAPIKeyAPIRepo) {},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return nil, nil }
			},
			wantErr: "API not found",
		},
		{
			name:     "find existing relationship error",
			apiUUIDs: []uuid.UUID{apiUUID1},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return buildAK(), nil }
			},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyAndAPIFn = func(_, _ int64) (*APIKeyAPI, error) { return nil, errors.New("find rel err") }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return buildAPI(10, apiUUID1), nil }
			},
			wantErr: "find rel err",
		},
		{
			name:     "skip existing relationship",
			apiUUIDs: []uuid.UUID{apiUUID1},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return buildAK(), nil }
			},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyAndAPIFn = func(_, _ int64) (*APIKeyAPI, error) { return &APIKeyAPI{}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return buildAPI(10, apiUUID1), nil }
			},
			expectCommit: true,
		},
		{
			name:     "create error",
			apiUUIDs: []uuid.UUID{apiUUID1},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return buildAK(), nil }
			},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.createFn = func(_ *APIKeyAPI) (*APIKeyAPI, error) { return nil, errors.New("create err") }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return buildAPI(10, apiUUID1), nil }
			},
			wantErr: "create err",
		},
		{
			name:     "success with multiple apis",
			apiUUIDs: []uuid.UUID{apiUUID1, apiUUID2},
			setupAK: func(r *mockAPIKeyRepo) {
				r.findByUUIDAndTenantIDFn = func(_ string, _ int64) (*APIKey, error) { return buildAK(), nil }
			},
			setupAKA: func(_ *mockAPIKeyAPIRepo) {},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(id any, _ ...string) (*API, error) {
					u := id.(uuid.UUID)
					if u == apiUUID1 {
						return buildAPI(10, apiUUID1), nil
					}
					return buildAPI(11, apiUUID2), nil
				}
			},
			expectCommit: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akRepo := &mockAPIKeyRepo{}
			akaRepo := &mockAPIKeyAPIRepo{}
			apiRepo := &mockAPIRepo{}
			tc.setupAK(akRepo)
			tc.setupAKA(akaRepo)
			tc.setupAPI(apiRepo)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			svc := NewAPIKeyService(gormDB, akRepo, akaRepo, &mockAPIKeyPermissionRepo{}, apiRepo, &mockUserRepo{}, &mockPermissionRepo{}, nil)
			err := svc.AddAPIKeyAPIs(context.Background(), tenantID, akUUID, tc.apiUUIDs)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RemoveAPIKeyAPI
// ---------------------------------------------------------------------------

func TestAPIKeyService_RemoveAPIKeyAPI(t *testing.T) {
	akUUID := uuid.New()
	apiUUID := uuid.New()

	cases := []struct {
		name         string
		setup        func(*mockAPIKeyAPIRepo)
		wantErr      string
		expectCommit bool
	}{
		{
			name: "find error",
			setup: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, errors.New("find err") }
			},
			wantErr: "find err",
		},
		{
			name: "not found",
			setup: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, nil }
			},
			wantErr: "API key API relationship not found",
		},
		{
			name: "remove error",
			setup: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{}, nil }
				r.removeByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) error { return errors.New("remove err") }
			},
			wantErr: "remove err",
		},
		{
			name: "success",
			setup: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{}, nil }
			},
			expectCommit: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akaRepo := &mockAPIKeyAPIRepo{}
			tc.setup(akaRepo)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			akRepo := &mockAPIKeyRepo{
				findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
					return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
				},
			}
			svc := NewAPIKeyService(gormDB, akRepo, akaRepo, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
			err := svc.RemoveAPIKeyAPI(context.Background(), tenantID, akUUID, apiUUID)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetAPIKeyAPIPermissions
// ---------------------------------------------------------------------------

func TestAPIKeyService_GetAPIKeyAPIPermissions(t *testing.T) {
	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()

	t.Run("find relationship error", func(t *testing.T) {
		akaRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, errors.New("find err") },
		}
		gormDB, _ := newMockGormDB(t)
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
				return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewAPIKeyService(gormDB, akRepo, akaRepo, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
		res, err := svc.GetAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find err")
		assert.Nil(t, res)
	})

	t.Run("relationship not found", func(t *testing.T) {
		akaRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, nil },
		}
		gormDB, _ := newMockGormDB(t)
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
				return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewAPIKeyService(gormDB, akRepo, akaRepo, &mockAPIKeyPermissionRepo{}, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
		res, err := svc.GetAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key API relationship not found")
		assert.Nil(t, res)
	})

	t.Run("find permissions error", func(t *testing.T) {
		akaRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*APIKeyAPI, error) {
				return &APIKeyAPI{APIKeyAPIID: 1}, nil
			},
		}
		akpRepo := &mockAPIKeyPermissionRepo{
			findByAPIKeyAPIIDFn: func(_ int64) ([]APIKeyPermission, error) { return nil, errors.New("perm err") },
		}
		gormDB, _ := newMockGormDB(t)
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
				return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewAPIKeyService(gormDB, akRepo, akaRepo, akpRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
		res, err := svc.GetAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "perm err")
		assert.Nil(t, res)
	})

	t.Run("success", func(t *testing.T) {
		akaRepo := &mockAPIKeyAPIRepo{
			findByAPIKeyUUIDAndAPIUUIDFn: func(_, _ uuid.UUID) (*APIKeyAPI, error) {
				return &APIKeyAPI{APIKeyAPIID: 1}, nil
			},
		}
		akpRepo := &mockAPIKeyPermissionRepo{
			findByAPIKeyAPIIDFn: func(_ int64) ([]APIKeyPermission, error) {
				return []APIKeyPermission{
					{Permission: &Permission{PermissionUUID: permUUID, Name: "read", Description: "Read", Status: "active", IsDefault: true}},
				}, nil
			},
		}
		gormDB, _ := newMockGormDB(t)
		akRepo := &mockAPIKeyRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
				return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
			},
		}
		svc := NewAPIKeyService(gormDB, akRepo, akaRepo, akpRepo, &mockAPIRepo{}, &mockUserRepo{}, &mockPermissionRepo{}, nil)
		res, err := svc.GetAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID)
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.Equal(t, "read", res[0].Name)
	})
}

// ---------------------------------------------------------------------------
// AddAPIKeyAPIPermissions
// ---------------------------------------------------------------------------

func TestAPIKeyService_AddAPIKeyAPIPermissions(t *testing.T) {
	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID1 := uuid.New()

	cases := []struct {
		name         string
		permUUIDs    []uuid.UUID
		setupAKA     func(*mockAPIKeyAPIRepo)
		setupAPI     func(*mockAPIRepo)
		setupPerm    func(*mockPermissionRepo)
		setupAKP     func(*mockAPIKeyPermissionRepo)
		wantErr      string
		expectCommit bool
	}{
		{
			name:      "find relationship error",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, errors.New("aka err") }
			},
			setupAPI:  func(_ *mockAPIRepo) {},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "aka err",
		},
		{
			name:      "relationship not found",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, nil }
			},
			setupAPI:  func(_ *mockAPIRepo) {},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "API key API relationship not found",
		},
		{
			name:      "api find error",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return nil, errors.New("api err") }
			},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "api err",
		},
		{
			name:      "api not found",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return nil, nil }
			},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "API not found",
		},
		{
			name:      "permission find error",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) { return nil, errors.New("perm err") }
			},
			setupAKP: func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:  "perm err",
		},
		{
			name:      "permission not found",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) { return nil, nil }
			},
			setupAKP: func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:  "permission not found",
		},
		{
			name:      "permission wrong api",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 999, TenantID: tenantID}, nil // Wrong API
				}
			},
			setupAKP: func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:  "permission does not belong to the specified API",
		},
		{
			name:      "find existing permission error",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 10, TenantID: tenantID}, nil
				}
			},
			setupAKP: func(r *mockAPIKeyPermissionRepo) {
				r.findByAPIKeyAPIAndPermissionFn = func(_, _ int64) (*APIKeyPermission, error) { return nil, errors.New("find akp err") }
			},
			wantErr: "find akp err",
		},
		{
			name:      "skip existing permission",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 10, TenantID: tenantID}, nil
				}
			},
			setupAKP: func(r *mockAPIKeyPermissionRepo) {
				r.findByAPIKeyAPIAndPermissionFn = func(_, _ int64) (*APIKeyPermission, error) { return &APIKeyPermission{}, nil }
			},
			expectCommit: true,
		},
		{
			name:      "create permission error",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 10, TenantID: tenantID}, nil
				}
			},
			setupAKP: func(r *mockAPIKeyPermissionRepo) {
				r.createFn = func(_ *APIKeyPermission) (*APIKeyPermission, error) {
					return nil, errors.New("create perm err")
				}
			},
			wantErr: "create perm err",
		},
		{
			name:      "success",
			permUUIDs: []uuid.UUID{permUUID1},
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 10, TenantID: tenantID}, nil
				}
			},
			setupAKP:     func(_ *mockAPIKeyPermissionRepo) {},
			expectCommit: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akaRepo := &mockAPIKeyAPIRepo{}
			apiRepo := &mockAPIRepo{}
			permRepo := &mockPermissionRepo{}
			akpRepo := &mockAPIKeyPermissionRepo{}
			tc.setupAKA(akaRepo)
			tc.setupAPI(apiRepo)
			tc.setupPerm(permRepo)
			tc.setupAKP(akpRepo)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			akRepo := &mockAPIKeyRepo{
				findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
					return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
				},
			}
			svc := NewAPIKeyService(gormDB, akRepo, akaRepo, akpRepo, apiRepo, &mockUserRepo{}, permRepo, nil)
			err := svc.AddAPIKeyAPIPermissions(context.Background(), tenantID, akUUID, apiUUID, tc.permUUIDs)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RemoveAPIKeyAPIPermission
// ---------------------------------------------------------------------------

func TestAPIKeyService_RemoveAPIKeyAPIPermission(t *testing.T) {
	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()

	cases := []struct {
		name         string
		setupAKA     func(*mockAPIKeyAPIRepo)
		setupAPI     func(*mockAPIRepo)
		setupPerm    func(*mockPermissionRepo)
		setupAKP     func(*mockAPIKeyPermissionRepo)
		wantErr      string
		expectCommit bool
	}{
		{
			name: "find relationship error",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, errors.New("aka err") }
			},
			setupAPI:  func(_ *mockAPIRepo) {},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "aka err",
		},
		{
			name: "relationship not found",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return nil, nil }
			},
			setupAPI:  func(_ *mockAPIRepo) {},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "API key API relationship not found",
		},
		{
			name: "api find error",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return nil, errors.New("api err") }
			},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "api err",
		},
		{
			name: "api not found",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return nil, nil }
			},
			setupPerm: func(_ *mockPermissionRepo) {},
			setupAKP:  func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:   "API not found",
		},
		{
			name: "permission find error",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) { return nil, errors.New("perm err") }
			},
			setupAKP: func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:  "perm err",
		},
		{
			name: "permission not found",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) { return nil, nil }
			},
			setupAKP: func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:  "permission not found",
		},
		{
			name: "permission wrong api",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 999, TenantID: tenantID}, nil
				}
			},
			setupAKP: func(_ *mockAPIKeyPermissionRepo) {},
			wantErr:  "permission does not belong to the specified API",
		},
		{
			name: "remove error",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 10, TenantID: tenantID}, nil
				}
			},
			setupAKP: func(r *mockAPIKeyPermissionRepo) {
				r.removeByAPIKeyAPIAndPermissionFn = func(_, _ int64) error { return errors.New("remove err") }
			},
			wantErr: "remove err",
		},
		{
			name: "success",
			setupAKA: func(r *mockAPIKeyAPIRepo) {
				r.findByAPIKeyUUIDAndAPIUUIDFn = func(_, _ uuid.UUID) (*APIKeyAPI, error) { return &APIKeyAPI{APIKeyAPIID: 1}, nil }
			},
			setupAPI: func(r *mockAPIRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*API, error) { return &API{APIID: 10, TenantID: tenantID}, nil }
			},
			setupPerm: func(r *mockPermissionRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*Permission, error) {
					return &Permission{PermissionID: 1, APIID: 10, TenantID: tenantID}, nil
				}
			},
			setupAKP:     func(_ *mockAPIKeyPermissionRepo) {},
			expectCommit: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			akaRepo := &mockAPIKeyAPIRepo{}
			apiRepo := &mockAPIRepo{}
			permRepo := &mockPermissionRepo{}
			akpRepo := &mockAPIKeyPermissionRepo{}
			tc.setupAKA(akaRepo)
			tc.setupAPI(apiRepo)
			tc.setupPerm(permRepo)
			tc.setupAKP(akpRepo)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			akRepo := &mockAPIKeyRepo{
				findByUUIDAndTenantIDFn: func(_ string, _ int64) (*APIKey, error) {
					return &APIKey{APIKeyID: 1, APIKeyUUID: akUUID, TenantID: tenantID}, nil
				},
			}
			svc := NewAPIKeyService(gormDB, akRepo, akaRepo, akpRepo, apiRepo, &mockUserRepo{}, permRepo, nil)
			err := svc.RemoveAPIKeyAPIPermission(context.Background(), tenantID, akUUID, apiUUID, permUUID)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
