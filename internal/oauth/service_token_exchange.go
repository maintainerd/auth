package oauth

import (
	"context"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	// tokenTypeAccessToken is the RFC 8693 token type URI for access tokens.
	tokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
	// tokenTypeIDTokenURI and tokenTypeRefreshTokenURI are accepted by the DTO
	// validator as syntactically valid subject_token_type values, so the service
	// has to name them to refuse them.
	tokenTypeIDTokenURI      = "urn:ietf:params:oauth:token-type:id_token"
	tokenTypeRefreshTokenURI = "urn:ietf:params:oauth:token-type:refresh_token"
)

var (
	oauthTokenExchangeGenerateAccessTokenWithOptionsContext = jwt.GenerateAccessTokenWithOptionsContext
	// Both the subject and the actor token are validated with the ACCESS-token
	// validator. The generic jwt.ValidateTokenWithContext deliberately does not
	// check what kind of token it was handed, so using it here meant an ID token —
	// which every relying party is handed by design — could be exchanged for a
	// first-class access token for its subject.
	oauthTokenExchangeValidateTokenWithContext = jwt.ValidateAccessTokenWithContext
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
	exchangeRepo        OAuthTokenExchangeRepository
	securitySettingRepo secpolicy.SecuritySettingRepository
}

// NewOAuthTokenExchangeService creates a new OAuthTokenExchangeService.
func NewOAuthTokenExchangeService(
	db *gorm.DB,
	clientRepo ClientRepository,
	userRepo UserRepository,
	authEventService authevent.AuthEventService,
	exchangeRepo OAuthTokenExchangeRepository,
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
		exchangeRepo:        exchangeRepo,
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

	// RFC 8693 §2.1: subject_token_type declares what was presented, and the server
	// must honour it. Only access tokens this server issued can be exchanged: an
	// ID token is an authentication receipt handed to the relying party (OIDC Core
	// §2) and a refresh token is a credential, so exchanging either would turn a
	// token the client legitimately holds into authority it was never granted.
	switch req.SubjectTokenType {
	case tokenTypeAccessToken, subjectTokenTypeJWT:
	case tokenTypeIDTokenURI, tokenTypeRefreshTokenURI:
		span.SetStatus(codes.Error, "unsupported subject_token_type")
		return nil, apperror.NewOAuthInvalidRequest("only access tokens may be exchanged as subject_token")
	default:
		span.SetStatus(codes.Error, "unsupported subject_token_type")
		return nil, apperror.NewOAuthInvalidRequest("unsupported subject_token_type")
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

	// Tenant binding. Every access token this server mints carries provider_id =
	// the issuing tenant's slug (see tokenRealm), and it is non-empty by
	// construction because GenerateAccessToken rejects an empty providerID.
	// Without this check any tenant's token was exchangeable by any other
	// tenant's client — one process-wide signing key means the signature alone
	// proves nothing about which tenant the assertion came from.
	if oerr := assertSameTenantToken(claims, client); oerr != nil {
		span.SetStatus(codes.Error, "subject token tenant mismatch")
		return nil, oerr
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
	clientIdentifier := resolveClientIdentifier(client)
	providerID := tokenRealm(client)

	// The caller's `audience`/`resource` used to be copied straight onto the token,
	// so a client could mint a token addressed to any resource server it liked and
	// that resource server's `aud` check would pass. It must name an API the client
	// is actually granted (client_apis), or nothing at all.
	audience, oerr := resolveRequestedAudience(s.db, client, req.Audience, req.Resource)
	if oerr != nil {
		span.SetStatus(codes.Error, "audience not allowed")
		return nil, oerr
	}
	if audience == "" {
		audience = clientIdentifier
	}
	if audience == "" {
		audience = issuer
	}

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

	// RFC 8693 §4.1: delegation is only a delegation if the actor is
	// authenticated and recorded. actor_token used to be read for one purpose —
	// labelling the audit row "delegation" — with no signature check, no binding
	// to the authenticated client and no `act` claim on the result, so the
	// delegation chain the RFC requires simply did not exist.
	actorClaims, oerr := s.validateActorToken(ctx, req, client)
	if oerr != nil {
		span.SetStatus(codes.Error, "actor token invalid")
		return nil, oerr
	}

	accessTokenOpts := oauthAccessTokenOptions(s.securitySettingRepo, client)
	accessTokenOpts.AMR = amrClaimValues(claims["amr"])
	if acr, ok := claims["acr"].(string); ok {
		accessTokenOpts.ACR = acr
	}
	// Carry the subject token's session through the exchange. Stamping
	// sub_type=exchange unconditionally made the issued token SESSIONLESS to the
	// session middleware, so a client holding the token-exchange grant could
	// launder a session-bound user token into one that survives logout, "sign out
	// everywhere", session revocation and password change for its full TTL.
	// The sessionless label is now only used when the subject token genuinely has
	// no session behind it (a machine principal, or another sessionless grant).
	if sid, ok := claims["sid"].(string); ok && sid != "" {
		accessTokenOpts.SessionID = sid
	} else {
		accessTokenOpts.SubjectType = subjectTypeExchange
	}
	if actorClaims != nil {
		if accessTokenOpts.ExtraClaims == nil {
			accessTokenOpts.ExtraClaims = map[string]any{}
		}
		act := map[string]any{"sub": actorClaims.Sub}
		if actorClaims.ClientID != "" {
			act["client_id"] = actorClaims.ClientID
		}
		accessTokenOpts.ExtraClaims["act"] = act
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

	// B2: Record the exchange for audit. Best-effort — do not fail the exchange if this fails.
	if s.exchangeRepo != nil {
		exchangeType := "impersonation"
		if actorClaims != nil {
			exchangeType = "delegation"
		}
		if err := s.exchangeRepo.Record(&OAuthTokenExchange{
			TenantID:      client.TenantID,
			ActorClientID: client.ClientID,
			ExchangeType:  exchangeType,
		}); err != nil {
			span.RecordError(err)
			// Intentionally not returning — audit failure must not block the exchange.
		}
	}

	span.SetStatus(codes.Ok, "")
	return &OAuthTokenExchangeResponseDTO{
		AccessToken:     newToken,
		IssuedTokenType: issuedTokenType,
		TokenType:       "Bearer",
		ExpiresIn:       oauthAccessTokenExpiresIn(s.securitySettingRepo, client),
		Scope:           scope,
	}, nil
}

// exchangeActor is the authenticated party a delegated token acts on behalf of.
type exchangeActor struct {
	Sub      string
	ClientID string
}

// validateActorToken verifies the RFC 8693 actor_token and binds it to the
// client that authenticated at the token endpoint.
//
// Returns (nil, nil) when the request carries no actor_token — that is a plain
// impersonation exchange, which is still allowed.
func (s *oauthTokenExchangeService) validateActorToken(ctx context.Context, req OAuthTokenExchangeRequestDTO, client *Client) (*exchangeActor, *apperror.OAuthError) {
	if req.ActorToken == "" {
		return nil, nil
	}

	switch req.ActorTokenType {
	case tokenTypeAccessToken, subjectTokenTypeJWT:
	default:
		return nil, apperror.NewOAuthInvalidRequest("only access tokens may be presented as actor_token")
	}

	actorClaims, err := oauthTokenExchangeValidateTokenWithContext(ctx, req.ActorToken)
	if err != nil {
		return nil, apperror.NewOAuthInvalidGrant("actor_token is invalid or expired")
	}

	if oerr := assertSameTenantToken(actorClaims, client); oerr != nil {
		return nil, apperror.NewOAuthInvalidGrant("actor_token was not issued to this tenant")
	}

	actorSub, _ := actorClaims["sub"].(string)
	if actorSub == "" {
		return nil, apperror.NewOAuthInvalidGrant("actor_token is missing the sub claim")
	}

	// The actor must be the party that authenticated. Accepting an actor_token
	// minted for a different client would let a caller name any principal as the
	// delegate and have this server attest to it.
	actorClientID, _ := actorClaims["client_id"].(string)
	if actorClientID == "" || actorClientID != resolveClientIdentifier(client) {
		return nil, apperror.NewOAuthInvalidGrant("actor_token was not issued to the authenticated client")
	}

	return &exchangeActor{Sub: actorSub, ClientID: actorClientID}, nil
}

// assertSameTenantToken checks that a presented token was issued to the same
// tenant as the client that authenticated at the token endpoint.
//
// Fails CLOSED on a missing provider_id: a token this server issued always has
// one, so its absence means the token came from somewhere this check cannot
// reason about.
func assertSameTenantToken(claims jwtlib.MapClaims, client *Client) *apperror.OAuthError {
	providerID, _ := claims["provider_id"].(string)
	if providerID == "" || providerID != tokenRealm(client) {
		return apperror.NewOAuthInvalidGrant("subject_token was not issued to this tenant")
	}
	return nil
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
