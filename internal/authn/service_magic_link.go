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
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// MagicLinkTokenTTL is how long a magic-link token remains valid.
const MagicLinkTokenTTL = 15 * time.Minute

// MagicLinkTemplateName is the template name registered in the seeder.
const MagicLinkTemplateName = "user:magic_link"

type MagicLinkService interface {
	SendMagicLink(ctx context.Context, email string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error)
	LoginWithMagicLink(ctx context.Context, token, clientID, providerID string) (*LoginResponseDTO, error)
}

type magicLinkService struct {
	db                   *gorm.DB
	userRepo             UserRepository
	userTokenRepo        UserTokenRepository
	clientRepo           ClientRepository
	userIdentityRepo     UserIdentityRepository
	identityProviderRepo IdentityProviderRepository
	emailTemplateRepo    branding.EmailTemplateRepository
}

func NewMagicLinkService(
	db *gorm.DB,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	clientRepo ClientRepository,
	userIdentityRepo UserIdentityRepository,
	identityProviderRepo IdentityProviderRepository,
	emailTemplateRepo branding.EmailTemplateRepository,
) MagicLinkService {
	return &magicLinkService{
		db:                   db,
		userRepo:             userRepo,
		userTokenRepo:        userTokenRepo,
		clientRepo:           clientRepo,
		userIdentityRepo:     userIdentityRepo,
		identityProviderRepo: identityProviderRepo,
		emailTemplateRepo:    emailTemplateRepo,
	}
}

func (s *magicLinkService) SendMagicLink(ctx context.Context, emailAddr string, clientID, providerID *string, isInternal bool) (*SendMagicLinkResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "magicLink.send")
	defer span.End()

	var user *User
	var Client *Client
	var token string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Resolve auth client (default if not specified). Client + provider context is
		// required at consume-time to issue tokens, so we capture them here.
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
			return apperror.NewInternal("auth client not found", nil)
		}

		// Find user by email, scoped to the client's tenant. Don't reveal whether
		// the address is registered.
		user, txErr = txUserRepo.FindByEmailAndTenantID(emailAddr, clientTenantID(Client))
		if txErr != nil || user == nil {
			user = nil
			return nil
		}

		// Skip if user is inactive — don't reveal status.
		if user.Status != shared.StatusActive {
			user = nil
			return nil
		}

		// Revoke any existing magic-link tokens for this user.
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, shared.TokenTypeMagicLink)
		if txErr != nil {
			return apperror.NewInternal("failed to find existing tokens", txErr)
		}
		for _, t := range existingTokens {
			if txErr := txUserTokenRepo.RevokeByUUID(t.UserTokenUUID); txErr != nil {
				return apperror.NewInternal("failed to revoke existing token", txErr)
			}
		}

		// Generate secure opaque token (32 bytes -> 64 hex chars).
		token = generateSecureToken(32)

		expiresAt := time.Now().Add(MagicLinkTokenTTL)
		if _, txErr := txUserTokenRepo.Create(&UserToken{
			UserID:    user.UserID,
			TokenType: shared.TokenTypeMagicLink,
			Token:     hashUserBearerToken(token),
			ExpiresAt: &expiresAt,
		}); txErr != nil {
			return apperror.NewInternal("failed to create magic link token", txErr)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction failed")
		return nil, err
	}

	response := &SendMagicLinkResponseDTO{
		Message: "If an account with that email exists, we've sent a sign-in link to it.",
		Success: true,
	}

	if user != nil && token != "" && Client != nil {
		if err := s.sendMagicLinkEmail(ctx, user.Email, token, Client, isInternal); err != nil {
			security.LogSecurityEvent(security.SecurityEvent{
				EventType: "magic_link_send_failure",
				UserID:    user.UserUUID.String(),
				Details:   fmt.Sprintf("Failed to send magic link email: %v", err),
				Severity:  "HIGH",
				Timestamp: time.Now(),
			})
		}
	}

	span.SetStatus(codes.Ok, "")
	return response, nil
}

