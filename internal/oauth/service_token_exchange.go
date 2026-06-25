package oauth

import (
	"context"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/config"
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
	// tokenTypeAccessToken is the RFC 8693 token type URI for access tokens.
	tokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
)

var (
	oauthTokenExchangeGenerateAccessTokenWithOptionsContext = jwt.GenerateAccessTokenWithOptionsContext
	oauthTokenExchangeValidateTokenWithContext              = jwt.ValidateTokenWithContext
)

// OAuthTokenExchangeService handles the Token Exchange grant (RFC 8693).
type OAuthTokenExchangeService interface {
	// Exchange validates the subject_token and issues a new token of the
	// requested type. Supports access-token-for-access-token exchanges within
	// the same authorization server.
	Exchange(ctx context.Context, req OAuthTokenExchangeRequestDTO, creds OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError)
}

type oauthTokenExchangeService struct {
	db                  *gorm.DB
	clientRepo          ClientRepository
	userRepo            UserRepository
	authEventService    authevent.AuthEventService
	securitySettingRepo secpolicy.SecuritySettingRepository
}

// NewOAuthTokenExchangeService creates a new OAuthTokenExchangeService.
func NewOAuthTokenExchangeService(
	db *gorm.DB,
	clientRepo ClientRepository,
	userRepo UserRepository,
	authEventService authevent.AuthEventService,
	securitySettingRepo ...secpolicy.SecuritySettingRepository,
) OAuthTokenExchangeService {
	var settings secpolicy.SecuritySettingRepository
	if len(securitySettingRepo) > 0 {
		settings = securitySettingRepo[0]
	}
	return &oauthTokenExchangeService{
		db:                  db,
		clientRepo:          clientRepo,
		userRepo:            userRepo,
		authEventService:    authEventService,
		securitySettingRepo: settings,
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
	claims, err := oauthTokenExchangeValidateTokenWithContext(ctx, req.SubjectToken)
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
	providerID := tokenRealm(client)

	scope := req.Scope
	if scope == "" {
		if s, ok := claims["scope"].(string); ok {
			scope = s
		}
	}
	if req.Scope != "" {
		if sourceScope, ok := claims["scope"].(string); ok {
			if oerr := validateRequestedScopesSubset(req.Scope, sourceScope); oerr != nil {
				return nil, oerr
			}
		}
	}
	if oerr := validateClientAllowedScopes(client, scope); oerr != nil {
		span.SetStatus(codes.Error, "scope not allowed")
		return nil, oerr
	}

	accessTokenOpts := oauthAccessTokenOptions(s.securitySettingRepo, client)
	accessTokenOpts.AMR = amrClaimValues(claims["amr"])
	if acr, ok := claims["acr"].(string); ok {
		accessTokenOpts.ACR = acr
	}

	newToken, err := oauthTokenExchangeGenerateAccessTokenWithOptionsContext(
		ctx,
		subjectSub,
		scope,
		issuer,
		audience,
		clientIdentifier,
		providerID,
		accessTokenOpts,
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
		ExpiresIn:       oauthAccessTokenExpiresIn(s.securitySettingRepo, client),
		Scope:           scope,
	}, nil
}

func amrClaimValues(raw any) []string {
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	amr := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok && s != "" {
			amr = append(amr, s)
		}
	}
	return amr
}
