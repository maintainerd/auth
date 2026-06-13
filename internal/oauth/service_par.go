package oauth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	parRequestTTL    = 90 * time.Second
	parRequestLength = 32
	// parURIPrefix is the request_uri prefix per RFC 9126 §2.2.
	parURIPrefix = "urn:ietf:params:oauth:request-uri:"
)

// OAuthPARService handles Pushed Authorization Requests (RFC 9126).
type OAuthPARService interface {
	// Push validates the client and stores the authorization request parameters.
	// Returns a request_uri and its TTL in seconds.
	Push(ctx context.Context, req OAuthPARRequestDTO, creds OAuthClientCredentials) (*OAuthPARResponseDTO, *apperror.OAuthError)

	// ConsumeRequestURI looks up and marks-as-used a PAR request_uri. Returns
	// the stored authorization parameters so the authorize endpoint can proceed.
	ConsumeRequestURI(ctx context.Context, requestURI string) (*OAuthAuthorizeRequestDTO, *apperror.OAuthError)
}

type oauthPARService struct {
	db               *gorm.DB
	clientRepo       ClientRepository
	clientURIRepo    ClientURIRepository
	parRepo          OAuthPARRequestRepository
	authEventService authevent.AuthEventService
	securitySettingRepo secpolicy.SecuritySettingRepository
}

// NewOAuthPARService creates a new OAuthPARService.
func NewOAuthPARService(
	db *gorm.DB,
	clientRepo ClientRepository,
	clientURIRepo ClientURIRepository,
	parRepo OAuthPARRequestRepository,
	authEventService authevent.AuthEventService,
	securitySettingRepo ...secpolicy.SecuritySettingRepository,
) OAuthPARService {
	var settings secpolicy.SecuritySettingRepository
	if len(securitySettingRepo) > 0 {
		settings = securitySettingRepo[0]
	}
	return &oauthPARService{
		db:               db,
		clientRepo:       clientRepo,
		clientURIRepo:    clientURIRepo,
		parRepo:          parRepo,
		authEventService: authEventService,
		securitySettingRepo: settings,
	}
}

// Push implements OAuthPARService.
func (s *oauthPARService) Push(ctx context.Context, req OAuthPARRequestDTO, creds OAuthClientCredentials) (*OAuthPARResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_par.push")
	defer span.End()
	span.SetAttributes(
		attribute.String("oauth.client_id", creds.ClientID),
		attribute.String("oauth.response_type", req.ResponseType),
	)

	client, oerr := s.resolveAndAuthenticateClient(creds)
	if oerr != nil {
		span.SetStatus(codes.Error, "client auth failed")
		return nil, oerr
	}

	// Validate that the client supports authorization_code.
	if !clientHasGrant(client, GrantTypeAuthorizationCode) {
		span.SetStatus(codes.Error, "grant not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for authorization_code grant")
	}

	if oerr := validateClientAllowedScopes(client, req.Scope); oerr != nil {
		span.SetStatus(codes.Error, "scope not allowed")
		return nil, oerr
	}

	if oerr := validateOAuthPKCE(req.CodeChallenge, req.CodeChallengeMethod, oauthEffectiveTokenPolicy(s.securitySettingRepo, client).RequirePKCE); oerr != nil {
		span.SetStatus(codes.Error, "pkce invalid")
		return nil, oerr
	}

	// Validate redirect_uri against registered URIs.
	if oerr := validateClientRedirectURI(client, req.RedirectURI); oerr != nil {
		span.SetStatus(codes.Error, "invalid redirect_uri")
		return nil, oerr
	}

	// Reject dangerous redirect URI schemes.
	if err := security.ValidateRedirectURI(req.RedirectURI); err != nil {
		span.SetStatus(codes.Error, "dangerous redirect_uri scheme")
		return nil, apperror.NewOAuthInvalidRequest("redirect_uri uses a forbidden scheme")
	}

	// Generate a unique request_uri token and hash it for storage.
	rawToken, err := crypto.GenerateRandomString(parRequestLength)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token generation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	tokenHash := crypto.HashAuthorizationCode(rawToken)

	parReq := &OAuthPARRequest{
		RequestURIHash:      tokenHash,
		ClientID:            client.ClientID,
		TenantID:            client.TenantID,
		ResponseType:        req.ResponseType,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(parRequestTTL),
	}
	if req.State != "" {
		parReq.State = &req.State
	}
	if req.Nonce != "" {
		parReq.Nonce = &req.Nonce
	}

	if _, err := s.parRepo.Create(parReq); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "par request creation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.TenantID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthAuthorize,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("PAR request pushed"),
	})

	span.SetStatus(codes.Ok, "")
	return &OAuthPARResponseDTO{
		RequestURI: parURIPrefix + rawToken,
		ExpiresIn:  int(parRequestTTL.Seconds()),
	}, nil
}

