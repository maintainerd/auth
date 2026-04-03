package service

import (
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetPasswordService_ResetPassword(t *testing.T) {
	cases := []struct {
		name         string
		setupClient  func(*mockClientRepo)
		expectCommit bool
		wantErr      bool
		wantErrMsg   string
	}{
		{
			name: "client findDefault error",
			setupClient: func(c *mockClientRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return nil, errors.New("db error") }
			},
			expectCommit: false,
			wantErr:      true,
		},
		{
			name: "client is nil - invalid client credentials",
			setupClient: func(c *mockClientRepo) {
				c.findDefaultFn = func() (*model.Client, error) { return nil, nil }
			},
			expectCommit: false,
			wantErr:      true,
			wantErrMsg:   "invalid client credentials",
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

			svc := NewResetPasswordService(gormDB, &mockUserRepo{}, &mockUserTokenRepo{}, clientRepo)
			resp, err := svc.ResetPassword("some-token", "NewP@ssw0rd!", nil, nil)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
				if tc.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tc.wantErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

