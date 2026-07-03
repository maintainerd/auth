package authn

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/email"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// EmailVerificationOTPLength is the length of the verification code sent by email.
const EmailVerificationOTPLength = 6

// EmailVerificationOTPTTL is how long a verification code remains valid.
const EmailVerificationOTPTTL = 1 * time.Hour

// EmailVerificationTemplateName is the template name registered in the seeder.
const EmailVerificationTemplateName = "user:email:verification"

type EmailVerificationService interface {
	SendVerificationEmail(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error)
	VerifyEmail(ctx context.Context, email, otp string, authContext ...*string) (*VerifyEmailResponseDTO, error)
}

type emailVerificationService struct {
	db                  *gorm.DB
	userRepo            UserRepository
	userTokenRepo       UserTokenRepository
	clientRepo          ClientRepository
	emailTemplateRepo   branding.EmailTemplateRepository
	userIdentityRepo    UserIdentityRepository
	cacheInvalidator    cache.Invalidator
	securitySettingRepo secpolicy.SecuritySettingRepository
}

var generateEmailVerificationOTP = crypto.GenerateOTP

func NewEmailVerificationService(
	db *gorm.DB,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	clientRepo ClientRepository,
	emailTemplateRepo branding.EmailTemplateRepository,
	userIdentityRepo UserIdentityRepository,
	cacheInvalidator cache.Invalidator,
	securitySettingRepo ...secpolicy.SecuritySettingRepository,
) EmailVerificationService {
	var settings secpolicy.SecuritySettingRepository
	if len(securitySettingRepo) > 0 {
		settings = securitySettingRepo[0]
	}
	if cacheInvalidator == nil {
		cacheInvalidator = cache.NopInvalidator{}
	}
	return &emailVerificationService{
		db:                  db,
		userRepo:            userRepo,
		userTokenRepo:       userTokenRepo,
		clientRepo:          clientRepo,
		emailTemplateRepo:   emailTemplateRepo,
		userIdentityRepo:    userIdentityRepo,
		cacheInvalidator:    cacheInvalidator,
		securitySettingRepo: settings,
	}
}

func (s *emailVerificationService) SendVerificationEmail(ctx context.Context, emailAddr string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailVerification.send")
	defer span.End()

	var user *User
	var otp string
	var authClient *Client

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Resolve auth client (default if not specified). We look it up to ensure the
		// caller is operating against a known client, mirroring the forgot-password flow.
		var txErr error
		if publicAuthSurfaceFromContext(ctx) {
			authClient, txErr = resolvePublicClient(txClientRepo, clientID, providerID)
		} else if clientID != nil && providerID != nil {
			if authClient, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *providerID); txErr != nil {
				return apperror.NewInternal("failed to find auth client", txErr)
			}
		} else if clientID != nil {
			authClient, txErr = txClientRepo.FindByIdentifier(*clientID)
		} else if providerID != nil {
			authClient, txErr = resolveClient(txClientRepo, nil, providerID)
		} else {
			if authClient, txErr = txClientRepo.FindSystem(); txErr != nil {
				return apperror.NewInternal("failed to find auth client", txErr)
			}
		}
		if txErr != nil {
			return apperror.NewInternal("failed to find auth client", txErr)
		}

		if authClient == nil {
			return nil
		}
		// Find user by email, scoped to the client's tenant. Don't reveal whether
		// the address is registered.
		user, txErr = txUserRepo.FindByEmailAndTenantID(emailAddr, clientTenantID(authClient))
		if txErr != nil {
			return nil
		}
		if user == nil {
			return nil
		}

		if user.IsEmailVerified {
			return nil
		}

		// Skip suspended/inactive users but allow pending accounts to complete
		// their initial email verification.
		if user.Status != shared.StatusActive && user.Status != shared.StatusPending {
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

		ttl := EmailVerificationOTPTTL
		regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, clientTenantID(authClient))
		if regPolicy.VerificationTokenTTLHours > 0 {
			ttl = time.Duration(regPolicy.VerificationTokenTTLHours) * time.Hour
		}
		expiresAt := time.Now().Add(ttl)
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

