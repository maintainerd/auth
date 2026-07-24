package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	// refreshTokenByteLength is the byte length of the raw refresh token.
	refreshTokenByteLength = 32
)

var (
	oauthTokenGenerateRandomString                  = crypto.GenerateRandomString
	oauthTokenGenerateAccessTokenWithOptionsContext = jwt.GenerateAccessTokenWithOptionsContext
	oauthTokenGenerateIDTokenWithContext            = jwt.GenerateIDTokenWithContext
	oauthTokenValidateTokenWithContext              = jwt.ValidateTokenWithContext
)

// OAuthTokenService handles the OAuth 2.0 token endpoint logic.
type OAuthTokenService interface {
	// Exchange processes a token request. It routes to the appropriate grant
	// handler (authorization_code, refresh_token, client_credentials).
	Exchange(ctx context.Context, req OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError)

	// Revoke revokes a token (access or refresh) per RFC 7009. The server
	// always responds 200 OK regardless of whether the token was found, to
	// prevent information leakage.
	Revoke(ctx context.Context, req OAuthRevokeRequestDTO, creds OAuthClientCredentials) *apperror.OAuthError

	// Introspect inspects a token per RFC 7662. Client authentication is
	// required per RFC 7662 §2.1. Returns active=false for invalid, expired,
	// or revoked tokens without revealing the reason.
	Introspect(ctx context.Context, req OAuthIntrospectRequestDTO, creds OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError)

	// SetClientPermissionResolver injects the permission resolver for M2M
	// client_credentials tokens.
	SetClientPermissionResolver(r ClientPermissionResolver)
}

type oauthTokenService struct {
	db                  *gorm.DB
	clientRepo          ClientRepository
	authCodeRepo        OAuthAuthorizationCodeRepository
	refreshTokenRepo    OAuthRefreshTokenRepository
	userRepo            UserRepository
	userIdentityRepo    UserIdentityRepository
	authEventService    authevent.AuthEventService
	jtiDenylist         cache.JTIDenylister
	securitySettingRepo secpolicy.SecuritySettingRepository
	permResolver        ClientPermissionResolver // nil → M2M permission resolution disabled
}

// NewOAuthTokenService creates a new OAuthTokenService.
func NewOAuthTokenService(
	db *gorm.DB,
	clientRepo ClientRepository,
	authCodeRepo OAuthAuthorizationCodeRepository,
	refreshTokenRepo OAuthRefreshTokenRepository,
	userRepo UserRepository,
	userIdentityRepo UserIdentityRepository,
	authEventService authevent.AuthEventService,
	jtiDenylist cache.JTIDenylister,
	securitySettingRepo ...secpolicy.SecuritySettingRepository,
) OAuthTokenService {
	var settings secpolicy.SecuritySettingRepository
	if len(securitySettingRepo) > 0 {
		settings = securitySettingRepo[0]
	}
	return &oauthTokenService{
		db:                  db,
		clientRepo:          clientRepo,
		authCodeRepo:        authCodeRepo,
		refreshTokenRepo:    refreshTokenRepo,
		userRepo:            userRepo,
		userIdentityRepo:    userIdentityRepo,
		authEventService:    authEventService,
		jtiDenylist:         jtiDenylist,
		securitySettingRepo: settings,
	}
}

// Exchange implements OAuthTokenService.
func (s *oauthTokenService) Exchange(ctx context.Context, req OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.exchange")
	defer span.End()
	span.SetAttributes(attribute.String("oauth.grant_type", req.GrantType))

	switch req.GrantType {
	case GrantTypeAuthorizationCode:
		return s.exchangeAuthorizationCode(ctx, req, creds)
	case GrantTypeRefreshToken:
		return s.exchangeRefreshToken(ctx, req, creds)
	case GrantTypeClientCredentials:
		return s.exchangeClientCredentials(ctx, req, creds)
	default:
		span.SetStatus(codes.Error, "unsupported grant type")
		return nil, apperror.NewOAuthUnsupportedGrantType("unsupported grant_type")
	}
}

