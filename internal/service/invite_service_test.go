package service

import (
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteService_SendInvite(t *testing.T) {
	cases := []struct {
		name         string
		setupRepos   func(*mockClientRepo, *mockRoleRepo, *mockInviteRepo)
		expectCommit bool
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name: "client findDefault error",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return nil, errors.New("db error") }
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "client is nil - invalid client",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return nil, nil }
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid client or identity provider",
		},
		{
			name: "active client with no identity provider - invalid",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				// Client has no IdentityProvider set
				c.findDefaultFn = func() (*model.Client, error) {
					return &model.Client{Status: model.StatusActive}, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid client or identity provider",
		},
		{
			name: "role findByUUIDs error",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findDefaultFn = func() (*model.Client, error) {
					domain := "example.com"
					return &model.Client{
						ClientID: 1,
						Status:   model.StatusActive,
						Domain:   &domain,
						IdentityProvider: &model.IdentityProvider{
							Tenant: &model.Tenant{TenantID: 10},
						},
					}, nil
				}
				r.findByUUIDsFn = func(_ []string, _ ...string) ([]model.Role, error) {
					return nil, errors.New("db error")
				}
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "role count mismatch - one or more roles not found",
			setupRepos: func(c *mockClientRepo, r *mockRoleRepo, i *mockInviteRepo) {
				c.findDefaultFn = func() (*model.Client, error) {
					domain := "example.com"
					return &model.Client{
						ClientID: 1,
						Status:   model.StatusActive,
						Domain:   &domain,
						IdentityProvider: &model.IdentityProvider{
							Tenant: &model.Tenant{TenantID: 10},
						},
					}, nil
				}
				// Return fewer roles than requested
				r.findByUUIDsFn = func(_ []string, _ ...string) ([]model.Role, error) {
					return []model.Role{}, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "one or more roles not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gormDB, mock := newMockGormDB(t)
			mock.ExpectBegin()
			if tc.expectCommit {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}

			clientRepo := &mockClientRepo{}
			roleRepo := &mockRoleRepo{}
			inviteRepo := &mockInviteRepo{}
			tc.setupRepos(clientRepo, roleRepo, inviteRepo)

			svc := NewInviteService(gormDB, inviteRepo, clientRepo, roleRepo, &mockEmailTemplateRepo{})
			result, err := svc.SendInvite(1, "user@example.com", 1, []string{"role-uuid-1"})

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
				if tc.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

