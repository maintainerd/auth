package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	deviceCodeTTL      = 15 * time.Minute
	deviceCodeLength   = 32
	devicePollInterval = 5 // seconds
	userCodeLength     = 8
)

// OAuthDeviceService handles the Device Authorization Grant (RFC 8628).
type OAuthDeviceService interface {
	// Authorize processes a device authorization request and returns the
	// device_code, user_code, and verification_uri.
	Authorize(ctx context.Context, req OAuthDeviceAuthorizationRequestDTO, creds OAuthClientCredentials) (*OAuthDeviceAuthorizationResponseDTO, *apperror.OAuthError)

	// VerifyUserCode is called when the authenticated user submits the user_code
	// at the verification URI. Marks the device code as approved.
	VerifyUserCode(ctx context.Context, req OAuthDeviceVerifyRequestDTO, userID int64) *apperror.OAuthError

	// DenyUserCode is called when the user explicitly rejects the request.
	DenyUserCode(ctx context.Context, req OAuthDeviceVerifyRequestDTO, userID int64) *apperror.OAuthError

	// ExchangeToken polls for an access token using a device_code. Returns
	// authorization_pending, slow_down, or a full token set.
	ExchangeToken(ctx context.Context, req OAuthDeviceTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError)
}

type oauthDeviceService struct {
	db               *gorm.DB
	clientRepo       ClientRepository
	deviceCodeRepo   OAuthDeviceCodeRepository
	userRepo         UserRepository
	userIdentityRepo UserIdentityRepository
	authEventService authevent.AuthEventService
}

// NewOAuthDeviceService creates a new OAuthDeviceService.
func NewOAuthDeviceService(
	db *gorm.DB,
	clientRepo ClientRepository,
	deviceCodeRepo OAuthDeviceCodeRepository,
	userRepo UserRepository,
	userIdentityRepo UserIdentityRepository,
	authEventService authevent.AuthEventService,
) OAuthDeviceService {
	return &oauthDeviceService{
		db:               db,
		clientRepo:       clientRepo,
		deviceCodeRepo:   deviceCodeRepo,
		userRepo:         userRepo,
		userIdentityRepo: userIdentityRepo,
		authEventService: authEventService,
	}
}

// Authorize implements OAuthDeviceService.
func (s *oauthDeviceService) Authorize(ctx context.Context, req OAuthDeviceAuthorizationRequestDTO, creds OAuthClientCredentials) (*OAuthDeviceAuthorizationResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_device.authorize")
	defer span.End()
	span.SetAttributes(attribute.String("oauth.client_id", creds.ClientID))

	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		span.SetStatus(codes.Error, "client auth failed")
		return nil, oerr
	}

	if !clientHasGrant(client, GrantTypeDeviceCode) {
		span.SetStatus(codes.Error, "grant not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for device_code grant")
	}

	rawDeviceCode, err := crypto.GenerateRandomString(deviceCodeLength)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	deviceCodeHash := crypto.HashAuthorizationCode(rawDeviceCode)

	userCode, err := generateUserCode()
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	deviceCode := &OAuthDeviceCode{
		DeviceCodeHash: deviceCodeHash,
		UserCode:       userCode,
		ClientID:       client.ClientID,
		TenantID:       client.TenantID,
		Scope:          req.Scope,
		Status:         DeviceCodeStatusPending,
		Interval:       devicePollInterval,
		ExpiresAt:      time.Now().Add(deviceCodeTTL),
	}

	if _, err := s.deviceCodeRepo.Create(deviceCode); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "device code creation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	verificationURI := config.AppPublicHostname + "/device"
	verificationURIComplete := fmt.Sprintf("%s?user_code=%s", verificationURI, userCode)

	span.SetStatus(codes.Ok, "")
	return &OAuthDeviceAuthorizationResponseDTO{
		DeviceCode:              rawDeviceCode,
		UserCode:                userCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		ExpiresIn:               int(deviceCodeTTL.Seconds()),
		Interval:                devicePollInterval,
	}, nil
}

