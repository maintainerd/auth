package user

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/platform/sms"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
		case UserMFABackupCodeRepository:
			svc.mfaBackupCodeRepo = v
		case UserIdentityRepository:
			svc.userIdentityRepo = v
		case IdentityProviderRepository:
			svc.identityProviderRepo = v
		case notifier.UserOTPRepository:
			svc.smsOtpRepo = v
		case UserAnonymizer:
			svc.anonymizer = v
		}
	}
	return svc
}

func TestNewAccountService(t *testing.T) {
	db, _ := newMockGormDB(t)
	svc := NewAccountService(db, &mockUserRepo{}, &mockUserTokenRepo{}, &mockProfileRepo{},
		&mockUserSettingRepo{}, &mockRoleRepo{}, &mockClientRepo{}, &mockUserMFABackupCodeRepo{},
		&mockUserIdentityRepo{}, &mockIdentityProviderRepo{}, authevent.NoopService(), nil, &mockUserOTPRepo{}, nil, nil, nil)
	assert.NotNil(t, svc)
}

func TestAccountService_InitiateEmailChange(t *testing.T) {
	origSend := email.SendEmail
	email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error { return nil }
	t.Cleanup(func() { email.SendEmail = origSend })

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

	t.Run("OTP create repo error", func(t *testing.T) {
		orig := crypto.GenerateOTP
		crypto.GenerateOTP = func(int) (string, error) { return "123456", nil }
		defer func() { crypto.GenerateOTP = orig }()

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_otps"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		}, &mockUserOTPRepo{createFn: func(*notifier.UserOTP) (*notifier.UserOTP, error) {
			return nil, errors.New("create failed")
		}})
		svc.db = db

		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to store pending email")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GenerateOTP error", func(t *testing.T) {
		orig := crypto.GenerateOTP
		crypto.GenerateOTP = func(int) (string, error) { return "", errors.New("otp error") }
		defer func() { crypto.GenerateOTP = orig }()

		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		})
		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate OTP")
	})

	t.Run("success", func(t *testing.T) {
		orig := crypto.GenerateOTP
		crypto.GenerateOTP = func(int) (string, error) { return "123456", nil }
		defer func() { crypto.GenerateOTP = orig }()

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_otps"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		// The send is synchronous now, so the render path runs for real and its
		// queries must be expected. Previously it happened in a detached
		// goroutine, so this test passed while the email silently failed —
		// asserting success about something it never exercised.
		// Non-empty so GetLogoURL short-circuits its system-tenant fallback.
		mock.ExpectQuery(`FROM "email_config"`).
			WillReturnRows(sqlmock.NewRows([]string{"logo_url"}).AddRow("https://logo.test/x.png"))
		mock.ExpectQuery(`FROM "email_templates"`).
			WillReturnRows(sqlmock.NewRows([]string{"subject", "body_html", "body_plain"}).
				AddRow("Confirm your email", "<p>{{.OTP}}</p>", "{{.OTP}}"))

		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		}, &mockUserOTPRepo{})
		svc.db = db

		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.NoError(t, err)
	})

	// A delivery failure must reach the caller. The endpoint used to return
	// success while the send failed in the background, so the UI said "check
	// your inbox" for a mail that was never sent.
	t.Run("a send failure is returned, not swallowed", func(t *testing.T) {
		origSend := email.SendEmail
		email.SendEmail = func(_ context.Context, _ *gorm.DB, _ email.SendEmailParams) error {
			return errors.New("smtp: connection refused")
		}
		defer func() { email.SendEmail = origSend }()

		orig := crypto.GenerateOTP
		crypto.GenerateOTP = func(int) (string, error) { return "123456", nil }
		defer func() { crypto.GenerateOTP = orig }()

		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_otps"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		// Non-empty so GetLogoURL short-circuits its system-tenant fallback.
		mock.ExpectQuery(`FROM "email_config"`).
			WillReturnRows(sqlmock.NewRows([]string{"logo_url"}).AddRow("https://logo.test/x.png"))
		mock.ExpectQuery(`FROM "email_templates"`).
			WillReturnRows(sqlmock.NewRows([]string{"subject", "body_html", "body_plain"}).
				AddRow("Confirm your email", "<p>{{.OTP}}</p>", "{{.OTP}}"))

		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
			findByEmailFn: func(_ string) (*User, error) { return nil, nil },
		}, &mockUserOTPRepo{})
		svc.db = db

		err := svc.InitiateEmailChange(context.Background(), userID, "new@example.com", "correctpass")
		require.Error(t, err, "the caller must learn the email did not go out")
	})
}

