package authn

import (
	"context"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type ResetPasswordService interface {
	ResetPassword(ctx context.Context, token, newPassword string, clientID, providerID *string) (*ResetPasswordResponseDTO, error)
}

type resetPasswordService struct {
	db                  *gorm.DB
	userRepo            UserRepository
	userTokenRepo       UserTokenRepository
	clientRepo          ClientRepository
	securitySettingRepo secpolicy.SecuritySettingRepository // nil → use defaults
	passwordHistoryRepo UserPasswordHistoryRepository       // nil → skip history
}

func NewResetPasswordService(
	db *gorm.DB,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	clientRepo ClientRepository,
	securitySettingRepo secpolicy.SecuritySettingRepository,
	passwordHistoryRepo UserPasswordHistoryRepository,
) ResetPasswordService {
	return &resetPasswordService{
		db:                  db,
		userRepo:            userRepo,
		userTokenRepo:       userTokenRepo,
		clientRepo:          clientRepo,
		securitySettingRepo: securitySettingRepo,
		passwordHistoryRepo: passwordHistoryRepo,
	}
}

var resetHashPassword = security.HashPassword

func (s *resetPasswordService) ResetPassword(ctx context.Context, token, newPassword string, clientID, providerID *string) (*ResetPasswordResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "password.reset")
	defer span.End()

	var user *User
	var userToken *UserToken

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Validate auth client first
		var Client *Client
		var txErr error
		if clientID != nil && providerID != nil {
			Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
		} else {
			Client, txErr = txClientRepo.FindSystem()
		}
		if txErr != nil {
			return apperror.NewInternal("failed to find auth client", txErr)
		}
		if Client == nil {
			return apperror.NewUnauthorized("invalid client credentials")
		}

		// Find the reset token by searching all password reset tokens
		// Note: This is not the most efficient approach, but works with current repository methods
		var foundToken *UserToken

		// We need to find all password reset tokens and check which one matches our token
		// This is a security consideration - we don't want to reveal if a token exists
		allTokens := []UserToken{}
		txErr = tx.Where("token_type = ? AND token = ? AND is_revoked = false", shared.TokenTypePasswordReset, hashUserBearerToken(token)).Find(&allTokens).Error
		if txErr != nil {
			return apperror.NewInternal("failed to find reset token", txErr)
		}

		if len(allTokens) == 0 {
			return apperror.NewUnauthorized("invalid or expired reset token")
		}

		foundToken = &allTokens[0]
		userToken = foundToken

		// Check if token is expired
		if userToken.ExpiresAt != nil && time.Now().After(*userToken.ExpiresAt) {
			return apperror.NewUnauthorized("reset token has expired")
		}

		// Check if token is revoked
		if userToken.IsRevoked {
			return apperror.NewUnauthorized("reset token has been revoked")
		}

		// Find the user
		user, txErr = txUserRepo.FindByID(userToken.UserID)
		if txErr != nil {
			return apperror.NewInternal("failed to find user", txErr)
		}
		if user == nil {
			return apperror.NewNotFound("user not found")
		}

		// Check if user is active
		if user.Status != shared.StatusActive {
			return apperror.NewUnauthorized("user account is not active")
		}

		// Validate password against tenant policy
		var tenantID int64
		if Client.IdentityProvider != nil {
			tenantID = Client.IdentityProvider.TenantID
		}
		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantID)
		if err := security.ValidatePasswordPolicy(newPassword, policy); err != nil {
			return apperror.NewValidation(err.Error())
		}

		// Check password history
		if err := checkPasswordHistory(s.passwordHistoryRepo, user.UserID, policy.HistoryCount, newPassword); err != nil {
			return apperror.NewValidation(err.Error())
		}

		// Hash the new password
		hashedPassword, txErr := resetHashPassword(ctx, []byte(newPassword))
		if txErr != nil {
			return apperror.NewInternal("failed to hash password", txErr)
		}

		now := time.Now()
		// Update user password using the base repository method
		_, txErr = txUserRepo.UpdateByID(user.UserID, map[string]any{
			"password":              string(hashedPassword),
			"force_password_change": false,
			"password_changed_at":   now,
		})
		if txErr != nil {
			return apperror.NewInternal("failed to update password", txErr)
		}

		// Record new hash in history
		secpolicy.RecordPasswordHistory(s.passwordHistoryRepo, user.UserID, policy.HistoryCount, string(hashedPassword))

		// Revoke all sessions so existing logins are invalidated after password change.
		if txErr = txUserTokenRepo.RevokeAllSessionsByUserID(user.UserID); txErr != nil {
			return apperror.NewInternal("failed to revoke sessions on password reset", txErr)
		}

		// Revoke the reset token
		txErr = txUserTokenRepo.RevokeByUUID(userToken.UserTokenUUID)
		if txErr != nil {
			return apperror.NewInternal("failed to revoke reset token", txErr)
		}

		// Revoke all other password reset tokens for this user
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, shared.TokenTypePasswordReset)
		if txErr != nil {
			return apperror.NewInternal("failed to find existing tokens", txErr)
		}
		for _, existingToken := range existingTokens {
			if existingToken.UserTokenUUID != userToken.UserTokenUUID {
				if txErr := txUserTokenRepo.RevokeByUUID(existingToken.UserTokenUUID); txErr != nil {
					return apperror.NewInternal("failed to revoke existing token", txErr)
				}
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "reset password failed")
		// authevent.Log security event for failed password reset
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "password_reset_failure",
			UserID:    token, // Use token as identifier since we might not have user
			Details:   fmt.Sprintf("Password reset failed: %v", err),
			Severity:  "HIGH",
			Timestamp: time.Now(),
		})
		return nil, err
	}

	// authevent.Log successful password reset
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "password_reset_success",
		UserID:    user.UserUUID.String(),
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
		Timestamp: time.Now(),
	})

	// Reset failed login attempts for this user
	security.ResetFailedAttempts(user.Email)

	span.SetStatus(codes.Ok, "")
	return &ResetPasswordResponseDTO{
		Message: "Password has been reset successfully. You can now log in with your new password.",
		Success: true,
	}, nil
}
