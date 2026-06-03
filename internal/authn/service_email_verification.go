package authn

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/branding"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// EmailVerificationOTPLength is the length of the verification code sent by email.
const EmailVerificationOTPLength = 6

// EmailVerificationOTPTTL is how long a verification code remains valid.
const EmailVerificationOTPTTL = 1 * time.Hour

// EmailVerificationTemplateName is the template name registered in the seeder.
const EmailVerificationTemplateName = "internal:user:email:verification"

type EmailVerificationService interface {
	SendVerificationEmail(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error)
	VerifyEmail(ctx context.Context, email, otp string) (*VerifyEmailResponseDTO, error)
}

type emailVerificationService struct {
	db                *gorm.DB
	userRepo          UserRepository
	userTokenRepo     UserTokenRepository
	clientRepo        ClientRepository
	emailTemplateRepo branding.EmailTemplateRepository
}

var generateEmailVerificationOTP = crypto.GenerateOTP

func NewEmailVerificationService(
	db *gorm.DB,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	clientRepo ClientRepository,
	emailTemplateRepo branding.EmailTemplateRepository,
) EmailVerificationService {
	return &emailVerificationService{
		db:                db,
		userRepo:          userRepo,
		userTokenRepo:     userTokenRepo,
		clientRepo:        clientRepo,
		emailTemplateRepo: emailTemplateRepo,
	}
}

func (s *emailVerificationService) SendVerificationEmail(ctx context.Context, emailAddr string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailVerification.send")
	defer span.End()

	var user *User
	var otp string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Resolve auth client (default if not specified). We look it up to ensure the
		// caller is operating against a known client, mirroring the forgot-password flow.
		var txErr error
		if clientID != nil && providerID != nil {
			if _, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID); txErr != nil {
				return apperror.NewInternal("failed to find auth client", txErr)
			}
		} else {
			if _, txErr = txClientRepo.FindSystem(); txErr != nil {
				return apperror.NewInternal("failed to find auth client", txErr)
			}
		}

		// Find user by email. Don't reveal whether the address is registered.
		user, txErr = txUserRepo.FindByEmail(emailAddr)
		if txErr != nil {
			return nil
		}
		if user == nil {
			return nil
		}

		// Skip if user is inactive — don't reveal status.
		if user.Status != shared.StatusActive {
			user = nil
			return nil
		}

		// If already verified, no token issued; respond with the same masked success.
		if user.IsEmailVerified {
			user = nil
			return nil
		}

		// Revoke any existing email-verification tokens for this user.
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, shared.TokenTypeEmailVerification)
		if txErr != nil {
			return apperror.NewInternal("failed to find existing tokens", txErr)
		}
		for _, t := range existingTokens {
			if txErr := txUserTokenRepo.RevokeByUUID(t.UserTokenUUID); txErr != nil {
				return apperror.NewInternal("failed to revoke existing token", txErr)
			}
		}

		// Generate a fresh OTP with a bounded TTL.
		otp, txErr = generateEmailVerificationOTP(EmailVerificationOTPLength)
		if txErr != nil {
			return apperror.NewInternal("failed to generate verification code", txErr)
		}
		otpHash := crypto.HashAuthorizationCode(otp)

		expiresAt := time.Now().Add(EmailVerificationOTPTTL)
		if _, txErr := txUserTokenRepo.Create(&UserToken{
			UserID:    user.UserID,
			TokenType: shared.TokenTypeEmailVerification,
			Token:     otpHash,
			ExpiresAt: &expiresAt,
		}); txErr != nil {
			return apperror.NewInternal("failed to create verification token", txErr)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction failed")
		return nil, err
	}

	response := &SendEmailVerificationResponseDTO{
		Message: "If an account with that email exists and is not yet verified, we've sent a verification code to it.",
		Success: true,
	}

	if user != nil && otp != "" {
		if err := s.sendVerificationEmail(ctx, user.Email, otp); err != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "email_verification_send_failure",
				UserID:    user.UserUUID.String(),
				Details:   fmt.Sprintf("Failed to send verification email: %v", err),
				Severity:  "HIGH",
				Timestamp: time.Now(),
			})
		}
	}

	span.SetStatus(codes.Ok, "")
	return response, nil
}

