package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/jwt"
	"github.com/maintainerd/auth/internal/middleware"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/ptr"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	// refreshTokenByteLength is the byte length of the raw refresh token.
	refreshTokenByteLength = 32
)

// OAuthTokenService handles the OAuth 2.0 token endpoint logic.
type OAuthTokenService interface {
	// Exchange processes a token request. It routes to the appropriate grant
	// handler (authorization_code, refresh_token, client_credentials).
	Exchange(ctx context.Context, req dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError)

	// Revoke revokes a token (access or refresh) per RFC 7009. The server
	// always responds 200 OK regardless of whether the token was found, to
	// prevent information leakage.
	Revoke(ctx context.Context, req dto.OAuthRevokeRequestDTO, creds dto.OAuthClientCredentials) *apperror.OAuthError

	// Introspect inspects a token per RFC 7662. Returns active=false for
	// invalid, expired, or revoked tokens without revealing the reason.
	Introspect(ctx context.Context, req dto.OAuthIntrospectRequestDTO) (*dto.OAuthIntrospectResponseDTO, *apperror.OAuthError)
}

type oauthTokenService struct {
	db               *gorm.DB
	clientRepo       repository.ClientRepository
	authCodeRepo     repository.OAuthAuthorizationCodeRepository
	refreshTokenRepo repository.OAuthRefreshTokenRepository
	userRepo         repository.UserRepository
	userIdentityRepo repository.UserIdentityRepository
	authEventService AuthEventService
}

// NewOAuthTokenService creates a new OAuthTokenService.
func NewOAuthTokenService(
	db *gorm.DB,
	clientRepo repository.ClientRepository,
	authCodeRepo repository.OAuthAuthorizationCodeRepository,
	refreshTokenRepo repository.OAuthRefreshTokenRepository,
	userRepo repository.UserRepository,
	userIdentityRepo repository.UserIdentityRepository,
	authEventService AuthEventService,
) OAuthTokenService {
	return &oauthTokenService{
		db:               db,
		clientRepo:       clientRepo,
		authCodeRepo:     authCodeRepo,
		refreshTokenRepo: refreshTokenRepo,
		userRepo:         userRepo,
		userIdentityRepo: userIdentityRepo,
		authEventService: authEventService,
	}
}

// Exchange implements OAuthTokenService.
func (s *oauthTokenService) Exchange(ctx context.Context, req dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.exchange")
	defer span.End()
	span.SetAttributes(attribute.String("oauth.grant_type", req.GrantType))

	switch req.GrantType {
	case model.GrantTypeAuthorizationCode:
		return s.exchangeAuthorizationCode(ctx, req, creds)
	case model.GrantTypeRefreshToken:
		return s.exchangeRefreshToken(ctx, req, creds)
	case model.GrantTypeClientCredentials:
		return s.exchangeClientCredentials(ctx, req, creds)
	default:
		span.SetStatus(codes.Error, "unsupported grant type")
		return nil, apperror.NewOAuthUnsupportedGrantType("unsupported grant_type")
	}
}