// exchangeAuthorizationCode handles the authorization_code grant (RFC 6749 §4.1.3).
func (s *oauthTokenService) exchangeAuthorizationCode(ctx context.Context, req OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.exchange_authorization_code")
	defer span.End()

	if req.Code == "" {
		return nil, apperror.NewOAuthInvalidRequest("code is required for authorization_code grant")
	}
	if req.RedirectURI == "" {
		return nil, apperror.NewOAuthInvalidRequest("redirect_uri is required for authorization_code grant")
	}
	if req.CodeVerifier == "" {
		return nil, apperror.NewOAuthInvalidRequest("code_verifier is required (PKCE)")
	}

	// Authenticate the client.
	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		return nil, oerr
	}

	if !clientHasGrant(client, GrantTypeAuthorizationCode) {
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for authorization_code grant")
	}

	// Look up the authorization code by hash.
	codeHash := crypto.HashAuthorizationCode(req.Code)
	authCode, err := s.authCodeRepo.FindByCodeHash(codeHash)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "authorization code lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if authCode == nil {
		span.SetStatus(codes.Error, "authorization code not found")
		return nil, apperror.NewOAuthInvalidGrant("the authorization code is invalid")
	}

	// Check that the code has not been used.
	if authCode.Used {
		// RFC 6749 §4.1.2: If an authorization code is used more than once,
		// the authorization server MUST deny the request and SHOULD revoke all
		// tokens previously issued based on that code.
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    authCode.TenantID,
			ActorUserID: &authCode.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeTokenReuse,
			Severity:    authevent.AuthEventSeverityCritical,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("Authorization code reuse detected"),
		})
		// Revoke all refresh tokens for this user-client pair.
		_, _ = s.refreshTokenRepo.RevokeByUserAndClient(authCode.UserID, authCode.ClientID)
		span.SetStatus(codes.Error, "authorization code reuse")
		return nil, apperror.NewOAuthInvalidGrant("the authorization code has already been used")
	}

	// Check expiry.
	if authCode.IsExpired() {
		span.SetStatus(codes.Error, "authorization code expired")
		return nil, apperror.NewOAuthInvalidGrant("the authorization code has expired")
	}

	// Verify client binding.
	if authCode.ClientID != client.ClientID {
		span.SetStatus(codes.Error, "client mismatch")
		return nil, apperror.NewOAuthInvalidGrant("the authorization code was not issued to this client")
	}
	if authCode.TenantID == 0 || authCode.TenantID != client.TenantID {
		span.SetStatus(codes.Error, "tenant mismatch")
		return nil, apperror.NewOAuthInvalidGrant("the authorization code was not issued to this tenant")
	}

	if oerr := validateClientAllowedScopes(client, strings.Join([]string(authCode.Scope), " ")); oerr != nil {
		span.SetStatus(codes.Error, "scope not allowed")
		return nil, oerr
	}

	// Verify redirect_uri matches.
	if authCode.RedirectURI != req.RedirectURI {
		span.SetStatus(codes.Error, "redirect_uri mismatch")
		return nil, apperror.NewOAuthInvalidGrant("redirect_uri does not match the value used in the authorization request")
	}

	// Validate PKCE code_verifier against the stored code_challenge.
	if err := crypto.ValidatePKCEChallenge(req.CodeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod); err != nil {
		span.SetStatus(codes.Error, "PKCE validation failed")
		return nil, apperror.NewOAuthInvalidGrant("PKCE validation failed")
	}

	// Mark the code as used — one-time use is enforced at the application level.
	if err := s.authCodeRepo.MarkUsed(authCode.OAuthAuthorizationCodeID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to mark code as used")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Resolve the user identity sub for token claims.
	sub, err := s.resolveUserSub(authCode.UserID, client.ClientID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to resolve user sub")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Get user for ID token profile claims.
	user, err := s.userRepo.FindByID(authCode.UserID)
	if err != nil || user == nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user not found")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Generate tokens.
	result, oerr := s.generateTokens(ctx, sub, user, client, strings.Join([]string(authCode.Scope), " "), authCode.Nonce, req.DPoPThumbprint)
	if oerr != nil {
		span.SetStatus(codes.Error, "token generation failed")
		return nil, oerr
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.TenantID,
		ActorUserID: &authCode.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthTokenExchange,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Authorization code exchanged for tokens"),
	})

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// exchangeRefreshToken handles the refresh_token grant (RFC 6749 §6).
func (s *oauthTokenService) exchangeRefreshToken(ctx context.Context, req OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.exchange_refresh_token")
	defer span.End()

	if req.RefreshToken == "" {
		return nil, apperror.NewOAuthInvalidRequest("refresh_token is required")
	}

	// Authenticate the client.
	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		return nil, oerr
	}

	if !clientHasGrant(client, GrantTypeRefreshToken) {
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for refresh_token grant")
	}

	// Look up the refresh token by hash.
	tokenHash := crypto.HashRefreshToken(req.RefreshToken)
	storedToken, err := s.refreshTokenRepo.FindByTokenHash(tokenHash)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "refresh token lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if storedToken == nil {
		span.SetStatus(codes.Error, "refresh token not found")
		return nil, apperror.NewOAuthInvalidGrant("the refresh token is invalid")
	}

	// Reuse detection — if the token is already revoked, the entire family is
	// compromised.
	if storedToken.IsRevoked {
		reuseInterval := oauthEffectiveSessionPolicy(s.securitySettingRepo, client).RefreshTokenReuseIntervalSeconds
		if reuseInterval > 0 && storedToken.RevokedAt != nil && time.Since(*storedToken.RevokedAt) <= time.Duration(reuseInterval)*time.Second {
			span.SetStatus(codes.Error, "refresh token reused within grace interval")
			return nil, apperror.NewOAuthInvalidGrant("the refresh token has already been used")
		}
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    storedToken.TenantID,
			ActorUserID: &storedToken.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeTokenReuse,
			Severity:    authevent.AuthEventSeverityCritical,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr(fmt.Sprintf("Refresh token reuse detected, revoking family %s", storedToken.FamilyID)),
		})
		_, _ = s.refreshTokenRepo.RevokeByFamily(storedToken.FamilyID)
		span.SetStatus(codes.Error, "refresh token reuse")
		return nil, apperror.NewOAuthInvalidGrant("the refresh token has been revoked")
	}

	// Check expiry.
	if storedToken.IsExpired() {
		span.SetStatus(codes.Error, "refresh token expired")
		return nil, apperror.NewOAuthInvalidGrant("the refresh token has expired")
	}

	// Verify client binding.
	if storedToken.ClientID != client.ClientID {
		span.SetStatus(codes.Error, "client mismatch")
		return nil, apperror.NewOAuthInvalidGrant("the refresh token was not issued to this client")
	}
	if storedToken.TenantID == 0 || storedToken.TenantID != client.TenantID {
		span.SetStatus(codes.Error, "tenant mismatch")
		return nil, apperror.NewOAuthInvalidGrant("the refresh token was not issued to this tenant")
	}

	// Rotate: revoke the old token and issue a new one in the same family.
	// When token rotation is disabled, the existing token is reused and no new
	// refresh token is issued — only access/id tokens are regenerated.
	sessionPolicy := oauthEffectiveSessionPolicy(s.securitySettingRepo, client)
	rotate := sessionPolicy.RotateRefreshTokens

	var result *OAuthTokenResult
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		txRefreshRepo := s.refreshTokenRepo.WithTx(tx)

		if rotate {
			// Revoke old token.
			if err := txRefreshRepo.RevokeByID(storedToken.OAuthRefreshTokenID); err != nil {
				return err
			}
		}

		// Resolve user sub.
		sub, err := s.resolveUserSub(storedToken.UserID, client.ClientID)
		if err != nil {
			return err
		}

		// Get user for profile claims.
		user, err := s.userRepo.FindByID(storedToken.UserID)
		if err != nil || user == nil {
			return fmt.Errorf("user not found: %w", err)
		}

		// Use the same scope unless a narrower scope was requested.
		scope := strings.Join([]string(storedToken.Scope), " ")
		if req.Scope != "" {
			if oerr := validateRequestedScopesSubset(req.Scope, strings.Join([]string(storedToken.Scope), " ")); oerr != nil {
				return oerr
			}
			scope = req.Scope
		}
		if oerr := validateClientAllowedScopes(client, scope); oerr != nil {
			return oerr
		}

		// Generate new access + ID tokens.
		result, oerr = s.generateTokens(ctx, sub, user, client, scope, nil, req.DPoPThumbprint)
		if oerr != nil {
			return oerr
		}

		if rotate {
			// Create the new refresh token in the same family.
			rawRT, err := oauthTokenGenerateRandomString(refreshTokenByteLength)
			if err != nil {
				return err
			}
			rtHash := crypto.HashRefreshToken(rawRT)

			rtTTL := s.refreshTokenTTL(client)
			newToken := &OAuthRefreshToken{
				TokenHash: rtHash,
				FamilyID:  storedToken.FamilyID,
				ClientID:  client.ClientID,
				UserID:    storedToken.UserID,
				TenantID:  client.TenantID,
				Scope:     parseScopeFields(scope),
				ExpiresAt: time.Now().Add(rtTTL),
			}
			if _, err := txRefreshRepo.Create(newToken); err != nil {
				return err
			}

			result.RefreshToken = rawRT
		} else {
			// Reuse the incoming refresh token without rotation.
			result.RefreshToken = req.RefreshToken
		}
		return nil
	})

	if txErr != nil {
		// Check if it's an OAuthError from generateTokens.
		if oe, ok := txErr.(*apperror.OAuthError); ok {
			return nil, oe
		}
		span.RecordError(txErr)
		span.SetStatus(codes.Error, "refresh token rotation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.TenantID,
		ActorUserID: &storedToken.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthTokenRefresh,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Refresh token rotated"),
	})

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// exchangeClientCredentials handles the client_credentials grant (RFC 6749 §4.4).
func (s *oauthTokenService) exchangeClientCredentials(ctx context.Context, _ OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.exchange_client_credentials")
	defer span.End()

	// Authenticate the client.
	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		return nil, oerr
	}

	// The client must have the client_credentials grant enabled.
	if !clientHasGrant(client, GrantTypeClientCredentials) {
		span.SetStatus(codes.Error, "client_credentials grant not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for client_credentials grant")
	}

	// Client credentials grant only produces an access token — no refresh or
	// ID token.
	issuer := ""
	audience := ""
	identifier := ""
	providerID := tokenRealm(client)
	subjectType := "client"
	if client.Domain != nil {
		issuer = *client.Domain
	}
	if client.Identifier != nil {
		audience = *client.Identifier
		identifier = *client.Identifier
	}
	subject := identifier
	serviceName := ""
	if client.Service != nil && client.Service.Name != "" {
		subject = client.Service.Name
		serviceName = client.Service.Name
		subjectType = "service"
	}
	opts := s.clientAccessTokenOpts(client)
	opts.Service = serviceName
	opts.SubjectType = subjectType

	// Resolve inherited permissions (client_roles → role_permissions) +
	// direct client_permissions for M2M tokens.
	if s.permResolver != nil && client.ClientID > 0 {
		perms, err := s.permResolver.ResolvePermissions(ctx, client.ClientID)
		if err == nil && len(perms) > 0 {
			if opts.ExtraClaims == nil {
				opts.ExtraClaims = map[string]any{}
			}
			opts.ExtraClaims["permissions"] = perms
		}
	}

	accessToken, err := oauthTokenGenerateAccessTokenWithOptionsContext(
		ctx,
		subject,
		"", // no user scope for m2m
		issuer,
		audience,
		identifier,
		providerID,
		opts,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "access token generation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:  client.TenantID,
		IPAddress: middleware.ClientIPFromContext(ctx),
		UserAgent: ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:  authevent.AuthEventCategoryAuthn,
		EventType: authevent.AuthEventTypeOAuthClientAuth,
		Severity:  authevent.AuthEventSeverityInfo,
		Result:    authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client credentials token issued for client %s",
			identifier)),
	})

	expiresIn := int64(oauthEffectiveSessionPolicy(s.securitySettingRepo, client).AccessTokenTTLSeconds)

	span.SetStatus(codes.Ok, "")
	return &OAuthTokenResult{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

// Revoke implements OAuthTokenService.
func (s *oauthTokenService) Revoke(ctx context.Context, req OAuthRevokeRequestDTO, creds OAuthClientCredentials) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.revoke")
	defer span.End()

	// Authenticate the client.
	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		return oerr
	}

	// Try to revoke as refresh token first (most common).
	tokenHash := crypto.HashRefreshToken(req.Token)
	storedRT, err := s.refreshTokenRepo.FindByTokenHash(tokenHash)
	if err != nil {
		span.RecordError(err)
	}
	if err == nil && storedRT != nil && storedRT.ClientID == client.ClientID {
		if !storedRT.IsRevoked {
			_ = s.refreshTokenRepo.RevokeByID(storedRT.OAuthRefreshTokenID)
			s.authEventService.Log(ctx, authevent.AuthEventInput{
				TenantID:    client.TenantID,
				ActorUserID: &storedRT.UserID,
				IPAddress:   middleware.ClientIPFromContext(ctx),
				UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
				Category:    authevent.AuthEventCategoryAuthn,
				EventType:   authevent.AuthEventTypeOAuthTokenRevoke,
				Severity:    authevent.AuthEventSeverityInfo,
				Result:      authevent.AuthEventResultSuccess,
				Description: ptr.Ptr("Refresh token revoked"),
			})
		}
	}

	if oerr := s.revokeAccessToken(ctx, req.Token, client); oerr != nil {
		span.RecordError(oerr)
		span.SetStatus(codes.Error, "access token revoke failed")
		return oerr
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *oauthTokenService) revokeAccessToken(ctx context.Context, rawToken string, client *Client) *apperror.OAuthError {
	if s.jtiDenylist == nil {
		return nil
	}

	claims, err := oauthTokenValidateTokenWithContext(ctx, rawToken)
	if err != nil || claims == nil {
		return nil
	}

	tokenType, _ := claims["token_type"].(string)
	if tokenType != "access_token" {
		return nil
	}

	if clientID, _ := claims["client_id"].(string); client.Identifier == nil || clientID != *client.Identifier {
		return nil
	}

	jti, _ := claims["jti"].(string)
	if strings.TrimSpace(jti) == "" {
		return nil
	}

	ttl := tokenRemainingTTL(claims["exp"])
	if ttl <= 0 {
		return nil
	}

	if err := s.jtiDenylist.DenyJTI(ctx, jti, ttl); err != nil {
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.TenantID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthTokenRevoke,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Access token revoked"),
	})
	return nil
}

