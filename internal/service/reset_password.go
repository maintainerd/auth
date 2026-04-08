package service

import (
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/security"
	"gorm.io/gorm"
	"github.com/maintainerd/auth/internal/apperror"
)

type ResetPasswordService interface {
	ResetPassword(token, newPassword string, clientID, providerID *string) (*dto.ResetPasswordResponseDTO, error)
}

type resetPasswordService struct {
	db            *gorm.DB
	userRepo      repository.UserRepository
	userTokenRepo repository.UserTokenRepository
	clientRepo    repository.ClientRepository
}

func NewResetPasswordService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
	clientRepo repository.ClientRepository,
) ResetPasswordService {
	return &resetPasswordService{
		db:            db,
		userRepo:      userRepo,
		userTokenRepo: userTokenRepo,
		clientRepo:    clientRepo,
	}
}

func (s *resetPasswordService) ResetPassword(token, newPassword string, clientID, providerID *string) (*dto.ResetPasswordResponseDTO, error) {
	var user *model.User
	var userToken *model.UserToken

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Validate auth client first
		var Client *model.Client
		var txErr error
		if clientID != nil && providerID != nil {
			Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
		} else {
			Client, txErr = txClientRepo.FindDefault()
		}
		if txErr != nil {
			return apperror.NewInternal("failed to find auth client", txErr)
		}
		if Client == nil {
			return apperror.NewUnauthorized("invalid client credentials")
		}

		// Find the reset token by searching all password reset tokens
		// Note: This is not the most efficient approach, but works with current repository methods
		var foundToken *model.UserToken

		// We need to find all password reset tokens and check which one matches our token
		// This is a security consideration - we don't want to reveal if a token exists
		allTokens := []model.UserToken{}
		txErr = tx.Where("token_type = ? AND token = ? AND is_revoked = false", model.TokenTypePasswordReset, token).Find(&allTokens).Error
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
		if user.Status != model.StatusActive {
			return apperror.NewUnauthorized("user account is not active")
		}

		// Validate password strength
		if err := security.ValidatePasswordStrength(newPassword); err != nil {
			return apperror.NewInternal("password validation failed", err)
		}

		// Hash the new password
		hashedPassword, txErr := security.HashPassword([]byte(newPassword))
		if txErr != nil {
			return apperror.NewInternal("failed to hash password", txErr)
		}

		// Update user password using the base repository method
		_, txErr = txUserRepo.UpdateByID(user.UserID, map[string]any{
			"password": string(hashedPassword),
		})
		if txErr != nil {
			return apperror.NewInternal("failed to update password", txErr)
		}

		// Revoke the reset token
		txErr = txUserTokenRepo.RevokeByUUID(userToken.UserTokenUUID)
		if txErr != nil {
			return apperror.NewInternal("failed to revoke reset token", txErr)
		}

		// Revoke all other password reset tokens for this user
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, model.TokenTypePasswordReset)
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
		// Log security event for failed password reset
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "password_reset_failure",
			UserID:    token, // Use token as identifier since we might not have user
			Details:   fmt.Sprintf("Password reset failed: %v", err),
			Severity:  "HIGH",
			Timestamp: time.Now(),
		})
		return nil, err
	}

	// Log successful password reset
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "password_reset_success",
		UserID:    user.UserUUID.String(),
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
		Timestamp: time.Now(),
	})

	// Reset failed login attempts for this user
	security.ResetFailedAttempts(user.Email)

	return &dto.ResetPasswordResponseDTO{
		Message: "Password has been reset successfully. You can now log in with your new password.",
		Success: true,
	}, nil
}
