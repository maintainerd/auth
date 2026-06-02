package oauth

import (
	"context"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	cibaRequestTTL    = 5 * time.Minute
	cibaRequestLength = 32
	cibaInterval      = 5 // seconds
)

// OAuthCIBAService handles Client-Initiated Backchannel Authentication (CIBA).
type OAuthCIBAService interface {
	// Initiate processes a backchannel authentication request, issues an auth_req_id,
	// and triggers an out-of-band push/notification to the user.
	Initiate(ctx context.Context, req OAuthCIBARequestDTO, creds OAuthClientCredentials) (*OAuthCIBAResponseDTO, *apperror.OAuthError)

	// ApproveRequest marks a CIBA request as approved by the user.
	ApproveRequest(ctx context.Context, authReqID string, userID int64) *apperror.OAuthError

	// DenyRequest marks a CIBA request as denied by the user.
	DenyRequest(ctx context.Context, authReqID string, userID int64) *apperror.OAuthError

	// ExchangeToken polls for a token using an auth_req_id.
	ExchangeToken(ctx context.Context, req OAuthCIBATokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError)
}

type oauthCIBAService struct {
	db               *gorm.DB
	clientRepo       ClientRepository
	cibaRepo         OAuthCIBARequestRepository
	userRepo         UserRepository
	authEventService authevent.AuthEventService
}

// NewOAuthCIBAService creates a new OAuthCIBAService.
func NewOAuthCIBAService(
	db *gorm.DB,
	clientRepo ClientRepository,
	cibaRepo OAuthCIBARequestRepository,
	userRepo UserRepository,
	authEventService authevent.AuthEventService,
) OAuthCIBAService {
	return &oauthCIBAService{
		db:               db,
		clientRepo:       clientRepo,
		cibaRepo:         cibaRepo,
		userRepo:         userRepo,
		authEventService: authEventService,
	}
}

// Initiate implements OAuthCIBAService.
func (s *oauthCIBAService) Initiate(ctx context.Context, req OAuthCIBARequestDTO, creds OAuthClientCredentials) (*OAuthCIBAResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_ciba.initiate")
	defer span.End()
	span.SetAttributes(attribute.String("oauth.client_id", creds.ClientID))

	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		span.SetStatus(codes.Error, "client auth failed")
		return nil, oerr
	}

	if !clientHasGrant(client, GrantTypeCIBA) {
		span.SetStatus(codes.Error, "grant not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for urn:openid:params:grant-type:ciba")
	}

	if oerr := validateClientAllowedScopes(client, req.Scope); oerr != nil {
		span.SetStatus(codes.Error, "scope not allowed")
		return nil, oerr
	}

	// Resolve the user from login_hint (email or phone).
	if req.LoginHint == "" {
		return nil, apperror.NewOAuthInvalidRequest("login_hint is required")
	}

	user, err := s.userRepo.FindByEmail(req.LoginHint)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if user == nil {
		return nil, apperror.NewOAuthInvalidRequest("no user found for the provided login_hint")
	}

	rawAuthReqID, err := crypto.GenerateRandomString(cibaRequestLength)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	authReqIDHash := crypto.HashAuthorizationCode(rawAuthReqID)

	cibaReq := &OAuthCIBARequest{
		AuthReqIDHash: authReqIDHash,
		ClientID:      client.ClientID,
		TenantID:      client.TenantID,
		UserID:        &user.UserID,
		Scope:         req.Scope,
		Status:        CIBAStatusPending,
		Interval:      cibaInterval,
		ExpiresAt:     time.Now().Add(cibaRequestTTL),
	}
	if req.BindingMessage != "" {
		cibaReq.BindingMessage = &req.BindingMessage
	}

	if _, err := s.cibaRepo.Create(cibaReq); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ciba request creation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	_ = s.sendCIBANotificationEmail(ctx, user, client, req.BindingMessage)

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.TenantID,
		ActorUserID: &user.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthAuthorize,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("CIBA authentication initiated"),
	})

	span.SetAttributes(attribute.Int64("user.id", user.UserID))
	span.SetStatus(codes.Ok, "")
	return &OAuthCIBAResponseDTO{
		AuthReqID: rawAuthReqID,
		ExpiresIn: int(cibaRequestTTL.Seconds()),
		Interval:  cibaInterval,
	}, nil
}

