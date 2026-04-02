package service

import (
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgotPasswordService_SendPasswordResetEmail(t *testing.T) {
	cases := []struct {
		name         string
		setupClient  func(*mockClientRepo)
		expectCommit bool
		wantErr      bool
	}{
		{
			name: "client findDefault error - returns error",
			setupClient: func(c *mockClientRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return nil, errors.New("db error") }
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "user not found - returns success (security masking)",
			setupClient: func(c *mockClientRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return buildActiveClient(), nil }
			},
			expectCommit: true,
			wantErr:      false,
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
			tc.setupClient(clientRepo)

			svc := NewForgotPasswordService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo, &mockEmailTemplateRepo{})
			resp, err := svc.SendPasswordResetEmail("user@example.com", nil, nil, false)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.True(t, resp.Success)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