func tokenRemainingTTL(expClaim any) time.Duration {
	var expUnix int64
	switch exp := expClaim.(type) {
	case float64:
		expUnix = int64(exp)
	case int64:
		expUnix = exp
	case int:
		expUnix = int64(exp)
	case json.Number:
		parsed, err := exp.Int64()
		if err != nil {
			return 0
		}
		expUnix = parsed
	default:
		return 0
	}
	return time.Until(time.Unix(expUnix, 0))
}

// Introspect implements OAuthTokenService.
func (s *oauthTokenService) Introspect(ctx context.Context, req OAuthIntrospectRequestDTO, creds OAuthClientCredentials) (*OAuthIntrospectResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.introspect")
	defer span.End()

	_, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		return nil, oerr
	}

	// Try to validate as a JWT (access token or ID token).
	claims, err := oauthTokenValidateTokenWithContext(ctx, req.Token)
	if err == nil && claims != nil {
		resp := &OAuthIntrospectResponseDTO{
			Active:    true,
			TokenType: "Bearer",
		}
		if sub, ok := claims["sub"].(string); ok {
			resp.Sub = sub
		}
		if scope, ok := claims["scope"].(string); ok {
			resp.Scope = scope
		}
		if clientID, ok := claims["client_id"].(string); ok {
			resp.ClientID = clientID
		}
		if aud, ok := claims["aud"].(string); ok {
			resp.Aud = aud
		}
		if iss, ok := claims["iss"].(string); ok {
			resp.Iss = iss
		}
		if jti, ok := claims["jti"].(string); ok {
			resp.Jti = jti
		}
		if exp, ok := claims["exp"].(float64); ok {
			resp.Exp = int64(exp)
		}
		if iat, ok := claims["iat"].(float64); ok {
			resp.Iat = int64(iat)
		}
		if nbf, ok := claims["nbf"].(float64); ok {
			resp.Nbf = int64(nbf)
		}

		span.SetStatus(codes.Ok, "")
		return resp, nil
	}

	// Try as a refresh token.
	tokenHash := crypto.HashRefreshToken(req.Token)
	storedRT, lookupErr := s.refreshTokenRepo.FindByTokenHash(tokenHash)
	if lookupErr == nil && storedRT != nil && storedRT.IsActive() {
		resp := &OAuthIntrospectResponseDTO{
			Active:    true,
			TokenType: "refresh_token",
			Scope:     strings.Join([]string(storedRT.Scope), " "),
			Exp:       storedRT.ExpiresAt.Unix(),
			Iat:       storedRT.CreatedAt.Unix(),
		}

		// Resolve the user sub.
		sub, subErr := s.resolveUserSub(storedRT.UserID, storedRT.ClientID)
		if subErr == nil {
			resp.Sub = sub
		}

		span.SetStatus(codes.Ok, "")
		return resp, nil
	}

	// Token is invalid, expired, revoked, or unknown — return active=false.
	span.SetStatus(codes.Ok, "token inactive")
	return &OAuthIntrospectResponseDTO{Active: false}, nil
}