func (s *emailVerificationService) VerifyEmail(ctx context.Context, emailAddr, otp string, authContext ...*string) (*VerifyEmailResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailVerification.verify")
	defer span.End()

	emailAddr = strings.TrimSpace(strings.ToLower(emailAddr))
	otp = strings.TrimSpace(otp)

	var user *User

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		expectedTenantID := int64(0)
		if len(authContext) >= 2 {
			txClientRepo := s.clientRepo.WithTx(tx)
			var authClient *Client
			var resolveErr error
			if publicAuthSurfaceFromContext(ctx) {
				authClient, resolveErr = resolvePublicClient(txClientRepo, authContext[0], authContext[1])
			} else {
				authClient, resolveErr = resolveClient(txClientRepo, authContext[0], authContext[1])
			}
			if resolveErr != nil {
				return apperror.NewInternal("failed to find auth client", resolveErr)
			}
			if authClient == nil || clientTenantID(authClient) == 0 {
				return apperror.NewUnauthorized("invalid or expired verification code")
			}
			expectedTenantID = clientTenantID(authClient)
		}

		otpHash := crypto.HashAuthorizationCode(otp)

		// Resolve the user from the verification token itself rather than a global
		// email lookup. The OTP binds to a single user (hence a single tenant), so
		// this avoids resolving an arbitrary user when the same email exists in more
		// than one tenant (cross-tenant bug). The supplied email must match the
		// token's user. The endpoint stays self-contained (no client_id needed).
		var candidateTokens []UserToken
		if txErr := tx.Where(
			"token_type = ? AND token = ? AND is_revoked = false",
			shared.TokenTypeEmailVerification, otpHash,
		).Find(&candidateTokens).Error; txErr != nil {
			return apperror.NewInternal("failed to find verification token", txErr)
		}
		var matchedToken *UserToken
		for i := range candidateTokens {
			candidate, lookupErr := txUserRepo.FindByID(candidateTokens[i].UserID)
			if lookupErr != nil {
				return apperror.NewInternal("failed to find user", lookupErr)
			}
			if candidate != nil &&
				(expectedTenantID == 0 || candidate.TenantID == expectedTenantID) &&
				strings.EqualFold(strings.TrimSpace(candidate.Email), emailAddr) {
				// A six-digit OTP can collide. Never pick an arbitrary tenant when
				// the same email+OTP pair is simultaneously valid more than once.
				if user != nil {
					return apperror.NewUnauthorized("invalid or expired verification code")
				}
				user = candidate
				matchedToken = &candidateTokens[i]
			}
		}
		if user == nil {
			return apperror.NewUnauthorized("invalid or expired verification code")
		}

		if user.Status != shared.StatusActive && user.Status != shared.StatusPending {
			return apperror.NewUnauthorized("user account is not active")
		}

		if user.IsEmailVerified {
			return nil
		}

		if matchedToken == nil {
			return apperror.NewUnauthorized("invalid or expired verification code")
		}

		if matchedToken.ExpiresAt != nil && time.Now().After(*matchedToken.ExpiresAt) {
			return apperror.NewUnauthorized("verification code has expired")
		}

		if user.IsEmailVerified {
			return nil
		}

		// Mark email verified.
		if _, txErr := txUserRepo.UpdateByID(user.UserID, map[string]any{
			"is_email_verified": true,
			"status":            shared.StatusActive,
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
		// Clear the cached user context so /account (and downstream auth checks)
		// immediately reflect the now-verified state instead of the stale value
		// captured at registration. Without this the user keeps getting routed
		// back to email verification until the cache TTL expires.
		s.invalidateUserContextCache(ctx, user.UserID)

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

// invalidateUserContextCache clears every cached user-context entry for the
// user's identities (one per sub), so middleware-loaded auth state reflects the
// latest user fields after a mutation. Best-effort: failures must not block the
// verification response.
func (s *emailVerificationService) invalidateUserContextCache(ctx context.Context, userID int64) {
	if s.userIdentityRepo == nil || s.cacheInvalidator == nil {
		return
	}
	identities, err := s.userIdentityRepo.FindByUserID(userID)
	if err != nil {
		return
	}
	seen := make(map[string]struct{})
	for _, id := range identities {
		if id.Sub == "" {
			continue
		}
		if _, ok := seen[id.Sub]; ok {
			continue
		}
		seen[id.Sub] = struct{}{}
		s.cacheInvalidator.InvalidateUserAll(ctx, id.Sub)
	}
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
		LogoURL: email.GetLogoURL(ctx, s.db),
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

	return email.SendEmail(ctx, s.db, email.SendEmailParams{
		To:        to,
		Subject:   templateEntity.Subject,
		BodyHTML:  bodyHTML.String(),
		BodyPlain: bodyPlainStr,
	})
}