func (s *emailVerificationService) VerifyEmail(ctx context.Context, emailAddr, otp string) (*VerifyEmailResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailVerification.verify")
	defer span.End()

	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))
	otp = strings.TrimSpace(otp)

	var user *User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)

		// Resolve user by email. Use a generic error message to avoid email enumeration.
		var txErr error
		user, txErr = txUserRepo.FindByEmail(emailAddr)
		if txErr != nil {
			return apperror.NewInternal("failed to find user", txErr)
		}
		if user == nil {
			return apperror.NewUnauthorized("invalid or expired verification code")
		}

		if user.Status != shared.StatusActive {
			return apperror.NewUnauthorized("user account is not active")
		}

		if user.IsEmailVerified {
			// Idempotent: already verified is treated as success at the handler layer.
			return nil
		}

		// Find an active, non-revoked verification token matching the OTP hash.
		otpHash := crypto.HashAuthorizationCode(otp)
		var match *UserToken
		var matches []UserToken
		if txErr := tx.Where(
			"user_id = ? AND token_type = ? AND token = ? AND is_revoked = false",
			user.UserID, shared.TokenTypeEmailVerification, otpHash,
		).Find(&matches).Error; txErr != nil {
			return apperror.NewInternal("failed to find verification token", txErr)
		}
		if len(matches) > 0 {
			match = &matches[0]
		}
		if match == nil {
			return apperror.NewUnauthorized("invalid or expired verification code")
		}

		if match.ExpiresAt != nil && time.Now().After(*match.ExpiresAt) {
			return apperror.NewUnauthorized("verification code has expired")
		}

		// Mark email verified.
		if _, txErr := txUserRepo.UpdateByID(user.UserID, map[string]any{
			"is_email_verified": true,
		}); txErr != nil {
			return apperror.NewInternal("failed to update user verification status", txErr)
		}

		// Revoke all email-verification tokens for the user (single-use).
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, shared.TokenTypeEmailVerification)
		if txErr != nil {
			return apperror.NewInternal("failed to find existing tokens", txErr)
		}
		for _, t := range existingTokens {
			if txErr := txUserTokenRepo.RevokeByUUID(t.UserTokenUUID); txErr != nil {
				return apperror.NewInternal("failed to revoke verification token", txErr)
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "verification failed")
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_failure",
			UserID:    emailAddr,
			Details:   fmt.Sprintf("Email verification failed: %v", err),
			Severity:  "MEDIUM",
			Timestamp: time.Now(),
		})
		return nil, err
	}

	if user != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "email_verification_success",
			UserID:    user.UserUUID.String(),
			Details:   "Email verified successfully",
			Severity:  "INFO",
			Timestamp: time.Now(),
		})
	}

	span.SetStatus(codes.Ok, "")
	return &VerifyEmailResponseDTO{
		Message: "Your email has been verified successfully.",
		Success: true,
	}, nil
}

func (s *emailVerificationService) sendVerificationEmail(ctx context.Context, to, otp string) error {
	templateEntity, err := s.emailTemplateRepo.FindByName(EmailVerificationTemplateName)
	if err != nil {
		return apperror.NewInternal("failed to fetch verification email template", err)
	}

	data := struct {
		OTP     string
		LogoURL string
	}{
		OTP:     otp,
		LogoURL: config.EmailLogo,
	}

	tmpl, err := template.New("verify_html").Parse(templateEntity.BodyHTML)
	if err != nil {
		return apperror.NewInternal("failed to parse HTML verification template", err)
	}
	var bodyHTML bytes.Buffer
	if err := tmpl.Execute(&bodyHTML, data); err != nil {
		return apperror.NewInternal("failed to execute HTML verification template", err)
	}

	var bodyPlainStr string
	if templateEntity.BodyPlain != nil {
		tmplPlain, err := template.New("verify_plain").Parse(*templateEntity.BodyPlain)
		if err != nil {
			return apperror.NewInternal("failed to parse plain verification template", err)
		}
		var bodyPlain bytes.Buffer
		if err := tmplPlain.Execute(&bodyPlain, data); err != nil {
			return apperror.NewInternal("failed to execute plain verification template", err)
		}
		bodyPlainStr = bodyPlain.String()
	}

	return email.SendEmail(ctx, email.SendEmailParams{
		To:        to,
		Subject:   templateEntity.Subject,
		BodyHTML:  bodyHTML.String(),
		BodyPlain: bodyPlainStr,
	})
}
