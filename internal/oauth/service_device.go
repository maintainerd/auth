package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/email"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/secpolicy"
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
	db                  *gorm.DB
	clientRepo          ClientRepository
	deviceCodeRepo      OAuthDeviceCodeRepository
	userRepo            UserRepository
	userIdentityRepo    UserIdentityRepository
	authEventService    authevent.AuthEventService
	securitySettingRepo secpolicy.SecuritySettingRepository
}

// NewOAuthDeviceService creates a new OAuthDeviceService.
func NewOAuthDeviceService(
	db *gorm.DB,
	clientRepo ClientRepository,
	deviceCodeRepo OAuthDeviceCodeRepository,
	userRepo UserRepository,
	userIdentityRepo UserIdentityRepository,
	authEventService authevent.AuthEventService,
	securitySettingRepo ...secpolicy.SecuritySettingRepository,
) OAuthDeviceService {
	var settings secpolicy.SecuritySettingRepository
	if len(securitySettingRepo) > 0 {
		settings = securitySettingRepo[0]
	}
	return &oauthDeviceService{
		db:                  db,
		clientRepo:          clientRepo,
		deviceCodeRepo:      deviceCodeRepo,
		userRepo:            userRepo,
		userIdentityRepo:    userIdentityRepo,
		authEventService:    authEventService,
		securitySettingRepo: settings,
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

	if oerr := validateClientAllowedScopes(client, req.Scope); oerr != nil {
		span.SetStatus(codes.Error, "scope not allowed")
		return nil, oerr
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

	acr, amr := authContextFromContext(ctx)
	if err := s.deviceCodeRepo.UpdateApproval(record.OAuthDeviceCodeID, userID, acr, amr); err != nil {
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
	providerID := tokenRealm(record.Client)

	issuer := config.AppPublicHostname
	clientIdentifier := resolveClientIdentifier(record.Client)

	accessToken, err := jwt.GenerateAccessTokenWithOptionsContext(
		ctx,
		user.UserUUID.String(),
		record.Scope,
		issuer,
		issuer,
		clientIdentifier,
		providerID,
		deviceAccessTokenOpts(s.securitySettingRepo, record),
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
		ExpiresIn:   oauthAccessTokenExpiresIn(s.securitySettingRepo, record.Client),
		Scope:       record.Scope,
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

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

func authContextFromContext(ctx context.Context) (string, []string) {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil {
		return jwt.ACRLevel1, []string{jwt.AMRPassword}
	}
	acr := claims.ACR
	if acr == "" {
		acr = jwt.ACRLevel1
	}
	amr := claims.AMR
	if len(amr) == 0 {
		amr = []string{jwt.AMRPassword}
	}
	return acr, amr
}

func deviceAccessTokenOpts(repo secpolicy.SecuritySettingRepository, record *OAuthDeviceCode) *jwt.AccessTokenOptions {
	opts := oauthAccessTokenOptions(repo, record.Client)
	acr, amr := persistedAuthContext(record.AuthACR, record.AuthAMR)
	opts.ACR = acr
	opts.AMR = amr
	return opts
}

func persistedAuthContext(acr string, raw []byte) (string, []string) {
	if acr == "" {
		acr = jwt.ACRLevel1
	}
	var amr []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &amr)
	}
	if len(amr) == 0 {
		amr = []string{jwt.AMRPassword}
	}
	return acr, amr
}

func (s *oauthDeviceService) sendDeviceApprovalEmail(ctx context.Context, user *User, client *Client) error {
	if user.Email == "" {
		return nil
	}
	clientName := "an application"
	if client != nil {
		clientName = client.DisplayName
	}

	data := struct {
		ClientName string
		LogoURL    string
	}{
		ClientName: clientName,
		LogoURL:    email.GetLogoURL(ctx, s.db),
	}

	var tenantID int64
	if client != nil {
		tenantID = client.TenantID
	}
	rendered, err := email.RenderTemplate(s.db, "user:device:approved", tenantID, data)
	if err != nil {
		return fmt.Errorf("failed to render device approval email template: %w", err)
	}
	return email.SendEmail(ctx, s.db, email.SendEmailParams{
		To:        user.Email,
		Subject:   rendered.Subject,
		BodyHTML:  rendered.BodyHTML,
		BodyPlain: rendered.BodyPlain,
	})
}