// ApproveRequest implements OAuthCIBAService.
func (s *oauthCIBAService) ApproveRequest(ctx context.Context, authReqID string, userID int64) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_ciba.approve_request")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	hash := crypto.HashAuthorizationCode(authReqID)
	record, err := s.cibaRepo.FindByAuthReqIDHash(hash)
	if err != nil {
		span.RecordError(err)
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if record == nil {
		return apperror.NewOAuthInvalidGrant("auth_req_id not found")
	}
	if record.IsExpired() {
		_ = s.cibaRepo.UpdateStatus(record.OAuthCIBARRequestID, CIBAStatusExpired)
		return apperror.NewOAuthInvalidGrant("auth_req_id has expired")
	}

	if err := s.cibaRepo.UpdateApproval(record.OAuthCIBARRequestID, userID); err != nil {
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
		Description: ptr.Ptr("CIBA request approved"),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// DenyRequest implements OAuthCIBAService.
func (s *oauthCIBAService) DenyRequest(ctx context.Context, authReqID string, userID int64) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_ciba.deny_request")
	defer span.End()

	hash := crypto.HashAuthorizationCode(authReqID)
	record, err := s.cibaRepo.FindByAuthReqIDHash(hash)
	if err != nil {
		span.RecordError(err)
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if record == nil {
		return apperror.NewOAuthInvalidGrant("auth_req_id not found")
	}

	if err := s.cibaRepo.UpdateStatus(record.OAuthCIBARRequestID, CIBAStatusDenied); err != nil {
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
		Description: ptr.Ptr("CIBA request denied by user"),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

// ExchangeToken implements OAuthCIBAService (poll mode).
func (s *oauthCIBAService) ExchangeToken(ctx context.Context, req OAuthCIBATokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_ciba.exchange_token")
	defer span.End()
	span.SetAttributes(attribute.String("oauth.client_id", creds.ClientID))

	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		span.SetStatus(codes.Error, "client auth failed")
		return nil, oerr
	}

	hash := crypto.HashAuthorizationCode(req.AuthReqID)
	record, err := s.cibaRepo.FindByAuthReqIDHash(hash)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if record == nil {
		return nil, apperror.NewOAuthInvalidGrant("auth_req_id not found")
	}

	if record.ClientID != client.ClientID {
		return nil, apperror.NewOAuthInvalidGrant("auth_req_id does not belong to this client")
	}

	if record.IsExpired() || record.Status == CIBAStatusExpired {
		return nil, &apperror.OAuthError{
			Code:        "expired_token",
			Description: "the auth_req_id has expired; restart the CIBA flow",
			StatusCode:  400,
		}
	}

	// Enforce slow-down on polling interval.
	if record.LastPollAt != nil && time.Since(*record.LastPollAt) < time.Duration(record.Interval)*time.Second {
		_ = s.cibaRepo.UpdateLastPollAt(record.OAuthCIBARRequestID)
		return nil, &apperror.OAuthError{
			Code:        "slow_down",
			Description: "polling too frequently; increase interval by 5 seconds",
			StatusCode:  400,
		}
	}
	_ = s.cibaRepo.UpdateLastPollAt(record.OAuthCIBARRequestID)

	switch record.Status {
	case CIBAStatusPending:
		return nil, &apperror.OAuthError{
			Code:        "authorization_pending",
			Description: "the user has not yet approved the request",
			StatusCode:  400,
		}
	case CIBAStatusDenied:
		return nil, apperror.NewOAuthAccessDenied("the user denied the CIBA request")
	case CIBAStatusApproved:
		// Fall through to token issuance.
	default:
		return nil, apperror.NewOAuthInvalidGrant("unexpected CIBA status")
	}

	if record.UserID == nil {
		return nil, apperror.NewOAuthServerError("approved CIBA request has no associated user")
	}

	user, err := s.userRepo.FindByID(*record.UserID)
	if err != nil || user == nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	providerID := ""
	if record.Client != nil && record.Client.IdentityProvider != nil {
		providerID = record.Client.IdentityProvider.IdentityProviderUUID.String()
	}

	issuer := config.AppPublicHostname
	clientIdentifier := resolveClientIdentifier(record.Client)

	accessToken, err := jwt.GenerateAccessTokenWithOptions(
		user.UserUUID.String(),
		record.Scope,
		issuer,
		issuer,
		clientIdentifier,
		providerID,
		clientAccessTokenOpts(record.Client),
	)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    record.TenantID,
		ActorUserID: record.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("CIBA token issued"),
	})

	span.SetStatus(codes.Ok, "")
	return &OAuthTokenResponseDTO{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(jwt.AccessTokenTTL.Seconds()),
		Scope:       record.Scope,
	}, nil
}

func (s *oauthCIBAService) sendCIBANotificationEmail(ctx context.Context, user *User, client *Client, bindingMessage string) error {
	if user.Email == "" {
		return nil
	}
	clientName := "an application"
	if client != nil {
		clientName = client.DisplayName
	}
	msg := ""
	if bindingMessage != "" {
		msg = fmt.Sprintf(`<p>Message: <strong>%s</strong></p>`, strings.ReplaceAll(bindingMessage, "<", "&lt;"))
	}
	bodyHTML := fmt.Sprintf(
		`<p><strong>%s</strong> is requesting access to your account.</p>%s
		 <p>Please approve or deny this request in your authenticator app or the login portal.</p>`,
		strings.ReplaceAll(clientName, "<", "&lt;"),
		msg,
	)
	return email.SendEmail(ctx, email.SendEmailParams{
		To:        user.Email,
		Subject:   "Authentication Request",
		BodyHTML:  bodyHTML,
		BodyPlain: fmt.Sprintf("%s is requesting access to your account. Please approve or deny.", clientName),
	})
}
