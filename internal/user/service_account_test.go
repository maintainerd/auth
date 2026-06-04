package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAccountSvc(repos ...interface{}) *accountService {
	svc := &accountService{
		authEventService: authevent.NoopService(),
	}
	for _, r := range repos {
		switch v := r.(type) {
		case UserRepository:
			svc.userRepo = v
		case UserTokenRepository:
			svc.userTokenRepo = v
		case ProfileRepository:
			svc.profileRepo = v
		case UserSettingRepository:
			svc.userSettingRepo = v
		case RoleRepository:
			svc.roleRepo = v
		case ClientRepository:
			svc.clientRepo = v
		case UserBackupCodeRepository:
			svc.backupCodeRepo = v
		case UserIdentityRepository:
			svc.userIdentityRepo = v
		case IdentityProviderRepository:
			svc.identityProviderRepo = v
		}
	}
	return svc
}

func TestAccountService_InitiateEmailChange(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()
	hashedPassBytes, _ := security.HashPassword(context.Background(), []byte("correctpass"))
	hashedPass := string(hashedPassBytes)

	t.Run("user not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("FindByID repo error", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, errors.New("db error") },
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("account has no password set", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no password set")
	})

	t.Run("invalid current password", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "wrongpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid current password")
	})

	t.Run("email already taken", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) {
				return &User{UserID: 99}, nil
			},
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already in use")
	})

	t.Run("FindByEmail repo error", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) { return nil, errors.New("db error") },
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check email availability")
	})

	t.Run("SetPendingEmail repo error", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
			setPendingEmailFn: func(_ uuid.UUID, _, _ string, _ time.Time) error {
				return errors.New("db error")
			},
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to store pending email")
	})

	t.Run("success", func(t *testing.T) {
		orig := crypto.GenerateOTP
		crypto.GenerateOTP = func(int) (string, error) { return "123456", nil }
		defer func() { crypto.GenerateOTP = orig }()

		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.NoError(t, err)
	})
}

func TestAccountService_VerifyEmailChange(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()
	pendingEmail := "new@example.com"
	validOTP := "123456"
	validHash := crypto.HashAuthorizationCode(validOTP)
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	t.Run("user not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("no pending email change", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no pending email change")
	})

	t.Run("OTP expired", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID:                  userID,
					UserUUID:                userUUID,
					PendingEmail:            &pendingEmail,
					EmailChangeOTP:          &validHash,
					EmailChangeOTPExpiresAt: &past,
				}, nil
			},
		})
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("UpdateEmail repo error", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID:                  userID,
					UserUUID:                userUUID,
					PendingEmail:            &pendingEmail,
					EmailChangeOTP:          &validHash,
					EmailChangeOTPExpiresAt: &future,
				}, nil
			},
			updateEmailFn: func(_ uuid.UUID, _ string) error { return errors.New("db error") },
		})
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update email")
	})

	t.Run("ClearEmailChange error is non-fatal", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID:                  userID,
					UserUUID:                userUUID,
					PendingEmail:            &pendingEmail,
					EmailChangeOTP:          &validHash,
					EmailChangeOTPExpiresAt: &future,
				}, nil
			},
			updateEmailFn: func(_ uuid.UUID, _ string) error { return nil },
			clearEmailChangeFn: func(_ uuid.UUID) error { return errors.New("clear failed") },
		})
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.NoError(t, err)
	})

	t.Run("success updates email and clears pending change", func(t *testing.T) {
		var updatedEmail string
		var cleared bool
		svc := &accountService{
			userRepo: &mockUserRepo{
				findByIDFn: func(id any, _ ...string) (*User, error) {
					assert.Equal(t, userID, id)
					return &User{
						UserID:                  userID,
						UserUUID:                userUUID,
						PendingEmail:            &pendingEmail,
						EmailChangeOTP:          &validHash,
						EmailChangeOTPExpiresAt: &future,
					}, nil
				},
				updateEmailFn: func(id uuid.UUID, email string) error {
					assert.Equal(t, userUUID, id)
					updatedEmail = email
					return nil
				},
				clearEmailChangeFn: func(id uuid.UUID) error {
					assert.Equal(t, userUUID, id)
					cleared = true
					return nil
				},
			},
		}

		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)

		require.NoError(t, err)
		assert.Equal(t, pendingEmail, updatedEmail)
		assert.True(t, cleared)
	})

	t.Run("invalid OTP does not update email", func(t *testing.T) {
		var updated bool
		svc := &accountService{
			userRepo: &mockUserRepo{
				findByIDFn: func(id any, _ ...string) (*User, error) {
					assert.Equal(t, userID, id)
					return &User{
						UserID:                  userID,
						UserUUID:                userUUID,
						PendingEmail:            &pendingEmail,
						EmailChangeOTP:          &validHash,
						EmailChangeOTPExpiresAt: &future,
					}, nil
				},
				updateEmailFn: func(uuid.UUID, string) error {
					updated = true
					return nil
				},
			},
		}

		err := svc.VerifyEmailChange(context.Background(), userID, "000000")

		require.Error(t, err)
		var unauthorized *apperror.UnauthorizedError
		assert.ErrorAs(t, err, &unauthorized)
		assert.False(t, updated)
	})

	t.Run("stored hash is not accepted as plaintext OTP", func(t *testing.T) {
		var updated bool
		svc := &accountService{
			userRepo: &mockUserRepo{
				findByIDFn: func(id any, _ ...string) (*User, error) {
					assert.Equal(t, userID, id)
					return &User{
						UserID:                  userID,
						UserUUID:                userUUID,
						PendingEmail:            &pendingEmail,
						EmailChangeOTP:          &validHash,
						EmailChangeOTPExpiresAt: &future,
					}, nil
				},
				updateEmailFn: func(uuid.UUID, string) error {
					updated = true
					return nil
				},
			},
		}

		err := svc.VerifyEmailChange(context.Background(), userID, validHash)

		require.Error(t, err)
		var unauthorized *apperror.UnauthorizedError
		assert.ErrorAs(t, err, &unauthorized)
		assert.False(t, updated)
	})
}

