package user

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/notifier"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// emailChangeOTPRepo is a UserOTPRepository that hands back one pending
// email-change OTP and records whether it was consumed.
type emailChangeOTPRepo struct {
	mockUserOTPRepo
	record   *notifier.UserOTP
	markUsed bool
}

func (r *emailChangeOTPRepo) FindValidByUserAndChannel(int64, string) (*notifier.UserOTP, error) {
	return r.record, nil
}

func (r *emailChangeOTPRepo) MarkUsed(int64) error {
	r.markUsed = true
	return nil
}

func pendingEmailChangeOTP(t *testing.T, userID int64, otp, newEmail string) *notifier.UserOTP {
	t.Helper()
	meta, err := json.Marshal(map[string]string{emailChangePendingEmailKey: newEmail})
	require.NoError(t, err)
	return &notifier.UserOTP{
		UserOTPID: 5,
		UserID:    userID,
		Channel:   emailChangeChannel,
		Recipient: newEmail,
		OTPHash:   crypto.HashAuthorizationCode(otp),
		Metadata:  datatypes.JSON(meta),
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func TestAccountService_VerifyEmailChange_LostUniquenessRace(t *testing.T) {
	const otp = "123456"
	userID := int64(42)
	user := &User{UserID: userID, UserUUID: uuid.New(), TenantID: 1, Email: "old@example.com"}

	otpRepo := &emailChangeOTPRepo{record: pendingEmailChangeOTP(t, userID, otp, "new@example.com")}
	db, _ := newMockGormDB(t)
	svc := &accountService{
		db:               db,
		authEventService: authevent.NoopService(),
		smsOtpRepo:       otpRepo,
		userRepo: &mockUserRepo{
			findByIDFn:    func(any, ...string) (*User, error) { return user, nil },
			updateEmailFn: func(uuid.UUID, string) error { return gorm.ErrDuplicatedKey },
			findByEmailFn: func(string) (*User, error) { return nil, nil },
		},
	}

	err := svc.VerifyEmailChange(context.Background(), userID, otp)

	// A lost race is a 409, not a 500. Wrapping it in NewInternal made it one:
	// HandleServiceError tests errors.As(&internal) BEFORE
	// errors.Is(gorm.ErrDuplicatedKey), so the duplicate-key backstop was
	// unreachable behind an Internal wrapper.
	require.Error(t, err)
	var conflict *apperror.ConflictError
	assert.ErrorAs(t, err, &conflict, "a lost uniqueness race must surface as a conflict")

	// And the loser must not have burned their single-use OTP on a write that
	// never landed — they used to have to restart the whole inbox round trip
	// because of someone else's collision.
	assert.False(t, otpRepo.markUsed, "the OTP must survive a failed write")
}

func TestAccountService_VerifyEmailChange_NotifiesPreviousAddress(t *testing.T) {
	const otp = "123456"
	userID := int64(42)
	user := &User{UserID: userID, UserUUID: uuid.New(), TenantID: 1, Email: "old@example.com"}

	origSend := email.SendEmail
	t.Cleanup(func() { email.SendEmail = origSend })
	var sentTo []string
	var sentBody string
	email.SendEmail = func(_ context.Context, _ *gorm.DB, p email.SendEmailParams) error {
		sentTo = append(sentTo, p.To)
		sentBody = p.BodyPlain
		return nil
	}

	otpRepo := &emailChangeOTPRepo{record: pendingEmailChangeOTP(t, userID, otp, "new@example.com")}
	db, _ := newMockGormDB(t)
	svc := &accountService{
		db:               db,
		authEventService: authevent.NoopService(),
		smsOtpRepo:       otpRepo,
		userRepo: &mockUserRepo{
			findByIDFn:    func(any, ...string) (*User, error) { return user, nil },
			updateEmailFn: func(uuid.UUID, string) error { return nil },
		},
	}

	require.NoError(t, svc.VerifyEmailChange(context.Background(), userID, otp))

	// The out-of-band notice to the address that just LOST the account is the
	// standard — and here the only — signal that makes an email-change takeover
	// detectable by the real owner.
	require.Equal(t, []string{"old@example.com"}, sentTo)
	assert.Contains(t, sentBody, "old@example.com")
	assert.Contains(t, sentBody, "new@example.com")
	assert.True(t, otpRepo.markUsed, "a successful change still consumes the OTP")
}

func TestAccountService_VerifyEmailChange_SendFailureDoesNotFailTheChange(t *testing.T) {
	const otp = "123456"
	userID := int64(42)
	user := &User{UserID: userID, UserUUID: uuid.New(), TenantID: 1, Email: "old@example.com"}

	origSend := email.SendEmail
	t.Cleanup(func() { email.SendEmail = origSend })
	email.SendEmail = func(context.Context, *gorm.DB, email.SendEmailParams) error {
		return errors.New("smtp down")
	}

	otpRepo := &emailChangeOTPRepo{record: pendingEmailChangeOTP(t, userID, otp, "new@example.com")}
	db, _ := newMockGormDB(t)
	svc := &accountService{
		db:               db,
		authEventService: authevent.NoopService(),
		smsOtpRepo:       otpRepo,
		userRepo: &mockUserRepo{
			findByIDFn:    func(any, ...string) (*User, error) { return user, nil },
			updateEmailFn: func(uuid.UUID, string) error { return nil },
		},
	}

	// The change has already committed; a best-effort notice must not roll it
	// back or surface an error the user cannot act on.
	require.NoError(t, svc.VerifyEmailChange(context.Background(), userID, otp))
}

func TestAccountService_ChangeUsername_LostUniquenessRaceIsConflict(t *testing.T) {
	userID := int64(42)
	pw, err := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	require.NoError(t, err)
	pwStr := string(pw)
	user := &User{UserID: userID, UserUUID: uuid.New(), TenantID: 1, Password: &pwStr}

	db, _ := newMockGormDB(t)
	svc := &accountService{
		db:               db,
		authEventService: authevent.NoopService(),
		userRepo: &mockUserRepo{
			findByIDFn:       func(any, ...string) (*User, error) { return user, nil },
			findByUsernameFn: func(string) (*User, error) { return nil, nil },
			updateUsernameFn: func(uuid.UUID, string) error { return gorm.ErrDuplicatedKey },
		},
	}

	err = svc.ChangeUsername(context.Background(), userID, "newname", "pw")

	require.Error(t, err)
	var conflict *apperror.ConflictError
	assert.ErrorAs(t, err, &conflict)
}
