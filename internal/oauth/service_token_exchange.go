package oauth

import (
	"context"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	// tokenTypeAccessToken is the RFC 8693 token type URI for access tokens.
	tokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
)

// OAuthTokenExchangeService handles the Token Exchange grant (RFC 8693).
type OAuthTokenExchangeService interface {
	// Exchange validates the subject_token and issues a new token of the
	// requested type. Supports access-token-for-access-token exchanges within
	// the same authorization server.
	Exchange(ctx context.Context, req OAuthTokenExchangeRequestDTO, creds OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError)
}

type oauthTokenExchangeService struct {
	db               *gorm.DB
	clientRepo       ClientRepository
	userRepo         UserRepository
	authEventService authevent.AuthEventService
}

// NewOAuthTokenExchangeService creates a new OAuthTokenExchangeService.
func NewOAuthTokenExchangeService(
	db *gorm.DB,
	clientRepo ClientRepository,
	userRepo UserRepository,
	authEventService authevent.AuthEventService,
) OAuthTokenExchangeService {
	return &oauthTokenExchangeService{
		db:               db,
		clientRepo:       clientRepo,
		userRepo:         userRepo,
		authEventService: authEventService,
	}
}

// Exchange implements OAuthTokenExchangeService.
func (s *oauthTokenExchangeService) Exchange(ctx context.Context, req OAuthTokenExchangeRequestDTO, creds OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token_exchange.exchange")
	defer span.End()
	span.SetAttributes(
		attribute.String("oauth.client_id", creds.ClientID),
		attribute.String("oauth.subject_token_type", req.SubjectTokenType),
	)

	// Authenticate the requesting client.
	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		span.SetStatus(codes.Error, "client auth failed")
		return nil, oerr
	}

	if !clientHasGrant(client, GrantTypeTokenExchange) {
		span.SetStatus(codes.Error, "grant not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for token-exchange grant")
	}

	// Validate the subject token.
	claims, err := jwt.ValidateToken(req.SubjectToken)
	if err != nil {
		span.SetStatus(codes.Error, "subject token invalid")
		return nil, apperror.NewOAuthInvalidGrant("subject_token is invalid or expired")
	}

	subjectSub, ok := claims["sub"].(string)
	if !ok || subjectSub == "" {
		span.SetStatus(codes.Error, "subject claim missing")
		return nil, apperror.NewOAuthInvalidGrant("subject_token is missing the sub claim")
	}

	// Determine the issued token type; default to access_token.
	issuedTokenType := req.RequestedTokenType
	if issuedTokenType == "" {
		issuedTokenType = tokenTypeAccessToken
	}

	// Only access_token issuance is supported for now.
	if issuedTokenType != tokenTypeAccessToken {
		span.SetStatus(codes.Error, "unsupported requested_token_type")
		return nil, apperror.NewOAuthInvalidRequest("only access_token issuance is supported as requested_token_type")
	}

	issuer := config.AppPublicHostname
	audience := req.Audience
	if audience == "" {
		audience = issuer
	}
	clientIdentifier := resolveClientIdentifier(client)
	providerID := ""
	if client.IdentityProvider != nil {
		providerID = client.IdentityProvider.IdentityProviderUUID.String()
	}

	scope := req.Scope
	if scope == "" {
		if s, ok := claims["scope"].(string); ok {
			scope = s
		}
	}

	newToken, err := jwt.GenerateAccessToken(
		subjectSub,
		scope,
		issuer,
		audience,
		clientIdentifier,
		providerID,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token generation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.TenantID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthTokenExchange,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Token exchange completed"),
	})

	span.SetStatus(codes.Ok, "")
	return &OAuthTokenExchangeResponseDTO{
		AccessToken:     newToken,
		IssuedTokenType: issuedTokenType,
		TokenType:       "Bearer",
		ExpiresIn:       int64(jwt.AccessTokenTTL.Seconds()),
		Scope:           scope,
	}, nil
}