func TestAccountService_ChangeUsername(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()
	hashedPassBytes, _ := security.HashPassword(context.Background(), []byte("correctpass"))
	hashedPass := string(hashedPassBytes)

	t.Run("user not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("no password set", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no password set")
	})

	t.Run("invalid current password", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "wrongpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid current password")
	})

	t.Run("FindByUsername repo error", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByUsernameFn: func(_ string) (*User, error) { return nil, errors.New("db error") },
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check username availability")
	})

	t.Run("username already taken by another user", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByUsernameFn: func(_ string) (*User, error) {
				return &User{UserID: 99}, nil
			},
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already taken")
	})

	t.Run("username taken by same user is allowed", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByUsernameFn: func(_ string) (*User, error) {
				return &User{UserID: userID}, nil
			},
			updateUsernameFn: func(_ uuid.UUID, _ string) error { return nil },
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "correctpass")
		require.NoError(t, err)
	})

	t.Run("UpdateUsername repo error", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByUsernameFn: func(_ string) (*User, error) { return nil, nil },
			updateUsernameFn: func(_ uuid.UUID, _ string) error { return errors.New("db error") },
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update username")
	})

	t.Run("success", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByUsernameFn: func(_ string) (*User, error) { return nil, nil },
			updateUsernameFn: func(_ uuid.UUID, _ string) error { return nil },
		})
		err := svc.ChangeUsername(context.Background(), userID, "newuser", "correctpass")
		require.NoError(t, err)
	})
}

