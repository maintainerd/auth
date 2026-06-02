package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountService_VerifyEmailChange(t *testing.T) {
	userID := int64(42)
	userUUID := uuid.New()
	pendingEmail := "new@example.com"
	validOTP := "123456"
	validHash := crypto.HashAuthorizationCode(validOTP)
	future := time.Now().Add(time.Hour)

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
