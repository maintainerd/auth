package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"time"

	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/email"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/security"
	"github.com/maintainerd/auth/internal/signedurl"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type ForgotPasswordService interface {
	SendPasswordResetEmail(ctx context.Context, email string, clientID, providerID *string, isInternal bool) (*dto.ForgotPasswordResponseDTO, error)
}

type forgotPasswordService struct {
	db                *gorm.DB
	userRepo          repository.UserRepository
	userTokenRepo     repository.UserTokenRepository
	clientRepo        repository.ClientRepository
	emailTemplateRepo repository.EmailTemplateRepository
}

func NewForgotPasswordService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	userTokenRepo repository.UserTokenRepository,
	clientRepo repository.ClientRepository,
	emailTemplateRepo repository.EmailTemplateRepository,
) ForgotPasswordService {
	return &forgotPasswordService{
		db:                db,
		userRepo:          userRepo,
		userTokenRepo:     userTokenRepo,
		clientRepo:        clientRepo,
		emailTemplateRepo: emailTemplateRepo,
	}
}

func (s *forgotPasswordService) SendPasswordResetEmail(ctx context.Context, email string, clientID, providerID *string, isInternal bool) (*dto.ForgotPasswordResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "forgotPassword.sendResetEmail")
	defer span.End()

	var user *model.User
	var Client *model.Client
	var resetToken string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Get auth client (default or specified)
		var txErr error
		if clientID != nil && providerID != nil {
			Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID)
		} else {
			Client, txErr = txClientRepo.FindDefault()
		}
		if txErr != nil {
			return apperror.NewInternal("failed to find auth client", txErr)
		}

		// Find user by email
		user, txErr = txUserRepo.FindByEmail(email)
		if txErr != nil {
			// Don't reveal if email exists or not for security
			return nil
		}
		if user == nil {
			// Don't reveal if email exists or not for security
			return nil
		}

		// Check if user is active
		if user.Status != model.StatusActive {
			// Don't reveal if user is inactive for security
			return nil
		}

		// Revoke any existing password reset tokens for this user
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, model.TokenTypePasswordReset)
		if txErr != nil {
			return apperror.NewInternal("failed to find existing tokens", txErr)
		}
		for _, token := range existingTokens {
			if txErr := txUserTokenRepo.RevokeByUUID(token.UserTokenUUID); txErr != nil {
				return apperror.NewInternal("failed to revoke existing token", txErr)
			}
		}

		// Generate secure reset token
		resetToken = generateSecureToken(32)

		// Create password reset token (expires in 1 hour)
		expiresAt := time.Now().Add(1 * time.Hour)
		userToken := &model.UserToken{
			UserID:    user.UserID,
			TokenType: model.TokenTypePasswordReset,
			Token:     resetToken,
			ExpiresAt: &expiresAt,
		}
		_, txErr = txUserTokenRepo.Create(userToken)
		if txErr != nil {
			return apperror.NewInternal("failed to create reset token", txErr)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction failed")
		return nil, err
	}

	// Always send success response for security (don't reveal if email exists)
	response := &dto.ForgotPasswordResponseDTO{
		Message: "If an account with that email exists, we've sent a password reset link to it.",
		Success: true,
	}

	// Only send email if user was found (user will be nil if not found due to security)
	if user != nil {
		// Generate reset URL and send email
		if err := s.sendPasswordResetEmail(ctx, user.Email, resetToken, Client, isInternal); err != nil {
			// Log error but don't reveal it to user for security
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "password_reset_email_failure",
				UserID:    user.UserUUID.String(),
				Details:   fmt.Sprintf("Failed to send password reset email: %v", err),
				Severity:  "HIGH",
				Timestamp: time.Now(),
			})
		}
	}

	span.SetStatus(codes.Ok, "")
	return response, nil
}

// generateSecureToken generates a cryptographically secure random token.
// rand.Read always succeeds in Go 1.24+ (see go.dev/issue/66821).
func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *forgotPasswordService) sendPasswordResetEmail(ctx context.Context, to, resetToken string, Client *model.Client, isInternal bool) error {
	// Get email template from DB
	templateEntity, err := s.emailTemplateRepo.FindByName("internal:user:password:reset")
	if err != nil {
		return apperror.NewInternal("failed to fetch password reset email template", err)
	}

	// Create signed URL for password reset
	baseURL := fmt.Sprintf("%s/api/v1/reset-password", config.AppPublicHostname)
	signedAPIURL, err := signedurl.GenerateSignedURL(baseURL, map[string]string{
		"token":       resetToken,
		"client_id":   *Client.Identifier,
		"provider_id": Client.IdentityProvider.Identifier,
	}, 1*time.Hour)
	if err != nil {
		return apperror.NewInternal("failed to create signed URL", err)
	}

	// Convert to frontend URL - use different hostname based on request type
	var frontendBaseURL string
	if isInternal {
		frontendBaseURL = config.AuthHostname + "/reset-password"
	} else {
		frontendBaseURL = config.AccountHostname + "/reset-password"
	}
	resetURL, err := signedurl.ConvertToFrontendURL(signedAPIURL, frontendBaseURL)
	if err != nil {
		return apperror.NewInternal("failed to convert to frontend URL", err)
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
		return apperror.NewInternal("failed to parse HTML reset template", err)
	}
	var bodyHTML bytes.Buffer
	if err := tmpl.Execute(&bodyHTML, data); err != nil {
		return apperror.NewInternal("failed to execute HTML reset template", err)
	}

	// Parse plain-text template if available
	var bodyPlainStr string
	if templateEntity.BodyPlain != nil {
		tmplPlain, err := template.New("reset_plain").Parse(*templateEntity.BodyPlain)
		if err != nil {
			return apperror.NewInternal("failed to parse plain reset template", err)
		}
		var bodyPlain bytes.Buffer
		if err := tmplPlain.Execute(&bodyPlain, data); err != nil {
			return apperror.NewInternal("failed to execute plain reset template", err)
		}
		bodyPlainStr = bodyPlain.String()
	}

	// Send email
	return email.SendEmail(ctx, email.SendEmailParams{
		To:        to,
		Subject:   templateEntity.Subject,
		BodyHTML:  bodyHTML.String(),
		BodyPlain: bodyPlainStr,
	})
}
