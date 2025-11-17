package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ResetPasswordService interface {
	ResetPassword(token, newPassword string, clientID, providerID *string) (*dto.ResetPasswordResponseDto, error)
}

type resetPasswordService struct {
	db             *gorm.DB
	userRepo       repository.UserRepository
	userTokenRepo  repository.UserTokenRepository
	authClientRepo repository.AuthClientRepository
}

func NewResetPasswordService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
	authClientRepo repository.AuthClientRepository,
) ResetPasswordService {
	return &resetPasswordService{
		db:             db,
		userRepo:       userRepo,
		userTokenRepo:  userTokenRepo,
		authClientRepo: authClientRepo,
	}
}

func (s *resetPasswordService) ResetPassword(token, newPassword string, clientID, providerID *string) (*dto.ResetPasswordResponseDto, error) {
	var user *model.User
	var userToken *model.UserToken

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Validate auth client first
		var authClient *model.AuthClient
		var txErr error
		if clientID != nil && providerID != nil {
			authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
		} else {
			authClient, txErr = txAuthClientRepo.FindDefault()
		}
		if txErr != nil {
			return fmt.Errorf("failed to find auth client: %w", txErr)
		}
		if authClient == nil {
			return errors.New("invalid client credentials")
		}

		// Find the reset token by searching all password reset tokens
		// Note: This is not the most efficient approach, but works with current repository methods
		var foundToken *model.UserToken

		// We need to find all password reset tokens and check which one matches our token
		// This is a security consideration - we don't want to reveal if a token exists
		allTokens := []model.UserToken{}
		txErr = tx.Where("token_type = ? AND token = ? AND is_revoked = false", "user:password:reset", token).Find(&allTokens).Error
		if txErr != nil {
			return fmt.Errorf("failed to find reset token: %w", txErr)
		}

		if len(allTokens) == 0 {
			return errors.New("invalid or expired reset token")
		}

		foundToken = &allTokens[0]
		userToken = foundToken

		// Check if token is expired
		if userToken.ExpiresAt != nil && time.Now().After(*userToken.ExpiresAt) {
			return errors.New("reset token has expired")
		}

		// Check if token is revoked
		if userToken.IsRevoked {
			return errors.New("reset token has been revoked")
		}

		// Find the user
		user, txErr = txUserRepo.FindByID(userToken.UserID)
		if txErr != nil {
			return fmt.Errorf("failed to find user: %w", txErr)
		}
		if user == nil {
			return errors.New("user not found")
		}

		// Check if user is active
		if !user.IsActive {
			return errors.New("user account is not active")
		}

		// Validate password strength
		if err := util.ValidatePasswordStrength(newPassword); err != nil {
			return fmt.Errorf("password validation failed: %w", err)
		}

		// Hash the new password
		hashedPassword, txErr := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if txErr != nil {
			return fmt.Errorf("failed to hash password: %w", txErr)
		}

		// Update user password using the base repository method
		_, txErr = txUserRepo.UpdateByID(user.UserID, map[string]interface{}{
			"password": string(hashedPassword),
		})
		if txErr != nil {
			return fmt.Errorf("failed to update password: %w", txErr)
		}

		// Revoke the reset token
		txErr = txUserTokenRepo.RevokeByUUID(userToken.UserTokenUUID)
		if txErr != nil {
			return fmt.Errorf("failed to revoke reset token: %w", txErr)
		}

		// Revoke all other password reset tokens for this user
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, "user:password:reset")
		if txErr != nil {
			return fmt.Errorf("failed to find existing tokens: %w", txErr)
		}
		for _, existingToken := range existingTokens {
			if existingToken.UserTokenUUID != userToken.UserTokenUUID {
				if txErr := txUserTokenRepo.RevokeByUUID(existingToken.UserTokenUUID); txErr != nil {
					return fmt.Errorf("failed to revoke existing token: %w", txErr)
				}
			}
		}

		return nil
	})

	if err != nil {
		// Log security event for failed password reset
		util.LogSecurityEvent(util.SecurityEvent{
			EventType: "password_reset_failure",
			UserID:    token, // Use token as identifier since we might not have user
			Details:   fmt.Sprintf("Password reset failed: %v", err),
			Severity:  "HIGH",
			Timestamp: time.Now(),
		})
		return nil, err
	}

	// Log successful password reset
	util.LogSecurityEvent(util.SecurityEvent{
		EventType: "password_reset_success",
		UserID:    user.UserUUID.String(),
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
		Timestamp: time.Now(),
	})

	// Reset failed login attempts for this user
	util.ResetFailedAttempts(user.Email)

	return &dto.ResetPasswordResponseDto{
		Message: "Password has been reset successfully. You can now log in with your new password.",
		Success: true,
	}, nil
}