func (s *oauthTokenService) SetClientPermissionResolver(r ClientPermissionResolver) {
	s.permResolver = r
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// generateTokens creates an access token, ID token, and a new refresh token.
func (s *oauthTokenService) generateTokens(ctx context.Context, sub string, user *User, client *Client, scope string, nonce *string, dpopThumbprint string) (*OAuthTokenResult, *apperror.OAuthError) {
	issuer := ""
	audience := ""
	identifier := ""
	providerID := tokenRealm(client)
	if client.Domain != nil {
		issuer = *client.Domain
	}
	if client.Identifier != nil {
		audience = *client.Identifier
		identifier = *client.Identifier
	}

	accessTokenOpts := s.clientAccessTokenOpts(client)
	if dpopThumbprint != "" {
		accessTokenOpts.DPoPThumbprint = dpopThumbprint
	}
	accessTokenOpts.AMR = []string{jwt.AMRPassword}
	accessTokenOpts.ACR = jwt.ACRLevel1

	accessToken, err := oauthTokenGenerateAccessTokenWithOptionsContext(ctx, sub, scope, issuer, audience, identifier, providerID, accessTokenOpts)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	nonceStr := ""
	if nonce != nil {
		nonceStr = *nonce
	}

	profile := buildUserProfile(user)

	idTokenParams := s.buildIDTokenParams(scope, client, roleNames(user))

	idToken, err := oauthTokenGenerateIDTokenWithContext(ctx, sub, issuer, identifier, providerID, profile, nonceStr, idTokenParams)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Refresh tokens are only issued when offline_access scope is requested
	// (RFC 6749 §1.5) or for authorization_code grant with a valid DPoP binding.
	var rawRT string
	if hasOfflineAccess(scope) {
		rawRT, err = oauthTokenGenerateRandomString(refreshTokenByteLength)
		if err != nil {
			return nil, apperror.NewOAuthServerError("an unexpected error occurred")
		}
		rtHash := crypto.HashRefreshToken(rawRT)

		rtTTL := s.refreshTokenTTL(client)
		newRT := &OAuthRefreshToken{
			TokenHash: rtHash,
			FamilyID:  uuid.New(),
			ClientID:  client.ClientID,
			UserID:    user.UserID,
			TenantID:  client.TenantID,
			Scope:     parseScopeFields(scope),
			ExpiresAt: time.Now().Add(rtTTL),
		}
		if _, err := s.refreshTokenRepo.Create(newRT); err != nil {
			return nil, apperror.NewOAuthServerError("an unexpected error occurred")
		}
	}

	expiresIn := int64(oauthEffectiveSessionPolicy(s.securitySettingRepo, client).AccessTokenTTLSeconds)

	tokenType := "Bearer"
	if dpopThumbprint != "" {
		tokenType = "DPoP"
	}

	_ = ctx // used by callers for auth event logging
	return &OAuthTokenResult{
		AccessToken:  accessToken,
		TokenType:    tokenType,
		ExpiresIn:    expiresIn,
		RefreshToken: rawRT,
		IDToken:      idToken,
		Scope:        scope,
	}, nil
}

// resolveUserSub looks up the user identity sub claim for the given
// user-client pair. Identity records are created during registration and
// login — the OAuth layer only reads them. Returns an error if no identity
// exists, since a user must have an identity to participate in an OAuth flow.
func (s *oauthTokenService) resolveUserSub(userID, clientID int64) (string, error) {
	identity, err := s.userIdentityRepo.FindByUserIDAndClientID(userID, clientID)
	if err != nil {
		return "", err
	}
	if identity != nil {
		return identity.Sub, nil
	}
	return "", fmt.Errorf("no identity found for user %d and client %d", userID, clientID)
}

// refreshTokenTTL returns the refresh token TTL for the client, falling back
// to the global default from the jwt package.
func (s *oauthTokenService) refreshTokenTTL(client *Client) time.Duration {
	policy := oauthEffectiveSessionPolicy(s.securitySettingRepo, client)
	if policy.RefreshTokenTTLSeconds > 0 {
		return time.Duration(policy.RefreshTokenTTLSeconds) * time.Second
	}
	return jwt.RefreshTokenTTL
}

// findActiveClientByIdentifier is a shared helper used by the token and
// revocation flows to look up an active client by its OAuth identifier.
func findActiveClientByIdentifier(db *gorm.DB, identifier string) (*Client, error) {
	var client Client
	err := db.
		Preload("Tenant").
		Preload("Service").
		Where("identifier = ? AND status = ?", identifier, shared.StatusActive).
		First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func tokenRealm(client *Client) string {
	if client == nil {
		return ""
	}
	if client.Tenant != nil && strings.TrimSpace(client.Tenant.Identifier) != "" {
		return client.Tenant.Identifier
	}
	return fmt.Sprintf("tenant:%d", client.TenantID)
}

func (s *oauthTokenService) clientAccessTokenOpts(client *Client) *jwt.AccessTokenOptions {
	return oauthAccessTokenOptions(s.securitySettingRepo, client)
}

func buildUserProfile(user *User) *jwt.UserProfile {
	p := &jwt.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	if user.Profile != nil {
		p.FirstName = user.Profile.FirstName
		if user.Profile.LastName != nil {
			p.LastName = *user.Profile.LastName
		}
		if user.Profile.ProfileURL != nil {
			p.Picture = *user.Profile.ProfileURL
		}

		parts := []string{}
		if p.FirstName != "" {
			parts = append(parts, p.FirstName)
		}
		if p.LastName != "" {
			parts = append(parts, p.LastName)
		}
		if len(parts) > 0 {
			p.Name = strings.Join(parts, " ")
		}
	}

	if p.Name == "" {
		p.Name = user.Fullname
	}

	return p
}

func (s *oauthTokenService) buildIDTokenParams(scope string, client *Client, userRoles []string) *jwt.IDTokenParams {
	return buildIDTokenParamsWithPolicy(s.securitySettingRepo, scope, client, userRoles)
}

func buildIDTokenParams(scope string, client *Client) *jwt.IDTokenParams {
	return buildIDTokenParamsWithPolicy(nil, scope, client, nil)
}

func buildIDTokenParamsWithPolicy(repo secpolicy.SecuritySettingRepository, scope string, client *Client, userRoles []string) *jwt.IDTokenParams {
	scopes := parseScopes(scope)
	if len(scopes) == 0 {
		return nil
	}
	tokenPolicy := oauthEffectiveTokenPolicy(repo, client)

	params := &jwt.IDTokenParams{
		RequestedScopes:  scopes,
		AMR:              []string{jwt.AMRPassword},
		ACR:              jwt.ACRLevel1,
		SigningAlgorithm: tokenPolicy.SigningAlgorithm,
	}

	if client.ScopeClaimMappings != nil {
		var mappings map[string][]string
		if err := json.Unmarshal(client.ScopeClaimMappings, &mappings); err == nil {
			params.ScopeClaimMappings = mappings
		}
	}

	if client.ClaimMappers != nil {
		var extraClaims map[string]any
		if err := json.Unmarshal(client.ClaimMappers, &extraClaims); err == nil {
			// Strip reserved names first. These mappers are operator-configured and
			// merged into the token last, so without this a mapper of
			// {"sub":"<victim>","permissions":["*"],"exp":9999999999} would yield a
			// correctly-signed token impersonating any subject, in any tenant, that
			// never expires.
			params.ExtraClaims = jwt.SanitizeClientClaimMappers(extraClaims)
		}
	}

	for _, claim := range tokenPolicy.AdditionalIDTokenClaims {
		switch claim {
		case "tenant_id":
			if client != nil && client.TenantID > 0 {
				if params.ExtraClaims == nil {
					params.ExtraClaims = map[string]any{}
				}
				params.ExtraClaims["tenant_id"] = client.TenantID
			}
		case "roles":
			if len(userRoles) > 0 {
				if params.ExtraClaims == nil {
					params.ExtraClaims = map[string]any{}
				}
				params.ExtraClaims["roles"] = userRoles
			}
		}
	}

	return params
}

func parseScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	parts := strings.Fields(scope)
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func hasOfflineAccess(scope string) bool {
	for _, s := range parseScopes(scope) {
		if s == "offline_access" {
			return true
		}
	}
	return false
}

func roleNames(user *User) []string {
	if user == nil || len(user.Roles) == 0 {
		return nil
	}
	names := make([]string, len(user.Roles))
	for i, r := range user.Roles {
		names[i] = r.Name
	}
	return names
}
