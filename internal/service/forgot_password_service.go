package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"time"

	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/util"
	"gorm.io/gorm"
)

type ForgotPasswordService interface {
	SendPasswordResetEmail(email string, clientID, providerID *string, isInternal bool) (*dto.ForgotPasswordResponseDto, error)
}

type forgotPasswordService struct {
	db                *gorm.DB
	userRepo          repository.UserRepository
	userTokenRepo     repository.UserTokenRepository
	authClientRepo    repository.AuthClientRepository
	emailTemplateRepo repository.EmailTemplateRepository
}

func NewForgotPasswordService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
	authClientRepo repository.AuthClientRepository,
	emailTemplateRepo repository.EmailTemplateRepository,
) ForgotPasswordService {
	return &forgotPasswordService{
		db:                db,
		userRepo:          userRepo,
		userTokenRepo:     userTokenRepo,
		authClientRepo:    authClientRepo,
		emailTemplateRepo: emailTemplateRepo,
	}
}

func (s *forgotPasswordService) SendPasswordResetEmail(email string, clientID, providerID *string, isInternal bool) (*dto.ForgotPasswordResponseDto, error) {
	var user *model.User
	var authClient *model.AuthClient
	var resetToken string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txAuthClientRepo := s.authClientRepo.WithTx(tx)

		// Get auth client (default or specified)
		var txErr error
		if clientID != nil && providerID != nil {
			authClient, txErr = txAuthClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
		} else {
			authClient, txErr = txAuthClientRepo.FindDefault()
		}
		if txErr != nil {
			return fmt.Errorf("failed to find auth client: %w", txErr)
		}

		// Get tenant ID from auth client
		tenantID := authClient.IdentityProvider.Tenant.TenantID

		// Find user by email
		user, txErr = txUserRepo.FindByEmail(email, tenantID)
		if txErr != nil {
			// Don't reveal if email exists or not for security
			return nil
		}
		if user == nil {
			// Don't reveal if email exists or not for security
			return nil
		}

		// Check if user is active
		if user.Status != "active" {
			// Don't reveal if user is inactive for security
			return nil
		}

		// Revoke any existing password reset tokens for this user
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, "user:password:reset")
		if txErr != nil {
			return fmt.Errorf("failed to find existing tokens: %w", txErr)
		}
		for _, token := range existingTokens {
			if txErr := txUserTokenRepo.RevokeByUUID(token.UserTokenUUID); txErr != nil {
				return fmt.Errorf("failed to revoke existing token: %w", txErr)
			}
		}

		// Generate secure reset token
		resetToken, txErr = generateSecureToken(32)
		if txErr != nil {
			return fmt.Errorf("failed to generate reset token: %w", txErr)
		}

		// Create password reset token (expires in 1 hour)
		expiresAt := time.Now().Add(1 * time.Hour)
		userToken := &model.UserToken{
			UserID:    user.UserID,
			TokenType: "user:password:reset",
			Token:     resetToken,
			ExpiresAt: &expiresAt,
		}
		_, txErr = txUserTokenRepo.Create(userToken)
		if txErr != nil {
			return fmt.Errorf("failed to create reset token: %w", txErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Always send success response for security (don't reveal if email exists)
	response := &dto.ForgotPasswordResponseDto{
		Message: "If an account with that email exists, we've sent a password reset link to it.",
		Success: true,
	}

	// Only send email if user was found (user will be nil if not found due to security)
	if user != nil {
		// Generate reset URL and send email
		if err := s.sendPasswordResetEmail(user.Email, resetToken, authClient, isInternal); err != nil {
			// Log error but don't reveal it to user for security
			util.LogSecurityEvent(util.SecurityEvent{
				EventType: "password_reset_email_failure",
				UserID:    user.UserUUID.String(),
				Details:   fmt.Sprintf("Failed to send password reset email: %v", err),
				Severity:  "HIGH",
				Timestamp: time.Now(),
			})
		}
	}

	return response, nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *forgotPasswordService) sendPasswordResetEmail(to, resetToken string, authClient *model.AuthClient, isInternal bool) error {
	// Get email template from DB
	templateEntity, err := s.emailTemplateRepo.FindByName("internal:user:password:reset")
	if err != nil {
		return fmt.Errorf("failed to fetch password reset email template: %w", err)
	}

	// Create signed URL for password reset
	baseURL := fmt.Sprintf("%s/api/v1/reset-password", config.AppPublicHostname)
	signedAPIURL, err := util.GenerateSignedURL(baseURL, map[string]string{
		"token":       resetToken,
		"client_id":   *authClient.ClientID,
		"provider_id": authClient.IdentityProvider.Identifier,
	}, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to create signed URL: %w", err)
	}

	// Convert to frontend URL - use different hostname based on request type
	var frontendBaseURL string
	if isInternal {
		frontendBaseURL = config.AuthHostname + "/reset-password"
	} else {
		frontendBaseURL = config.AccountHostname + "/reset-password"
	}
	resetURL, err := util.ConvertToFrontendURL(signedAPIURL, frontendBaseURL)
	if err != nil {
		return fmt.Errorf("failed to convert to frontend URL: %w", err)
	}

	// Prepare data for the template
	data := struct {
		ResetURL string
		LogoURL  string
	}{
		ResetURL: resetURL,
		LogoURL:  config.EmailLogo,
	}

	// Parse HTML template
	tmpl, err := template.New("reset_html").Parse(templateEntity.BodyHTML)
	if err != nil {
		return fmt.Errorf("failed to parse HTML reset template: %w", err)
	}
	var bodyHTML bytes.Buffer
	if err := tmpl.Execute(&bodyHTML, data); err != nil {
		return fmt.Errorf("failed to execute HTML reset template: %w", err)
	}

	// Parse plain-text template if available
	var bodyPlainStr string
	if templateEntity.BodyPlain != nil {
		tmplPlain, err := template.New("reset_plain").Parse(*templateEntity.BodyPlain)
		if err != nil {
			return fmt.Errorf("failed to parse plain reset template: %w", err)
		}
		var bodyPlain bytes.Buffer
		if err := tmplPlain.Execute(&bodyPlain, data); err != nil {
			return fmt.Errorf("failed to execute plain reset template: %w", err)
		}
		bodyPlainStr = bodyPlain.String()
	}

	// Send email
	return util.SendEmail(util.SendEmailParams{
		To:        to,
		Subject:   templateEntity.Subject,
		BodyHTML:  bodyHTML.String(),
		BodyPlain: bodyPlainStr,
	})
}