// ConsumeRequestURI implements OAuthPARService.
func (s *oauthPARService) ConsumeRequestURI(ctx context.Context, requestURI string) (*OAuthAuthorizeRequestDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_par.consume_request_uri")
	defer span.End()

	rawToken := strings.TrimPrefix(requestURI, parURIPrefix)
	if rawToken == requestURI {
		// Prefix not present — not a valid PAR request_uri.
		span.SetStatus(codes.Error, "invalid request_uri format")
		return nil, apperror.NewOAuthInvalidRequest("request_uri is not a valid PAR URI")
	}

	tokenHash := crypto.HashAuthorizationCode(rawToken)
	parReq, err := s.parRepo.FindByRequestURIHash(tokenHash)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "par lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if parReq == nil {
		span.SetStatus(codes.Error, "par not found")
		return nil, apperror.NewOAuthInvalidRequest("request_uri not found or already used")
	}

	if parReq.IsExpired() {
		span.SetStatus(codes.Error, "par expired")
		return nil, apperror.NewOAuthInvalidRequest("request_uri has expired")
	}

	// Mark as used before returning to prevent replay.
	if err := s.parRepo.MarkUsed(parReq.OAuthPARRequestID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "par mark-used failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	state := ""
	if parReq.State != nil {
		state = *parReq.State
	}
	nonce := ""
	if parReq.Nonce != nil {
		nonce = *parReq.Nonce
	}

	span.SetStatus(codes.Ok, "")
	return &OAuthAuthorizeRequestDTO{
		ResponseType:        parReq.ResponseType,
		ClientID:            resolveClientIdentifier(parReq.Client),
		RedirectURI:         parReq.RedirectURI,
		Scope:               parReq.Scope,
		State:               state,
		Nonce:               nonce,
		CodeChallenge:       parReq.CodeChallenge,
		CodeChallengeMethod: parReq.CodeChallengeMethod,
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers shared across PAR and other OAuth services
// ──────────────────────────────────────────────────────────────────────────────

func (s *oauthPARService) resolveAndAuthenticateClient(creds OAuthClientCredentials) (*Client, *apperror.OAuthError) {
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

	// Verify client secret when the client has one.
	if client.SecretHash != nil && *client.SecretHash != "" {
		if !clientSecretMatches(&client, creds.ClientSecret) {
			return nil, apperror.NewOAuthInvalidClient("client authentication failed")
		}
	}

	return &client, nil
}

func validateClientRedirectURI(client *Client, redirectURI string) *apperror.OAuthError {
	if client.ClientURIs == nil {
		return apperror.NewOAuthInvalidRequest("no redirect URIs registered for this client")
	}
	for _, uri := range *client.ClientURIs {
		if uri.Type == shared.ClientURITypeRedirect && uri.URI == redirectURI {
			return nil
		}
	}
	return apperror.NewOAuthInvalidRequest("redirect_uri does not match any registered redirect URIs")
}

func resolveClientIdentifier(client *Client) string {
	if client == nil {
		return ""
	}
	if client.Identifier != nil {
		return *client.Identifier
	}
	return ""
}