func TestAccountService_VerifyEmailChange(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()
	pendingEmail := "new@example.com"
	validOTP := "123456"
	validHash := crypto.HashAuthorizationCode(validOTP)

	makeValidOTPRecord := func(hash string) *notifier.UserOTP {
		meta, _ := json.Marshal(map[string]string{"pending_email": pendingEmail})
		return &notifier.UserOTP{
			UserOTPID: 1,
			OTPHash:   hash,
			Metadata:  datatypes.JSON(meta),
		}
	}

	t.Run("user not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		})
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("no pending email change (OTP lookup returns nil)", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
			},
			&mockUserOTPRepo{findValidByUserFn: func(int64, string) (*notifier.UserOTP, error) {
				return nil, nil
			}},
		)
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no pending email change")
	})

	t.Run("invalid OTP (hash mismatch)", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
			},
			&mockUserOTPRepo{
				findValidByUserFn: func(int64, string) (*notifier.UserOTP, error) {
					return makeValidOTPRecord(validHash), nil
				},
			},
		)
		err := svc.VerifyEmailChange(context.Background(), userID, "000000")
		require.Error(t, err)
		var unauthorized *apperror.UnauthorizedError
		assert.ErrorAs(t, err, &unauthorized)
	})

	t.Run("stored hash is not accepted as plaintext OTP", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
			},
			&mockUserOTPRepo{
				findValidByUserFn: func(int64, string) (*notifier.UserOTP, error) {
					return makeValidOTPRecord(validHash), nil
				},
			},
		)
		err := svc.VerifyEmailChange(context.Background(), userID, validHash)
		require.Error(t, err)
		var unauthorized *apperror.UnauthorizedError
		assert.ErrorAs(t, err, &unauthorized)
	})

	t.Run("mark used error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
			},
			&mockUserOTPRepo{
				findValidByUserFn: func(int64, string) (*notifier.UserOTP, error) {
					return makeValidOTPRecord(validHash), nil
				},
				markUsedFn: func(int64) error { return errors.New("mark used failed") },
			},
		)
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark OTP used")
	})

	t.Run("UpdateEmail repo error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
				updateEmailFn: func(_ uuid.UUID, _ string) error { return errors.New("db error") },
			},
			&mockUserOTPRepo{
				findValidByUserFn: func(int64, string) (*notifier.UserOTP, error) {
					return makeValidOTPRecord(validHash), nil
				},
			},
		)
		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update email")
	})

	t.Run("success marks OTP used and updates email", func(t *testing.T) {
		var updatedEmail string
		var markedUsed bool
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID}, nil
				},
				updateEmailFn: func(id uuid.UUID, email string) error {
					assert.Equal(t, userUUID, id)
					updatedEmail = email
					return nil
				},
			},
			&mockUserOTPRepo{
				findValidByUserFn: func(int64, string) (*notifier.UserOTP, error) {
					return makeValidOTPRecord(validHash), nil
				},
				markUsedFn: func(id int64) error {
					assert.Equal(t, int64(1), id)
					markedUsed = true
					return nil
				},
			},
		)

		err := svc.VerifyEmailChange(context.Background(), userID, validOTP)

		require.NoError(t, err)
		assert.Equal(t, pendingEmail, updatedEmail)
		assert.True(t, markedUsed)
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

	// Self-service deletion now routes through the canonical anonymization
	// cascade (UserAnonymizer) instead of clearing a partial subset inline, so a
	// missing anonymizer fails closed and a failed anonymize aborts the deletion.
	t.Run("anonymizer unavailable", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(_ any, _ ...string) (*User, error) {
				return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
			},
		})
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account deletion is unavailable")
	})

	t.Run("anonymize error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
			},
			&mockAnonymizer{fn: func(int64) error { return errors.New("anon err") }},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "anon err")
	})

	t.Run("deactivate (UpdateByID) error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return nil, errors.New("db error") },
			},
			&mockAnonymizer{},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deactivate account")
	})

	t.Run("success", func(t *testing.T) {
		anonymized := int64(0)
		svc := newAccountSvc(
			&mockUserRepo{
				findByIDFn: func(_ any, _ ...string) (*User, error) {
					return &User{UserID: userID, UserUUID: userUUID, Password: &hashedPass}, nil
				},
				updateByIDFn: func(_, _ any) (*User, error) { return &User{}, nil },
			},
			&mockAnonymizer{fn: func(uid int64) error { anonymized = uid; return nil }},
		)
		err := svc.DeleteAccount(context.Background(), userID, "correctpass")
		require.NoError(t, err)
		assert.Equal(t, userID, anonymized)
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

// TestAccountService_NoBackupCodeGeneration is the INVERSION of the former
// TestAccountService_GenerateBackupCodes, which asserted that accountService
// minted 10 bcrypt-hashed codes.
//
// The behaviour it locked in was wrong at the policy layer, not the crypto
// layer: the generator hardcoded 10 codes and never consulted the tenant's MFA
// policy, so a tenant that forbids the backup_code method or configures a
// different recovery_codes_count had that decision bypassed by whoever called
// POST /account/backup-codes (which no SPA did). Generation is now owned solely
// by mfaService.RegenerateBackupCodes, which checks both. What this asserts is
// that accountService can no longer mint codes at all — only verify them.
func TestAccountService_NoBackupCodeGeneration(t *testing.T) {
	_, mints := reflect.TypeOf((*AccountService)(nil)).Elem().MethodByName("GenerateBackupCodes")
	assert.False(t, mints, "backup-code generation must live only on mfaService.RegenerateBackupCodes, which enforces tenant MFA policy")
}

func TestAccountService_VerifyBackupCode(t *testing.T) {
	initJWTKeys(t)

	userUUID := uuid.New()
	userID := int64(42)
	backupCodeID := int64(100)
	clientIDStr := "test-client"
	providerID := "test-provider"
	code := "12345678"
	// bcrypt, matching what both the account and mfa services now write. The
	// old fixture used an unsalted crypto.HashAuthorizationCode digest, which is
	// exactly the scheme the verifier could never match.
	codeHashBytes, hashErr := bcrypt.GenerateFromPassword([]byte(code), bcrypt.MinCost)
	require.NoError(t, hashErr)
	codeHash := string(codeHashBytes)
	// Backup-code recovery is now password + code, so every fixture user needs a
	// real password hash and every request a matching password.
	password := "correct-horse-battery"
	passwordHashBytes, pwErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, pwErr)
	passwordHash := string(passwordHashBytes)
	domain := "example.com"
	identifier := "client-id"

	t.Run("success", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		user := &User{
			UserID:   userID,
			UserUUID: userUUID,
			Email:    "test@example.com",
			Password: &passwordHash,
			Status:   shared.StatusActive,
		}
		backupCode := &UserMFABackupCode{
			BackupCodeID: backupCodeID,
			UserID:       userID,
			CodeHash:     codeHash,
		}
		userIdentity := &UserIdentity{
			UserIdentityUUID:   uuid.New(),
			UserID:             userID,
			IdentityProviderID: 1,
			Sub:                "test-sub",
		}

		svc := &accountService{
			db: db,
			// Recovery binds its token to a session; without a creator it fails closed.
			sessionCreator: stubSessionCreator{},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) { return user, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			userIdentityRepo: &mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return userIdentity, nil },
			},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{
				findUnusedByUserIDFn: func(int64) ([]UserMFABackupCode, error) { return []UserMFABackupCode{*backupCode}, nil },
			},
			authEventService: authevent.NoopService(),
		}

		res, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.AccessToken)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("identity provider not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		svc := &accountService{
			db:                db,
			userRepo:          &mockUserRepo{},
			clientRepo:        &mockClientRepo{},
			userIdentityRepo:  &mockUserIdentityRepo{},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return nil, nil },
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("client not active", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		svc := &accountService{
			db:                db,
			userRepo:          &mockUserRepo{},
			userIdentityRepo:  &mockUserIdentityRepo{},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) {
					return &Client{Status: shared.StatusInactive}, nil
				},
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found by email", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		svc := &accountService{
			db:                db,
			userIdentityRepo:  &mockUserIdentityRepo{},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) { return nil, nil },
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email, password, or backup code")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("find user by email error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		svc := &accountService{
			db:                db,
			userIdentityRepo:  &mockUserIdentityRepo{},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) { return nil, assert.AnError },
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to look up user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user status not active", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		svc := &accountService{
			db:                db,
			userIdentityRepo:  &mockUserIdentityRepo{},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) {
					return &User{UserID: userID, Status: shared.StatusInactive}, nil
				},
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account is not active")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("backup code lookup error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		user := &User{
			UserID:   userID,
			Email:    "test@example.com",
			Password: &passwordHash,
			Status:   shared.StatusActive,
		}
		svc := &accountService{
			db:               db,
			userIdentityRepo: &mockUserIdentityRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) { return user, nil },
			},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{
				findUnusedByUserIDFn: func(int64) ([]UserMFABackupCode, error) {
					return nil, assert.AnError
				},
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify backup code")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("backup code not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		user := &User{
			UserID:   userID,
			Email:    "test@example.com",
			Password: &passwordHash,
			Status:   shared.StatusActive,
		}
		svc := &accountService{
			db:               db,
			userIdentityRepo: &mockUserIdentityRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) { return user, nil },
			},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{
				findUnusedByUserIDFn: func(int64) ([]UserMFABackupCode, error) { return nil, nil },
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email, password, or backup code")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("mark backup code used error", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		user := &User{
			UserID:   userID,
			UserUUID: userUUID,
			Email:    "test@example.com",
			Password: &passwordHash,
			Status:   shared.StatusActive,
		}
		backupCode := &UserMFABackupCode{
			BackupCodeID: backupCodeID,
			UserID:       userID,
			CodeHash:     codeHash,
		}
		svc := &accountService{
			db:               db,
			userIdentityRepo: &mockUserIdentityRepo{},
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) { return user, nil },
			},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{
				findUnusedByUserIDFn: func(int64) ([]UserMFABackupCode, error) { return []UserMFABackupCode{*backupCode}, nil },
				markUsedFn:           func(_ int64) error { return assert.AnError },
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark backup code as used")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user identity not found", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		idp := &IdentityProvider{Identifier: identifier}
		client := &Client{
			ClientID:   1,
			Domain:     &domain,
			Identifier: &identifier,
			Status:     shared.StatusActive,
			IdentityProvider: &IdentityProvider{
				Identifier: identifier,
			},
		}
		user := &User{
			UserID:   userID,
			UserUUID: userUUID,
			Email:    "test@example.com",
			Password: &passwordHash,
			Status:   shared.StatusActive,
		}
		backupCode := &UserMFABackupCode{
			BackupCodeID: backupCodeID,
			UserID:       userID,
			CodeHash:     codeHash,
		}
		svc := &accountService{
			db: db,
			identityProviderRepo: &mockIdentityProviderRepo{
				findByIdentifierFn: func(_ string) (*IdentityProvider, error) { return idp, nil },
			},
			clientRepo: &mockClientRepo{
				findByClientIDAndIdentityProviderFn: func(_, _ string) (*Client, error) { return client, nil },
			},
			userRepo: &mockUserRepo{
				findByEmailFn: func(_ string) (*User, error) { return user, nil },
			},
			mfaBackupCodeRepo: &mockUserMFABackupCodeRepo{
				findUnusedByUserIDFn: func(int64) ([]UserMFABackupCode, error) { return []UserMFABackupCode{*backupCode}, nil },
			},
			userIdentityRepo: &mockUserIdentityRepo{
				findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil },
			},
			authEventService: authevent.NoopService(),
		}

		_, err := svc.VerifyBackupCode(context.Background(), VerifyBackupCodeDTO{
			Email:      "test@example.com",
			Password:   password,
			Code:       code,
			ClientID:   clientIDStr,
			ProviderID: providerID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAccountService_generateTokenResponse_Errors(t *testing.T) {
	initJWTKeys(t)
	domain := "https://auth.example.com"
	identifier := "client-id"
	client := &Client{
		Domain:     &domain,
		Identifier: &identifier,
		IdentityProvider: &IdentityProvider{
			Identifier: "provider",
		},
	}
	user := &User{Email: "test@example.com", Phone: "+15555550100"}
	svc := &accountService{}

	t.Run("access token error", func(t *testing.T) {
		origAccess := accountGenerateAccessTokenWithContext
		accountGenerateAccessTokenWithOptionsContext = func(context.Context, string, string, string, string, string, string, *jwt.AccessTokenOptions) (string, error) {
			return "", assert.AnError
		}
		t.Cleanup(func() { accountGenerateAccessTokenWithContext = origAccess })

		_, err := svc.generateTokenResponse(context.Background(), "sub", user, client, "11111111-1111-1111-1111-111111111111")

		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("id token error", func(t *testing.T) {
		origAccess := accountGenerateAccessTokenWithContext
		origID := accountGenerateIDTokenWithContext
		accountGenerateAccessTokenWithOptionsContext = func(context.Context, string, string, string, string, string, string, *jwt.AccessTokenOptions) (string, error) {
			return "access", nil
		}
		accountGenerateIDTokenWithContext = func(context.Context, string, string, string, string, *jwt.UserProfile, string, *jwt.IDTokenParams) (string, error) {
			return "", assert.AnError
		}
		t.Cleanup(func() {
			accountGenerateAccessTokenWithContext = origAccess
			accountGenerateIDTokenWithContext = origID
		})

		_, err := svc.generateTokenResponse(context.Background(), "sub", user, client, "11111111-1111-1111-1111-111111111111")

		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("refresh token error", func(t *testing.T) {
		origAccess := accountGenerateAccessTokenWithContext
		origID := accountGenerateIDTokenWithContext
		origRefresh := accountGenerateRefreshTokenWithContext
		accountGenerateAccessTokenWithOptionsContext = func(context.Context, string, string, string, string, string, string, *jwt.AccessTokenOptions) (string, error) {
			return "access", nil
		}
		accountGenerateIDTokenWithContext = func(context.Context, string, string, string, string, *jwt.UserProfile, string, *jwt.IDTokenParams) (string, error) {
			return "id", nil
		}
		accountGenerateRefreshTokenWithContext = func(context.Context, string, string, string, string) (string, error) {
			return "", assert.AnError
		}
		t.Cleanup(func() {
			accountGenerateAccessTokenWithContext = origAccess
			accountGenerateIDTokenWithContext = origID
			accountGenerateRefreshTokenWithContext = origRefresh
		})

		_, err := svc.generateTokenResponse(context.Background(), "sub", user, client, "11111111-1111-1111-1111-111111111111")

		require.ErrorIs(t, err, assert.AnError)
	})
}

func initJWTKeys(t *testing.T) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})
	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	require.NoError(t, jwt.InitJWTKeys())
}

func TestAccountService_SendPhoneVerification(t *testing.T) {
	const testUserID int64 = 42
	const testPhone = "+15550001111"

	t.Run("phone missing returns validation error", func(t *testing.T) {
		svc := newAccountSvc()
		err := svc.SendPhoneVerification(context.Background(), testUserID, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "phone is required")
	})

	t.Run("user not found returns not found", func(t *testing.T) {
		svc := newAccountSvc(&mockUserRepo{
			findByIDFn: func(any, ...string) (*User, error) { return nil, nil },
		})
		err := svc.SendPhoneVerification(context.Background(), testUserID, testPhone)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("store error returns internal error", func(t *testing.T) {
		orig := accountNewSMSProvider
		t.Cleanup(func() { accountNewSMSProvider = orig })
		accountNewSMSProvider = func(context.Context, *gorm.DB, int64) (sms.Provider, error) {
			t.Fatal("provider must not be reached when store fails")
			return nil, nil
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_otps"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		svc := newAccountSvc(
			&mockUserRepo{findByIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: testUserID, TenantID: tenantID}, nil
			}},
			&mockUserOTPRepo{createFn: func(*notifier.UserOTP) (*notifier.UserOTP, error) {
				return nil, errors.New("db down")
			}},
		)
		svc.db = db

		err := svc.SendPhoneVerification(context.Background(), testUserID, testPhone)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to store SMS OTP")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("provider init failure still succeeds and logs", func(t *testing.T) {
		orig := accountNewSMSProvider
		t.Cleanup(func() { accountNewSMSProvider = orig })
		accountNewSMSProvider = func(context.Context, *gorm.DB, int64) (sms.Provider, error) {
			return nil, errors.New("no sms config")
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_otps"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		var stored *notifier.UserOTP
		svc := newAccountSvc(
			&mockUserRepo{findByIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: testUserID, TenantID: tenantID}, nil
			}},
			&mockUserOTPRepo{createFn: func(o *notifier.UserOTP) (*notifier.UserOTP, error) {
				stored = o
				return o, nil
			}},
		)
		svc.db = db

		err := svc.SendPhoneVerification(context.Background(), testUserID, testPhone)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, phoneVerifyChannel, stored.Channel)
		assert.Equal(t, testPhone, stored.Recipient)
		assert.Equal(t, testUserID, stored.UserID)
		assert.NotEmpty(t, stored.OTPHash)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success sends via provider using fallback message", func(t *testing.T) {
		orig := accountNewSMSProvider
		t.Cleanup(func() { accountNewSMSProvider = orig })
		fake := &fakeSMSProvider{}
		accountNewSMSProvider = func(context.Context, *gorm.DB, int64) (sms.Provider, error) {
			return fake, nil
		}
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "user_otps"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		// RenderTemplate reads sms_templates; return an error to exercise the fallback message.
		mock.ExpectQuery(`sms_templates`).WillReturnError(errors.New("no template"))

		svc := newAccountSvc(
			&mockUserRepo{findByIDFn: func(any, ...string) (*User, error) {
				return &User{UserID: testUserID, TenantID: tenantID}, nil
			}},
			&mockUserOTPRepo{},
		)
		svc.db = db

		err := svc.SendPhoneVerification(context.Background(), testUserID, testPhone)
		require.NoError(t, err)
		assert.Equal(t, testPhone, fake.sentTo)
		assert.Contains(t, fake.sentBody, "Your phone verification code is:")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// fakeSMSProvider records the last message sent for assertions.
type fakeSMSProvider struct {
	sentTo   string
	sentBody string
	sendErr  error
}

func (f *fakeSMSProvider) Send(_ context.Context, to, body string) error {
	f.sentTo = to
	f.sentBody = body
	return f.sendErr
}

func TestAccountService_VerifyPhone(t *testing.T) {
	const testUserID int64 = 42
	const testPhone = "+15550001111"
	const testCode = "123456"

	t.Run("no matching OTP returns unauthorized", func(t *testing.T) {
		svc := newAccountSvc(&mockUserOTPRepo{
			findValidFn: func(string, string) (*notifier.UserOTP, error) { return nil, nil },
		})
		err := svc.VerifyPhone(context.Background(), testUserID, testPhone, testCode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired verification code")
	})

	t.Run("wrong code records failure and returns unauthorized", func(t *testing.T) {
		var recorded bool
		svc := newAccountSvc(&mockUserOTPRepo{
			findValidFn: func(string, string) (*notifier.UserOTP, error) {
				return &notifier.UserOTP{UserOTPID: 1, OTPHash: crypto.HashAuthorizationCode("999999")}, nil
			},
			recordFailFn: func(id int64, maxAttempts int) error {
				recorded = true
				assert.Equal(t, int64(1), id)
				assert.Equal(t, phoneVerifyMaxFailed, maxAttempts)
				return nil
			},
		})
		err := svc.VerifyPhone(context.Background(), testUserID, testPhone, testCode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid verification code")
		assert.True(t, recorded, "RecordFailure must be called on a wrong code")
	})

	t.Run("mark used error returns internal error", func(t *testing.T) {
		svc := newAccountSvc(&mockUserOTPRepo{
			findValidFn: func(string, string) (*notifier.UserOTP, error) {
				return &notifier.UserOTP{UserOTPID: 1, OTPHash: crypto.HashAuthorizationCode(testCode)}, nil
			},
			markUsedFn: func(int64) error { return errors.New("db error") },
		})
		err := svc.VerifyPhone(context.Background(), testUserID, testPhone, testCode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to mark verification code used")
	})

	t.Run("update error returns internal error", func(t *testing.T) {
		svc := newAccountSvc(
			&mockUserOTPRepo{
				findValidFn: func(string, string) (*notifier.UserOTP, error) {
					return &notifier.UserOTP{UserOTPID: 1, OTPHash: crypto.HashAuthorizationCode(testCode)}, nil
				},
			},
			&mockUserRepo{updateByIDFn: func(any, any) (*User, error) {
				return nil, errors.New("db error")
			}},
		)
		err := svc.VerifyPhone(context.Background(), testUserID, testPhone, testCode)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update phone")
	})

	t.Run("success marks phone verified", func(t *testing.T) {
		var markedUsed bool
		var gotID any
		var gotData any
		svc := newAccountSvc(
			&mockUserOTPRepo{
				findValidFn: func(channel, recipient string) (*notifier.UserOTP, error) {
					assert.Equal(t, phoneVerifyChannel, channel)
					assert.Equal(t, testPhone, recipient)
					return &notifier.UserOTP{UserOTPID: 7, OTPHash: crypto.HashAuthorizationCode(testCode)}, nil
				},
				markUsedFn: func(id int64) error {
					markedUsed = true
					assert.Equal(t, int64(7), id)
					return nil
				},
			},
			&mockUserRepo{updateByIDFn: func(id, data any) (*User, error) {
				gotID = id
				gotData = data
				return &User{UserID: testUserID, IsPhoneVerified: true}, nil
			}},
		)
		err := svc.VerifyPhone(context.Background(), testUserID, testPhone, testCode)
		require.NoError(t, err)
		assert.True(t, markedUsed, "MarkUsed must be called on success")
		assert.Equal(t, testUserID, gotID)
		updates, ok := gotData.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, updates["is_phone_verified"])
		assert.Equal(t, testPhone, updates["phone"])
	})
}
