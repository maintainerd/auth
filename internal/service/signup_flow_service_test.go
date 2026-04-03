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

func newSignupFlowSvc(t *testing.T, sfRepo *mockSignupFlowRepo, sfrRepo *mockSignupFlowRoleRepo, roleRepo *mockRoleRepo, clientRepo *mockClientRepo) SignupFlowService {
	t.Helper()
	gormDB, _ := newMockGormDB(t)
	return NewSignupFlowService(gormDB, sfRepo, sfrRepo, roleRepo, clientRepo)
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestSignupFlowService_GetByUUID(t *testing.T) {
	sf := buildSignupFlow()
	sfUUID := sf.SignupFlowUUID

	cases := []struct {
		name    string
		setup   func(*mockSignupFlowRepo)
		wantErr string
	}{
		{
			name: "not found",
			setup: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return nil, nil
				}
			},
			wantErr: "signup flow not found",
		},
		{
			name: "repo error",
			setup: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: "signup flow not found",
		},
		{
			name: "success",
			setup: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return sf, nil
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sfRepo := &mockSignupFlowRepo{}
			tc.setup(sfRepo)
			svc := newSignupFlowSvc(t, sfRepo, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, &mockClientRepo{findDefaultFn: func() (*model.Client, error) { return nil, nil }, findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) { return nil, nil }})
			res, err := svc.GetByUUID(sfUUID, 1)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Nil(t, res)
			} else {
				require.NoError(t, err)
				assert.Equal(t, sf.Name, res.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetAll
// ---------------------------------------------------------------------------

func TestSignupFlowService_GetAll(t *testing.T) {
	sf := buildSignupFlow()
	sfRepo := &mockSignupFlowRepo{
		findPaginatedFn: func(_ repository.SignupFlowRepositoryGetFilter) (*repository.PaginationResult[model.SignupFlow], error) {
			return &repository.PaginationResult[model.SignupFlow]{Data: []model.SignupFlow{*sf}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
		},
	}
	cr := &mockClientRepo{findDefaultFn: func() (*model.Client, error) { return nil, nil }, findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) { return nil, nil }}
	svc := newSignupFlowSvc(t, sfRepo, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
	res, err := svc.GetAll(1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Len(t, res.Data, 1)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestSignupFlowService_Create(t *testing.T) {
	sf := buildSignupFlow()
	clientUUID := uuid.New()

	cases := []struct {
		name    string
		setupCR func(*mockClientRepo)
		setupSF func(*mockSignupFlowRepo)
		wantErr string
	}{
		{
			name: "client not found",
			setupCR: func(cr *mockClientRepo) {
				cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) { return nil, nil }
			},
			setupSF: func(_ *mockSignupFlowRepo) {},
			wantErr: "auth client not found",
		},
		{
			name: "name already exists",
			setupCR: func(cr *mockClientRepo) {
				cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) {
					return &model.Client{ClientID: 1}, nil
				}
			},
			setupSF: func(r *mockSignupFlowRepo) {
				r.findByNameFn = func(_ string) (*model.SignupFlow, error) {
					return &model.SignupFlow{Name: "test-flow"}, nil
				}
			},
			wantErr: "name already exists",
		},
		{
			name: "success",
			setupCR: func(cr *mockClientRepo) {
				cr.findByUUIDFn = func(_ any, _ ...string) (*model.Client, error) {
					return &model.Client{ClientID: 1}, nil
				}
			},
			setupSF: func(r *mockSignupFlowRepo) {
				r.findByNameFn = func(_ string) (*model.SignupFlow, error) { return nil, nil }
				r.createFn = func(e *model.SignupFlow) (*model.SignupFlow, error) { return sf, nil }
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return sf, nil
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			cr := &mockClientRepo{findDefaultFn: func() (*model.Client, error) { return nil, nil }, findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) { return nil, nil }}
			tc.setupCR(cr)
			sfRepo := &mockSignupFlowRepo{}
			tc.setupSF(sfRepo)
			if tc.wantErr == "" {
				mock.ExpectBegin()
				mock.ExpectCommit()
			} else {
				mock.ExpectBegin()
				mock.ExpectRollback()
			}
			svc := NewSignupFlowService(gormDB, sfRepo, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
			res, err := svc.Create(1, "test-flow", "desc", nil, model.StatusActive, clientUUID)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestSignupFlowService_Delete(t *testing.T) {
	sf := buildSignupFlow()
	sfUUID := sf.SignupFlowUUID

	cases := []struct {
		name    string
		setup   func(*mockSignupFlowRepo)
		wantErr string
	}{
		{
			name: "not found",
			setup: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return nil, nil
				}
			},
			wantErr: "signup flow not found",
		},
		{
			name: "success",
			setup: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return sf, nil
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sfRepo := &mockSignupFlowRepo{}
			tc.setup(sfRepo)
			cr := &mockClientRepo{findDefaultFn: func() (*model.Client, error) { return nil, nil }, findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) { return nil, nil }}
			svc := newSignupFlowSvc(t, sfRepo, &mockSignupFlowRoleRepo{}, &mockRoleRepo{}, cr)
			res, err := svc.Delete(sfUUID, 1)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, sf.Name, res.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AssignRoles
// ---------------------------------------------------------------------------

func TestSignupFlowService_AssignRoles(t *testing.T) {
	sf := buildSignupFlow()
	role := &model.Role{RoleID: 10, RoleUUID: uuid.New(), Name: "editor", Status: model.StatusActive}

	cases := []struct {
		name    string
		setupSF func(*mockSignupFlowRepo)
		setupRL func(*mockRoleRepo)
		wantErr string
	}{
		{
			name: "flow not found",
			setupSF: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return nil, nil
				}
			},
			setupRL: func(_ *mockRoleRepo) {},
			wantErr: "signup flow not found",
		},
		{
			name: "role not found",
			setupSF: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return sf, nil
				}
			},
			setupRL: func(r *mockRoleRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return nil, nil }
			},
			wantErr: "role not found",
		},
		{
			name: "success",
			setupSF: func(r *mockSignupFlowRepo) {
				r.findByUUIDAndTenantIDFn = func(_ uuid.UUID, _ int64, _ ...string) (*model.SignupFlow, error) {
					return sf, nil
				}
			},
			setupRL: func(r *mockRoleRepo) {
				r.findByUUIDFn = func(_ any, _ ...string) (*model.Role, error) { return role, nil }
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			sfRepo := &mockSignupFlowRepo{}
			tc.setupSF(sfRepo)
			roleRepo := &mockRoleRepo{}
			tc.setupRL(roleRepo)
			if tc.wantErr == "" {
				mock.ExpectBegin()
				mock.ExpectCommit()
			} else {
				mock.ExpectBegin()
				mock.ExpectRollback()
			}
			cr := &mockClientRepo{findDefaultFn: func() (*model.Client, error) { return nil, nil }, findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) { return nil, nil }}
			svc := NewSignupFlowService(gormDB, sfRepo, &mockSignupFlowRoleRepo{}, roleRepo, cr)
			res, err := svc.AssignRoles(sf.SignupFlowUUID, 1, []uuid.UUID{role.RoleUUID})
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Len(t, res, 1)
			}
		})
	}
}