// VerifyUserCode implements OAuthDeviceService.
func (s *oauthDeviceService) VerifyUserCode(ctx context.Context, req OAuthDeviceVerifyRequestDTO, userID int64) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_device.verify_user_code")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	record, err := s.deviceCodeRepo.FindByUserCode(req.UserCode)
	if err != nil {
		span.RecordError(err)
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if record == nil {
		return apperror.NewOAuthInvalidGrant("invalid or expired user_code")
	}
	if record.IsExpired() {
		_ = s.deviceCodeRepo.UpdateStatus(record.OAuthDeviceCodeID, DeviceCodeStatusExpired, nil)
		return apperror.NewOAuthInvalidGrant("user_code has expired")
	}

	if err := s.deviceCodeRepo.UpdateStatus(record.OAuthDeviceCodeID, DeviceCodeStatusApproved, &userID); err != nil {
		span.RecordError(err)
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    record.TenantID,
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthConsent,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Device authorization approved"),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// DenyUserCode implements OAuthDeviceService.
func (s *oauthDeviceService) DenyUserCode(ctx context.Context, req OAuthDeviceVerifyRequestDTO, userID int64) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_device.deny_user_code")
	defer span.End()

	record, err := s.deviceCodeRepo.FindByUserCode(req.UserCode)
	if err != nil {
		span.RecordError(err)
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if record == nil {
		return apperror.NewOAuthInvalidGrant("invalid or expired user_code")
	}

	if err := s.deviceCodeRepo.UpdateStatus(record.OAuthDeviceCodeID, DeviceCodeStatusDenied, nil); err != nil {
		span.RecordError(err)
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    record.TenantID,
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthConsentDeny,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultFailure,
		Description: ptr.Ptr("Device authorization denied by user"),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// ExchangeToken implements OAuthDeviceService (device polling flow).
func (s *oauthDeviceService) ExchangeToken(ctx context.Context, req OAuthDeviceTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_device.exchange_token")
	defer span.End()
	span.SetAttributes(attribute.String("oauth.client_id", creds.ClientID))

	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		span.SetStatus(codes.Error, "client auth failed")
		return nil, oerr
	}

	deviceCodeHash := crypto.HashAuthorizationCode(req.DeviceCode)
	record, err := s.deviceCodeRepo.FindByDeviceCodeHash(deviceCodeHash)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if record == nil {
		return nil, apperror.NewOAuthInvalidGrant("device_code not found")
	}

	// Ensure the device_code belongs to the requesting client.
	if record.ClientID != client.ClientID {
		return nil, apperror.NewOAuthInvalidGrant("device_code does not belong to this client")
	}

	if record.IsExpired() || record.Status == DeviceCodeStatusExpired {
		return nil, &apperror.OAuthError{
			Code:        "expired_token",
			Description: "the device_code has expired; restart the device authorization flow",
			StatusCode:  400,
		}
	}

	// Enforce slow-down: if the client is polling too quickly, return slow_down.
	if record.LastPollAt != nil && time.Since(*record.LastPollAt) < time.Duration(record.Interval)*time.Second {
		_ = s.deviceCodeRepo.UpdateLastPollAt(record.OAuthDeviceCodeID)
		return nil, &apperror.OAuthError{
			Code:        "slow_down",
			Description: "polling too frequently; increase interval by 5 seconds",
			StatusCode:  400,
		}
	}
	_ = s.deviceCodeRepo.UpdateLastPollAt(record.OAuthDeviceCodeID)

	switch record.Status {
	case DeviceCodeStatusPending:
		return nil, &apperror.OAuthError{
			Code:        "authorization_pending",
			Description: "the user has not yet approved the request",
			StatusCode:  400,
		}
	case DeviceCodeStatusDenied:
		return nil, apperror.NewOAuthAccessDenied("the user denied the device authorization request")
	case DeviceCodeStatusApproved:
		// Fall through to token issuance.
	default:
		return nil, apperror.NewOAuthInvalidGrant("unexpected device code status")
	}

	if record.UserID == nil {
		return nil, apperror.NewOAuthServerError("approved device code has no associated user")
	}

	user, err := s.userRepo.FindByID(*record.UserID)
	if err != nil || user == nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Look up the user identity for this client's identity provider.
	providerID := ""
	if record.Client != nil && record.Client.IdentityProvider != nil {
		providerID = record.Client.IdentityProvider.IdentityProviderUUID.String()
	}

	issuer := config.AppPublicHostname
	clientIdentifier := resolveClientIdentifier(record.Client)

	accessToken, err := jwt.GenerateAccessToken(
		user.UserUUID.String(),
		record.Scope,
		issuer,
		issuer,
		clientIdentifier,
		providerID,
	)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	_ = s.sendDeviceApprovalEmail(ctx, user, record.Client)

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    record.TenantID,
		ActorUserID: record.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Device code token issued"),
	})

	span.SetStatus(codes.Ok, "")
	return &OAuthTokenResponseDTO{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(jwt.AccessTokenTTL.Seconds()),
		Scope:       record.Scope,
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

func (s *oauthDeviceService) authenticateClient(creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
	var client Client
	err := s.db.
		Preload("IdentityProvider").
		Preload("ClientURIs").
		Where("identifier = ? AND status = ?", creds.ClientID, shared.StatusActive).
		First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewOAuthInvalidClient("unknown client_id")
		}
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client.SecretHash != nil && *client.SecretHash != "" {
		if !clientSecretMatches(&client, creds.ClientSecret) {
			return nil, apperror.NewOAuthInvalidClient("client authentication failed")
		}
	}
	return &client, nil
}

// generateUserCode returns an 8-character uppercase alphanumeric code in the
// format XXXX-XXXX for easy human entry.
func generateUserCode() (string, error) {
	const charset = "BCDFGHJKLMNPQRSTVWXYZ23456789"
	result := make([]byte, userCodeLength+1) // +1 for separator
	raw, err := crypto.GenerateRandomString(16)
	if err != nil {
		return "", err
	}
	for i := 0; i < userCodeLength+1; i++ {
		if i == 4 {
			result[i] = '-'
			continue
		}
		idx := i
		if i > 4 {
			idx--
		}
		result[i] = charset[int(raw[idx])%len(charset)]
	}
	return string(result), nil
}

func (s *oauthDeviceService) sendDeviceApprovalEmail(ctx context.Context, user *User, client *Client) error {
	if user.Email == "" {
		return nil
	}
	clientName := "an application"
	if client != nil {
		clientName = client.DisplayName
	}
	bodyHTML := fmt.Sprintf(
		`<p>You have successfully authorized <strong>%s</strong> to access your account.</p>
		 <p>If you did not approve this request, please secure your account immediately.</p>`,
		strings.ReplaceAll(clientName, "<", "&lt;"),
	)
	return email.SendEmail(ctx, email.SendEmailParams{
		To:        user.Email,
		Subject:   "Device Authorization Approved",
		BodyHTML:  bodyHTML,
		BodyPlain: fmt.Sprintf("You authorized %s to access your account.", clientName),
	})
}