func (s *magicLinkService) LoginWithMagicLink(ctx context.Context, token, clientID, providerID string) (*LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "magicLink.login")
	defer func() { span.End() }()
	startTime := time.Now()

	token = strings.TrimSpace(token)

	var user *User
	var Client *Client
	var userIdentitySub string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdentityProviderRepo := s.identityProviderRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

		// Validate identity provider + client (mirrors LoginPublic).
		identityProvider, txErr := txIdentityProviderRepo.FindByIdentifier(providerID)
		if txErr != nil || identityProvider == nil {
			return apperror.NewUnauthorized("authentication failed")
		}

		Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(clientID, providerID)
		if txErr != nil || Client == nil ||
			Client.Status != shared.StatusActive ||
			Client.Domain == nil || *Client.Domain == "" {
			return apperror.NewUnauthorized("authentication failed")
		}

		// Find an active, non-revoked magic-link token matching the supplied value.
		var matches []UserToken
		if txErr := tx.Where(
			"token_type = ? AND token = ? AND is_revoked = false",
			shared.TokenTypeMagicLink, hashUserBearerToken(token),
		).Find(&matches).Error; txErr != nil {
			return apperror.NewInternal("failed to find magic link token", txErr)
		}
		if len(matches) == 0 {
			return apperror.NewUnauthorized("invalid or expired sign-in link")
		}
		match := &matches[0]

		if match.ExpiresAt != nil && time.Now().After(*match.ExpiresAt) {
			return apperror.NewUnauthorized("sign-in link has expired")
		}

		// Resolve user.
		user, txErr = txUserRepo.FindByID(match.UserID)
		if txErr != nil || user == nil {
			return apperror.NewUnauthorized("authentication failed")
		}
		if Client.IdentityProvider != nil && user.TenantID != Client.IdentityProvider.TenantID {
			return apperror.NewUnauthorized("authentication failed")
		}
		if user.Status != shared.StatusActive {
			return apperror.NewUnauthorized("account is not active")
		}

		// Resolve user identity sub for token issuance.
		userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientID(user.UserID, Client.ClientID)
		if txErr != nil || userIdentity == nil {
			return apperror.NewUnauthorized("authentication failed")
		}
		userIdentitySub = userIdentity.Sub

		// Single-use: revoke this token (and any other outstanding magic-link tokens).
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, shared.TokenTypeMagicLink)
		if txErr != nil {
			return apperror.NewInternal("failed to find existing tokens", txErr)
		}
		for _, t := range existingTokens {
			if txErr := txUserTokenRepo.RevokeByUUID(t.UserTokenUUID); txErr != nil {
				return apperror.NewInternal("failed to revoke magic link token", txErr)
			}
		}

		// Possessing the magic-link implies email ownership; mark verified if not already.
		if !user.IsEmailVerified {
			if _, txErr := txUserRepo.UpdateByID(user.UserID, map[string]any{
				"is_email_verified": true,
			}); txErr != nil {
				return apperror.NewInternal("failed to update user verification status", txErr)
			}
			user.IsEmailVerified = true
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "magic link login failed")
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "magic_link_login_failure",
			ClientID:  clientID,
			Timestamp: startTime,
			Details:   fmt.Sprintf("Magic link login failed: %v", err),
			Severity:  "MEDIUM",
		})
		return nil, err
	}

	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "magic_link_login_success",
		UserID:    user.UserUUID.String(),
		ClientID:  clientID,
		Timestamp: startTime,
		Details:   fmt.Sprintf("Successful magic link login for user %s", user.Username),
	})

	span.SetStatus(codes.Ok, "")
	return s.generateTokenResponse(ctx, userIdentitySub, user, Client)
}

func (s *magicLinkService) sendMagicLinkEmail(ctx context.Context, to, token string, Client *Client, isInternal bool) error {
	var templateEntity *branding.EmailTemplate
	var err error
	templateEntity, err = s.emailTemplateRepo.FindByNameAndTenantID(MagicLinkTemplateName, Client.IdentityProvider.TenantID)
	if err != nil || templateEntity == nil {
		templateEntity, err = s.emailTemplateRepo.FindByName(MagicLinkTemplateName)
	}
	if err != nil {
		return apperror.NewInternal("failed to fetch magic link email template", err)
	}

	// Build a signed API URL that the frontend forwards to the verify endpoint.
	baseURL := fmt.Sprintf("%s/api/v1/magic-link/verify", config.AppPublicHostname)
	signedAPIURL, err := signedurl.GenerateSignedURL(baseURL, map[string]string{
		"token":       token,
		"client_id":   *Client.Identifier,
		"provider_id": Client.IdentityProvider.Identifier,
	}, MagicLinkTokenTTL)
	if err != nil {
		return apperror.NewInternal("failed to create signed URL", err)
	}

	var frontendBaseURL string
	if isInternal {
		frontendBaseURL = config.AppFrontendConsoleHostname + "/magic-link"
	} else {
		frontendBaseURL = config.AppFrontendIdentityHostname + "/magic-link"
	}
	magicLinkURL, err := signedurl.ConvertToFrontendURL(signedAPIURL, frontendBaseURL)
	if err != nil {
		return apperror.NewInternal("failed to convert to frontend URL", err)
	}

	data := struct {
		MagicLinkURL string
		LogoURL      string
	}{
		MagicLinkURL: magicLinkURL,
		LogoURL:      email.GetLogoURL(ctx, s.db),
	}

	tmpl, err := template.New("magic_link_html").Parse(templateEntity.BodyHTML)
	if err != nil {
		return apperror.NewInternal("failed to parse HTML magic link template", err)
	}
	var bodyHTML bytes.Buffer
	if err := tmpl.Execute(&bodyHTML, data); err != nil {
		return apperror.NewInternal("failed to execute HTML magic link template", err)
	}

	var bodyPlainStr string
	if templateEntity.BodyPlain != nil {
		tmplPlain, err := template.New("magic_link_plain").Parse(*templateEntity.BodyPlain)
		if err != nil {
			return apperror.NewInternal("failed to parse plain magic link template", err)
		}
		var bodyPlain bytes.Buffer
		if err := tmplPlain.Execute(&bodyPlain, data); err != nil {
			return apperror.NewInternal("failed to execute plain magic link template", err)
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

func (s *magicLinkService) generateTokenResponse(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	accessToken, idToken, refreshToken, err := generateTokenSetWithContext(ctx, sub, user, client)
	if err != nil {
		return nil, err
	}
	return buildLoginTokenResponse(accessToken, idToken, refreshToken, time.Now().Unix()), nil
}
