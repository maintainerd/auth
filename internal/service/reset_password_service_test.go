package service

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// tokenColumns returns the column set GORM will SELECT for user_tokens.
var tokenColumns = []string{
	"user_token_id", "user_token_uuid", "user_id", "token_type",
	"token", "user_agent", "ip_address", "is_revoked",
	"expires_at", "created_at", "updated_at",
}

func validTokenRow(tok string, userID int64, tuuid uuid.UUID) *sqlmock.Rows {
	future := time.Now().Add(time.Hour)
	return sqlmock.NewRows(tokenColumns).AddRow(
		int64(1), tuuid, userID, model.TokenTypePasswordReset,
		tok, nil, nil, false,
		&future, time.Now(), nil,
	)
}

func expiredTokenRow(tok string, userID int64, tuuid uuid.UUID) *sqlmock.Rows {
	past := time.Now().Add(-time.Hour)
	return sqlmock.NewRows(tokenColumns).AddRow(
		int64(1), tuuid, userID, model.TokenTypePasswordReset,
		tok, nil, nil, false,
		&past, time.Now(), nil,
	)
}

func revokedTokenRow(tok string, userID int64, tuuid uuid.UUID) *sqlmock.Rows {
	future := time.Now().Add(time.Hour)
	return sqlmock.NewRows(tokenColumns).AddRow(
		int64(1), tuuid, userID, model.TokenTypePasswordReset,
		tok, nil, nil, true,
		&future, time.Now(), nil,
	)
}

// strongPassword satisfies ValidatePasswordStrength.
const strongPassword = "Str0ng!Pass#99"

// ---------------------------------------------------------------------------
// ResetPassword
// ---------------------------------------------------------------------------

func TestResetPasswordService_ResetPassword(t *testing.T) {
	tok := "some-token"
	userID := int64(42)
	tokenUUID := uuid.New()
	userUUID := uuid.New()
	clientID := "cid"
	providerID := "pid"

	// ----- Client resolution errors -----

	t.Run("FindDefault error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) { return nil, errors.New("db error") },
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to find auth client")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindDefault nil client → invalid client credentials", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) { return nil, nil },
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid client credentials")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByClientIDAndIdentityProvider error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
				return nil, errors.New("client lookup error")
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, &clientID, &providerID)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to find auth client")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByClientIDAndIdentityProvider nil → invalid client credentials", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
				return nil, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, &clientID, &providerID)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid client credentials")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- Token lookup errors -----

	t.Run("token query DB error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnError(errors.New("query error"))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to find reset token")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no tokens found → invalid or expired reset token", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(sqlmock.NewRows(tokenColumns))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid or expired reset token")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- Token validation errors -----

	t.Run("token expired → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(expiredTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "reset token has expired")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("token revoked → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(revokedTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "reset token has been revoked")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- User lookup errors -----

	t.Run("FindByID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return nil, errors.New("user db error")
			},
		}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to find user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID nil user → user not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return nil, nil
			},
		}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "user not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- User status check -----

	t.Run("user inactive → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusInactive}, nil
			},
		}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "user account is not active")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- Password validation -----

	t.Run("weak password → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive}, nil
			},
		}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, "weak", nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "password validation failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- UpdateByID error -----

	t.Run("UpdateByID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive}, nil
			},
			updateByIDFn: func(_, _ any) (*model.User, error) {
				return nil, errors.New("update failed")
			},
		}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to update password")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- RevokeByUUID error -----

	t.Run("RevokeByUUID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive}, nil
			},
		}, &mockUserTokenRepo{
			revokeByUUIDFn: func(_ uuid.UUID) error {
				return errors.New("revoke failed")
			},
		}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to revoke reset token")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- FindByUserIDAndTokenType error -----

	t.Run("FindByUserIDAndTokenType error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive}, nil
			},
		}, &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
				return nil, errors.New("find tokens error")
			},
		}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to find existing tokens")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- Revoke other token error -----

	t.Run("revoke other existing token error → rollback", func(t *testing.T) {
		otherUUID := uuid.New()
		revokeCallCount := 0
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive}, nil
			},
		}, &mockUserTokenRepo{
			revokeByUUIDFn: func(id uuid.UUID) error {
				revokeCallCount++
				// First call succeeds (revoking the current token)
				if revokeCallCount == 1 {
					return nil
				}
				// Second call (revoking the other token) fails
				return errors.New("revoke other failed")
			},
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
				return []model.UserToken{
					{UserTokenUUID: tokenUUID}, // same as current (skipped)
					{UserTokenUUID: otherUUID}, // other token
				}, nil
			},
		}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to revoke existing token")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// ----- Success cases -----

	t.Run("success with FindDefault (no clientID/providerID)", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectCommit()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive, Email: "test@test.com"}, nil
			},
		}, &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
				return []model.UserToken{{UserTokenUUID: tokenUUID}}, nil // only the current token
			},
		}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Message, "Password has been reset successfully")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with clientID and providerID", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectCommit()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive, Email: "test@test.com"}, nil
			},
		}, &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
				return nil, nil
			},
		}, &mockClientRepo{
			findByClientIDAndIdentityProviderFn: func(_, _ string) (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, &clientID, &providerID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success revoking other tokens", func(t *testing.T) {
		otherUUID := uuid.New()
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectCommit()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive, Email: "test@test.com"}, nil
			},
		}, &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
				return []model.UserToken{
					{UserTokenUUID: tokenUUID},
					{UserTokenUUID: otherUUID},
				}, nil
			},
		}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("token with nil ExpiresAt → not expired", func(t *testing.T) {
		// Token with nil ExpiresAt should pass expiry check
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		rows := sqlmock.NewRows(tokenColumns).AddRow(
			int64(1), tokenUUID, userID, model.TokenTypePasswordReset,
			tok, nil, nil, false,
			nil, time.Now(), nil, // ExpiresAt is nil
		)
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).WillReturnRows(rows)
		mock.ExpectCommit()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive, Email: "test@test.com"}, nil
			},
		}, &mockUserTokenRepo{
			findByUserIDAndTokenTypeFn: func(_ int64, _ string) ([]model.UserToken, error) {
				return nil, nil
			},
		}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("HashPassword error", func(t *testing.T) {
		origHash := util.HashPassword
		defer func() { util.HashPassword = origHash }()
		util.HashPassword = func(_ []byte) ([]byte, error) { return nil, errors.New("hash error") }

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "user_tokens"`).
			WillReturnRows(validTokenRow(tok, userID, tokenUUID))
		mock.ExpectRollback()
		svc := NewResetPasswordService(db, &mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: userID, UserUUID: userUUID, Status: model.StatusActive, Email: "test@test.com"}, nil
			},
		}, &mockUserTokenRepo{}, &mockClientRepo{
			findDefaultFn: func() (*model.Client, error) {
				return &model.Client{ClientID: 1}, nil
			},
		})
		resp, err := svc.ResetPassword(tok, strongPassword, nil, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "hash error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