// exchangeAuthorizationCode handles the authorization_code grant (RFC 6749 §4.1.3).
func (s *oauthTokenService) exchangeAuthorizationCode(ctx context.Context, req dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
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
	client, oerr := s.authenticateClient(ctx, creds)
	if oerr != nil {
		return nil, oerr
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
	if authCode.IsUsed {
		// RFC 6749 §4.1.2: If an authorization code is used more than once,
		// the authorization server MUST deny the request and SHOULD revoke all
		// tokens previously issued based on that code.
		s.authEventService.Log(ctx, AuthEventInput{
			TenantID:    authCode.TenantID,
			ActorUserID: &authCode.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    model.AuthEventCategoryAuthn,
			EventType:   model.AuthEventTypeTokenReuse,
			Severity:    model.AuthEventSeverityCritical,
			Result:      model.AuthEventResultFailure,
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
	result, oerr := s.generateTokens(ctx, sub, user, client, authCode.Scope, authCode.Nonce)
	if oerr != nil {
		span.SetStatus(codes.Error, "token generation failed")
		return nil, oerr
	}

	s.authEventService.Log(ctx, AuthEventInput{
		TenantID:    client.TenantID,
		ActorUserID: &authCode.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    model.AuthEventCategoryAuthn,
		EventType:   model.AuthEventTypeOAuthTokenExchange,
		Severity:    model.AuthEventSeverityInfo,
		Result:      model.AuthEventResultSuccess,
		Description: ptr.Ptr("Authorization code exchanged for tokens"),
	})

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// exchangeRefreshToken handles the refresh_token grant (RFC 6749 §6).
func (s *oauthTokenService) exchangeRefreshToken(ctx context.Context, req dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.exchange_refresh_token")
	defer span.End()

	if req.RefreshToken == "" {
		return nil, apperror.NewOAuthInvalidRequest("refresh_token is required")
	}

	// Authenticate the client.
	client, oerr := s.authenticateClient(ctx, creds)
	if oerr != nil {
		return nil, oerr
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
		s.authEventService.Log(ctx, AuthEventInput{
			TenantID:    storedToken.TenantID,
			ActorUserID: &storedToken.UserID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    model.AuthEventCategoryAuthn,
			EventType:   model.AuthEventTypeTokenReuse,
			Severity:    model.AuthEventSeverityCritical,
			Result:      model.AuthEventResultFailure,
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

	// Rotate: revoke the old token and issue a new one in the same family.
	var result *dto.OAuthTokenResult
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		txRefreshRepo := s.refreshTokenRepo.WithTx(tx)

		// Revoke old token.
		if err := txRefreshRepo.RevokeByID(storedToken.OAuthRefreshTokenID); err != nil {
			return err
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
		scope := storedToken.Scope
		if req.Scope != "" {
			scope = req.Scope
		}

		// Generate new access + ID tokens.
		result, oerr = s.generateTokens(ctx, sub, user, client, scope, nil)
		if oerr != nil {
			return oerr
		}

		// Create the new refresh token in the same family.
		rawRT, err := crypto.GenerateRandomString(refreshTokenByteLength)
		if err != nil {
			return err
		}
		rtHash := crypto.HashRefreshToken(rawRT)

		rtTTL := s.refreshTokenTTL(client)
		newToken := &model.OAuthRefreshToken{
			TokenHash: rtHash,
			FamilyID:  storedToken.FamilyID,
			ClientID:  client.ClientID,
			UserID:    storedToken.UserID,
			TenantID:  client.TenantID,
			Scope:     scope,
			ExpiresAt: time.Now().Add(rtTTL),
		}
		if _, err := txRefreshRepo.Create(newToken); err != nil {
			return err
		}

		result.RefreshToken = rawRT
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

	s.authEventService.Log(ctx, AuthEventInput{
		TenantID:    client.TenantID,
		ActorUserID: &storedToken.UserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    model.AuthEventCategoryAuthn,
		EventType:   model.AuthEventTypeOAuthTokenRefresh,
		Severity:    model.AuthEventSeverityInfo,
		Result:      model.AuthEventResultSuccess,
		Description: ptr.Ptr("Refresh token rotated"),
	})

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// exchangeClientCredentials handles the client_credentials grant (RFC 6749 §4.4).
func (s *oauthTokenService) exchangeClientCredentials(ctx context.Context, _ dto.OAuthTokenRequestDTO, creds dto.OAuthClientCredentials) (*dto.OAuthTokenResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.exchange_client_credentials")
	defer span.End()

	// Authenticate the client.
	client, oerr := s.authenticateClient(ctx, creds)
	if oerr != nil {
		return nil, oerr
	}

	// The client must have the client_credentials grant enabled.
	if !hasGrant(client, model.GrantTypeClientCredentials) {
		span.SetStatus(codes.Error, "client_credentials grant not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for client_credentials grant")
	}

	// Client credentials grant only produces an access token — no refresh or
	// ID token.
	issuer := ""
	audience := ""
	identifier := ""
	providerID := ""
	if client.Domain != nil {
		issuer = *client.Domain
	}
	if client.Identifier != nil {
		audience = *client.Identifier
		identifier = *client.Identifier
	}
	if client.IdentityProvider != nil {
		providerID = client.IdentityProvider.Identifier
	}

	accessToken, err := jwt.GenerateAccessToken(
		identifier,
		"", // no user scope for m2m
		issuer,
		audience,
		identifier,
		providerID,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "access token generation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, AuthEventInput{
		TenantID:  client.TenantID,
		IPAddress: middleware.ClientIPFromContext(ctx),
		UserAgent: ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:  model.AuthEventCategoryAuthn,
		EventType: model.AuthEventTypeOAuthClientAuth,
		Severity:  model.AuthEventSeverityInfo,
		Result:    model.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("Client credentials token issued for client %s",
			identifier)),
	})

	expiresIn := int64(jwt.AccessTokenTTL.Seconds())
	if client.AccessTokenTTL != nil {
		expiresIn = int64(*client.AccessTokenTTL)
	}

	span.SetStatus(codes.Ok, "")
	return &dto.OAuthTokenResult{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

// Revoke implements OAuthTokenService.
func (s *oauthTokenService) Revoke(ctx context.Context, req dto.OAuthRevokeRequestDTO, creds dto.OAuthClientCredentials) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.revoke")
	defer span.End()

	// Authenticate the client.
	client, oerr := s.authenticateClient(ctx, creds)
	if oerr != nil {
		return oerr
	}

	// Try to revoke as refresh token first (most common).
	tokenHash := crypto.HashRefreshToken(req.Token)
	storedRT, err := s.refreshTokenRepo.FindByTokenHash(tokenHash)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token lookup failed")
		return nil // RFC 7009 §2.2: always 200 OK
	}

	if storedRT != nil && storedRT.ClientID == client.ClientID {
		if !storedRT.IsRevoked {
			_ = s.refreshTokenRepo.RevokeByID(storedRT.OAuthRefreshTokenID)
			s.authEventService.Log(ctx, AuthEventInput{
				TenantID:    client.TenantID,
				ActorUserID: &storedRT.UserID,
				IPAddress:   middleware.ClientIPFromContext(ctx),
				UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
				Category:    model.AuthEventCategoryAuthn,
				EventType:   model.AuthEventTypeOAuthTokenRevoke,
				Severity:    model.AuthEventSeverityInfo,
				Result:      model.AuthEventResultSuccess,
				Description: ptr.Ptr("Refresh token revoked"),
			})
		}
	}

	// Access tokens are stateless JWTs — we cannot revoke them server-side
	// without a blacklist. RFC 7009 says the server SHOULD revoke if possible;
	// we log the event but don't fail.

	span.SetStatus(codes.Ok, "")
	return nil
}

// Introspect implements OAuthTokenService.
func (s *oauthTokenService) Introspect(ctx context.Context, req dto.OAuthIntrospectRequestDTO) (*dto.OAuthIntrospectResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_token.introspect")
	defer span.End()

	// Try to validate as a JWT (access token or ID token).
	claims, err := jwt.ValidateToken(req.Token)
	if err == nil && claims != nil {
		resp := &dto.OAuthIntrospectResponseDTO{
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
		resp := &dto.OAuthIntrospectResponseDTO{
			Active:    true,
			TokenType: "refresh_token",
			Scope:     storedRT.Scope,
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
	return &dto.OAuthIntrospectResponseDTO{Active: false}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// authenticateClient resolves and validates client credentials from either
// HTTP Basic auth or the request body.
func (s *oauthTokenService) authenticateClient(ctx context.Context, creds dto.OAuthClientCredentials) (*model.Client, *apperror.OAuthError) {
	if creds.ClientID == "" {
		return nil, apperror.NewOAuthInvalidClient("client_id is required")
	}

	// Look up the client.
	client, err := findActiveClientByIdentifier(s.db, creds.ClientID)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil {
		s.logClientAuthFail(ctx, 0, "unknown client_id")
		return nil, apperror.NewOAuthInvalidClient("client authentication failed")
	}

	// Authenticate based on the client's configured method.
	switch client.TokenEndpointAuthMethod {
	case model.TokenAuthMethodNone:
		// Public clients (SPA/mobile) do not have a secret.
	case model.TokenAuthMethodSecretBasic, model.TokenAuthMethodSecretPost:
		if client.Secret == nil || creds.ClientSecret != *client.Secret {
			s.logClientAuthFail(ctx, client.TenantID, "invalid client_secret")
			return nil, apperror.NewOAuthInvalidClient("client authentication failed")
		}
	default:
		return nil, apperror.NewOAuthInvalidClient("unsupported token_endpoint_auth_method")
	}

	return client, nil
}

// generateTokens creates an access token, ID token, and a new refresh token.
func (s *oauthTokenService) generateTokens(ctx context.Context, sub string, user *model.User, client *model.Client, scope string, nonce *string) (*dto.OAuthTokenResult, *apperror.OAuthError) {
	issuer := ""
	audience := ""
	identifier := ""
	providerID := ""
	if client.Domain != nil {
		issuer = *client.Domain
	}
	if client.Identifier != nil {
		audience = *client.Identifier
		identifier = *client.Identifier
	}
	if client.IdentityProvider != nil {
		providerID = client.IdentityProvider.Identifier
	}

	accessToken, err := jwt.GenerateAccessToken(sub, scope, issuer, audience, identifier, providerID)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	nonceStr := ""
	if nonce != nil {
		nonceStr = *nonce
	}

	profile := &jwt.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	idToken, err := jwt.GenerateIDToken(sub, issuer, identifier, providerID, profile, nonceStr)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Generate refresh token.
	rawRT, err := crypto.GenerateRandomString(refreshTokenByteLength)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	rtHash := crypto.HashRefreshToken(rawRT)

	rtTTL := s.refreshTokenTTL(client)
	newRT := &model.OAuthRefreshToken{
		TokenHash: rtHash,
		FamilyID:  uuid.New(),
		ClientID:  client.ClientID,
		UserID:    user.UserID,
		TenantID:  client.TenantID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(rtTTL),
	}
	if _, err := s.refreshTokenRepo.Create(newRT); err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	expiresIn := int64(jwt.AccessTokenTTL.Seconds())
	if client.AccessTokenTTL != nil {
		expiresIn = int64(*client.AccessTokenTTL)
	}

	_ = ctx // used by callers for auth event logging
	return &dto.OAuthTokenResult{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
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
func (s *oauthTokenService) refreshTokenTTL(client *model.Client) time.Duration {
	if client.RefreshTokenTTL != nil {
		return time.Duration(*client.RefreshTokenTTL) * time.Second
	}
	return jwt.RefreshTokenTTL
}

// logClientAuthFail logs a failed client authentication attempt.
func (s *oauthTokenService) logClientAuthFail(ctx context.Context, tenantID int64, reason string) {
	s.authEventService.Log(ctx, AuthEventInput{
		TenantID:    tenantID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    model.AuthEventCategoryAuthn,
		EventType:   model.AuthEventTypeOAuthClientAuthFail,
		Severity:    model.AuthEventSeverityWarn,
		Result:      model.AuthEventResultFailure,
		Description: ptr.Ptr(reason),
	})
}

// hasGrant checks whether the client has the given grant type.
func hasGrant(client *model.Client, grantType string) bool {
	for _, g := range client.GrantTypes {
		if g == grantType {
			return true
		}
	}
	return false
}

// findActiveClientByIdentifier is a shared helper used by the token and
// revocation flows to look up an active client by its OAuth identifier.
func findActiveClientByIdentifier(db *gorm.DB, identifier string) (*model.Client, error) {
	var client model.Client
	err := db.
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		Where("identifier = ? AND status = ?", identifier, model.StatusActive).
		First(&client).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}
