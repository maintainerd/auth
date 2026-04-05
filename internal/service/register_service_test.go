package service

import (
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterService_RegisterPublic(t *testing.T) {
	cases := []struct {
		name         string
		setupRepos   func(*mockClientRepo, *mockIdentityProviderRepo, *mockUserRepo)
		expectCommit bool
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name: "client not found - invalid or inactive auth client",
			setupRepos: func(c *mockClientRepo, idp *mockIdentityProviderRepo, u *mockUserRepo) {
				c.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					return nil, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid or inactive auth client",
		},
		{
			name: "client repo error",
			setupRepos: func(c *mockClientRepo, idp *mockIdentityProviderRepo, u *mockUserRepo) {
				c.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					return nil, errors.New("db error")
				}
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "client inactive - invalid",
			setupRepos: func(c *mockClientRepo, idp *mockIdentityProviderRepo, u *mockUserRepo) {
				c.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					domain := "example.com"
					return &model.Client{Status: model.StatusInactive, Domain: &domain}, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid or inactive auth client",
		},
		{
			name: "identity provider not found",
			setupRepos: func(c *mockClientRepo, idp *mockIdentityProviderRepo, u *mockUserRepo) {
				c.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					domain := "example.com"
					return &model.Client{Status: model.StatusActive, Domain: &domain}, nil
				}
				idp.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return nil, nil
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "identity provider not found",
		},
		{
			name: "identity provider lookup error",
			setupRepos: func(c *mockClientRepo, idp *mockIdentityProviderRepo, u *mockUserRepo) {
				c.findByClientIDAndIdentityProviderFn = func(_, _ string) (*model.Client, error) {
					domain := "example.com"
					return &model.Client{Status: model.StatusActive, Domain: &domain}, nil
				}
				idp.findByIdentifierFn = func(_ string) (*model.IdentityProvider, error) {
					return nil, errors.New("db error")
				}
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "identity provider lookup failed",
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
			idpRepo := &mockIdentityProviderRepo{}
			userRepo := &mockUserRepo{}
			tc.setupRepos(clientRepo, idpRepo, userRepo)

			svc := NewRegistrationService(gormDB, clientRepo, userRepo, &mockUserRoleRepo{}, &mockUserTokenRepo{},
				&mockUserIdentityRepo{}, &mockRoleRepo{}, &mockInviteRepo{}, idpRepo, &mockTenantUserRepo{})

			resp, err := svc.RegisterPublic("testuser", "Test User", "P@ssw0rd!",
				nil, nil, "client-id", "provider-id")

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
				if tc.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