func TestAccountService_DeleteAccount(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()
	hashedPassBytes, _ := security.HashPassword(context.Background(), []byte("correctpass"))
	hashedPass := string(hashedPassBytes)

	t.Run("user not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		err := svc.DeleteAccount(context.Background(), userID, "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("no password set", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID}, nil
			},
		})
		err := svc.DeleteAccount(context.Background(), userID, "pass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no password set")
	})

	t.Run("invalid current password", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
		})
		err := svc.DeleteAccount(context.Background(), userID, "wrongpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid current password")
	})

	t.Run("UpdateByID error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return nil, errors.New("db error") },
			},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete account")
	})

	t.Run("RevokeAllByUserID error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return &User{}, nil },
			},
			&mockUserTokenRepo{
				revokeAllByUserIDFn: func(int64) error { return errors.New("db error") },
			},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to revoke account tokens")
	})

	t.Run("DeleteByUserID (token) error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return &User{}, nil },
			},
			&mockUserTokenRepo{
				deleteByUserIDFn: func(int64) error { return errors.New("db error") },
			},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove account tokens")
	})

	t.Run("profile DeleteByUserID error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return &User{}, nil },
			},
			&mockUserTokenRepo{},
			&mockProfileRepo{
				deleteByUserIDFn: func(int64) error { return errors.New("db error") },
			},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove profile data")
	})

	t.Run("userSetting DeleteByUserID error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return &User{}, nil },
			},
			&mockUserTokenRepo{},
			&mockProfileRepo{},
			&mockUserSettingRepo{
				deleteByUserIDFn: func(int64) error { return errors.New("db error") },
			},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove user settings")
	})

	t.Run("identity DeleteByUserID error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return &User{}, nil },
			},
			&mockUserTokenRepo{},
			&mockProfileRepo{},
			&mockUserSettingRepo{},
			&mockUserIdentityRepo{
				deleteByUserIDFn: func(int64) error { return errors.New("db error") },
			},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove linked identities")
	})

	t.Run("success", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return &User{}, nil },
			},
			&mockUserTokenRepo{},
			&mockProfileRepo{},
			&mockUserSettingRepo{},
			&mockUserIdentityRepo{},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.NoError(t, err)
	})
}

func TestAccountService_ExportAccountData(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		data, err := svc.ExportAccountData(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success without profile or settings", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID:   userID,
					UserUUID: userUUID,
					Username: "alice",
					Email:    "alice@example.com",
				}, nil
			},
		})
		data, err := svc.ExportAccountData(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, "alice", data.Username)
		assert.Nil(t, data.Profile)
		assert.Nil(t, data.Settings)
	})

	t.Run("success with profile and settings", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{
					UserID:      userID,
					UserUUID:    userUUID,
					Username:    "alice",
					Email:       "alice@example.com",
					Roles:       []Role{{Name: "admin"}, {Name: "user"}},
					Profile:     &Profile{FirstName: "Alice"},
					UserSetting: &UserSetting{PreferredLanguage: strPtr("en")},
				}, nil
			},
		})
		data, err := svc.ExportAccountData(context.Background(), userID)
		require.NoError(t, err)
		assert.Equal(t, "alice", data.Username)
		assert.NotNil(t, data.Profile)
		assert.NotNil(t, data.Settings)
		assert.Len(t, data.Roles, 2)
	})
}

func TestAccountService_GenerateBackupCodes(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		codes, err := svc.GenerateBackupCodes(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, codes)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("DeleteAllByUserID error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
			},
			&mockUserBackupCodeRepo{
				deleteAllByUserIDFn: func(int64) error { return errors.New("db error") },
			},
		)
		codes, err := svc.GenerateBackupCodes(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, codes)
		assert.Contains(t, err.Error(), "failed to clear existing backup codes")
	})

	t.Run("CreateBulk error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
			},
			&mockUserBackupCodeRepo{
				createBulkFn: func(_ []*UserBackupCode) error { return errors.New("db error") },
			},
		)
		codes, err := svc.GenerateBackupCodes(context.Background(), userID)
		require.Error(t, err)
		assert.Nil(t, codes)
		assert.Contains(t, err.Error(), "failed to store backup codes")
	})

	t.Run("success", func(t *testing.T) {
		orig := crypto.GenerateRandomString
		crypto.GenerateRandomString = func(int) (string, error) { return "abcdefgh", nil }
		defer func() { crypto.GenerateRandomString = orig }()

		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
			},
			&mockUserBackupCodeRepo{},
		)
		codes, err := svc.GenerateBackupCodes(context.Background(), userID)
		require.NoError(t, err)
		assert.NotNil(t, codes)
		assert.Len(t, codes.Codes, 10)
	})
}
