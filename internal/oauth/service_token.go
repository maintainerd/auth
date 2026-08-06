package oauth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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

	// SetSessionAuthContextResolver injects the reader for a session's recorded
	// acr/amr/auth_time. Without it, tokens fall back to asserting a
	// single-factor password login regardless of how the user authenticated.
	SetSessionAuthContextResolver(r SessionAuthContextResolver)
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
	permResolver        ClientPermissionResolver   // nil → M2M permission resolution disabled
	sessionAuthResolver SessionAuthContextResolver // nil → fall back to pwd / acr=1
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
		// RFC 6749 §4.1.2 says to revoke the tokens issued FROM THIS CODE. Revoking
		// every refresh token for the user-client pair does both too much and too
		// little: it destroys unrelated sessions on that client (a DoS lever — the
		// attacker only has to replay a code to sign the victim out everywhere),
		// while the access token the first redemption actually minted stays live
		// for its full TTL, which is the one credential the attacker's replay is
		// racing for. Deny that token's JTI as well.
		s.revokeTokensIssuedFromCode(ctx, codeHash, authCode)
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

	// Validate PKCE against the challenge stored WITH THIS CODE.
	//
	// The binding is per-authorization-request, not global: /authorize only
	// demands a code_challenge when the client's RequirePKCE policy is on, so a
	// confidential web client legitimately running without PKCE receives a code
	// that has no challenge attached. This used to reject every request that
	// arrived without a code_verifier before the code was even loaded, so such a
	// client could obtain a code and then never redeem it.
	//
	// A code that DOES carry a challenge still requires a matching verifier —
	// omitting it is rejected below, so a public client cannot strip PKCE off its
	// own authorization by simply not sending the verifier.
	//
	// A PUBLIC client is the exception to "no challenge, no check". It presents
	// no credential at this endpoint at all, so without PKCE the code is the only
	// secret in the flow and anyone who observes it — custom-scheme hijack on
	// mobile, a Referer leak, a proxy log — redeems it for the victim's tokens.
	// The code is only issued without a challenge when /authorize did not demand
	// one, which is now impossible for a public client (see isPublicOAuthClient),
	// so reaching here means the code predates that rule or was minted around it.
	// RFC 9700 §2.1.1.
	if authCode.CodeChallenge == "" && isPublicOAuthClient(client) {
		span.SetStatus(codes.Error, "PKCE required for public client")
		return nil, apperror.NewOAuthInvalidGrant("PKCE is required for this client; re-authorize with a code_challenge")
	}
	if authCode.CodeChallenge != "" && req.CodeVerifier == "" {
		span.SetStatus(codes.Error, "PKCE verifier missing")
		return nil, apperror.NewOAuthInvalidGrant("code_verifier is required (PKCE)")
	}
	if authCode.CodeChallenge != "" {
		if err := crypto.ValidatePKCEChallenge(req.CodeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod); err != nil {
			span.SetStatus(codes.Error, "PKCE validation failed")
			return nil, apperror.NewOAuthInvalidGrant("PKCE validation failed")
		}
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

	// RFC 8707 §2.2: the client may name the resource server this token is for.
	// Validated against client_apis, so a client can only ever be issued a token
	// for an API an operator granted it.
	apiAudience, oerr := resolveRequestedAudience(s.db, client, req.Audience, req.Resource)
	if oerr != nil {
		span.SetStatus(codes.Error, "audience not allowed")
		return nil, oerr
	}

	// Generate tokens.
	result, oerr := s.generateTokens(ctx, sub, user, client, strings.Join([]string(authCode.Scope), " "), authCode.Nonce, req.DPoPThumbprint, true, authCode.UserSessionUUID, apiAudience)
	if oerr != nil {
		span.SetStatus(codes.Error, "token generation failed")
		return nil, oerr
	}

	// Remember which access token this code produced, so a later replay of the
	// same code can actually revoke it (see the reuse branch above).
	s.rememberTokenIssuedFromCode(ctx, codeHash, result.AccessToken)

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

	// Verify the DPoP binding (RFC 9449 §5). A refresh token issued to a
	// proofing client is sender-constrained: it may only be redeemed by a caller
	// that proves possession of the SAME key. Without this the binding is
	// decorative — a stolen refresh token could be replayed with any freshly
	// generated key, or with no proof at all, and be honoured.
	if storedToken.DPoPJKT != nil && *storedToken.DPoPJKT != "" {
		if req.DPoPThumbprint == "" {
			span.SetStatus(codes.Error, "dpop proof missing")
			return nil, apperror.NewOAuthInvalidGrant("the refresh token is bound to a DPoP key and requires a DPoP proof")
		}
		// Constant-time: the thumbprint is not a secret, but comparing it in
		// constant time costs nothing and keeps the token path uniform.
		if subtle.ConstantTimeCompare([]byte(*storedToken.DPoPJKT), []byte(req.DPoPThumbprint)) != 1 {
			s.authEventService.Log(ctx, authevent.AuthEventInput{
				TenantID:    storedToken.TenantID,
				ActorUserID: &storedToken.UserID,
				IPAddress:   middleware.ClientIPFromContext(ctx),
				UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
				Category:    authevent.AuthEventCategoryAuthn,
				EventType:   authevent.AuthEventTypeTokenReuse,
				Severity:    authevent.AuthEventSeverityCritical,
				Result:      authevent.AuthEventResultFailure,
				Description: ptr.Ptr(fmt.Sprintf("DPoP key mismatch on refresh, revoking family %s", storedToken.FamilyID)),
			})
			// A proof over a different key means the token is in hands other than
			// the ones it was issued to — treat it exactly like reuse.
			_, _ = s.refreshTokenRepo.RevokeByFamily(storedToken.FamilyID)
			span.SetStatus(codes.Error, "dpop key mismatch")
			return nil, apperror.NewOAuthInvalidGrant("the refresh token is bound to a different DPoP key")
		}
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
		apiAudience, audErr := resolveRequestedAudience(s.db, client, req.Audience, req.Resource)
		if audErr != nil {
			return audErr
		}

		result, oerr = s.generateTokens(ctx, sub, user, client, scope, nil, req.DPoPThumbprint, false, storedToken.UserSessionUUID, apiAudience)
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
				// Both bindings are properties of the family, not of the individual
				// token — carry them across rotation so a DPoP-bound family can
				// never silently downgrade to bearer on its next hop, and so a
				// rotated token stays revocable with its session.
				DPoPJKT:         storedToken.DPoPJKT,
				UserSessionUUID: storedToken.UserSessionUUID,
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
func (s *oauthTokenService) exchangeClientCredentials(ctx context.Context, req OAuthTokenRequestDTO, creds OAuthClientCredentials) (*OAuthTokenResult, *apperror.OAuthError) {
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
	// The issuer identifies the authorization server, not the client — see
	// jwt.TokenIssuer. Stamping the client's domain made `iss` disagree with the
	// `issuer` in the discovery document, which a compliant RP must reject.
	issuer = jwt.TokenIssuerPtr(client.Domain)
	if client.Identifier != nil {
		audience = *client.Identifier
		identifier = *client.Identifier
	}
	// RFC 8707: an m2m client naming a registered API gets a token addressed to
	// that API instead of to itself, which is the only way an api_permissions-
	// gated resource server can ever be handed a usable token.
	apiAudience, oerr := resolveRequestedAudience(s.db, client, req.Audience, req.Resource)
	if oerr != nil {
		span.SetStatus(codes.Error, "audience not allowed")
		return nil, oerr
	}
	if apiAudience != "" {
		audience = apiAudience
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

	// A DPoP-bound access token carries token_type "DPoP" (RFC 9449 §6.1), so an
	// exact "access_token" comparison silently skipped revocation for exactly the
	// tokens a client took the trouble to sender-constrain.
	tokenType, _ := claims["token_type"].(string)
	if !jwt.IsAccessTokenType(tokenType) {
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

	client, oerr := authenticateOAuthClient(s.db, creds)
	if oerr != nil {
		return nil, oerr
	}

	// Try to validate as a JWT (access token or ID token).
	claims, err := oauthTokenValidateTokenWithContext(ctx, req.Token)
	if err == nil && claims != nil {
		// The authenticated caller used to be discarded, so this endpoint reported
		// active=true plus sub, scope and client_id for ANY token this server ever
		// signed — every tenant's tokens are signed with the same key, so tenant A
		// could introspect tenant B's token and read who it belongs to and what it
		// can do. RFC 7662 §2.2: a token the caller is not authorized to introspect
		// is reported as inactive, not as an error, so the answer is
		// indistinguishable from an unknown token.
		if !s.callerMayIntrospect(claims, client) {
			span.SetStatus(codes.Ok, "token inactive")
			return &OAuthIntrospectResponseDTO{Active: false}, nil
		}
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

	// Try as a refresh token. Same tenant scoping as above — the stored row
	// carries the tenant it was issued for.
	tokenHash := crypto.HashRefreshToken(req.Token)
	storedRT, lookupErr := s.refreshTokenRepo.FindByTokenHash(tokenHash)
	if lookupErr == nil && storedRT != nil && storedRT.IsActive() &&
		client != nil && storedRT.TenantID == client.TenantID {
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

// callerMayIntrospect reports whether the authenticated client is entitled to
// see this token's contents.
//
// A client may introspect its own tokens, and any token belonging to a client in
// its own tenant. Crossing the tenant boundary is refused: a shared signing key
// means a valid signature says nothing about which tenant a token came from, so
// the tenant of the token's client is the only binding available.
func (s *oauthTokenService) callerMayIntrospect(claims map[string]any, caller *Client) bool {
	if caller == nil {
		return false
	}
	tokenClientID, _ := claims["client_id"].(string)
	tokenClientID = strings.TrimSpace(tokenClientID)
	if tokenClientID == "" {
		// A token this server issued always carries client_id. One that does not
		// cannot be attributed to a tenant, so it is not introspectable.
		return false
	}
	if caller.Identifier != nil && tokenClientID == *caller.Identifier {
		return true
	}
	tokenClient, err := findActiveClientByIdentifier(s.db, tokenClientID)
	if err != nil || tokenClient == nil {
		return false
	}
	return tokenClient.TenantID != 0 && tokenClient.TenantID == caller.TenantID
}

func (s *oauthTokenService) SetSessionAuthContextResolver(r SessionAuthContextResolver) {
	s.sessionAuthResolver = r
}

// tokenAuthContext is the acr/amr/auth_time a token should assert.
type tokenAuthFacts struct {
	ACR      string
	AMR      []string
	AuthTime time.Time
}

// resolveSessionAuthContext reads the real authentication facts for the session
// this token is being minted from.
//
// The fallback is single-factor password, matching what the code asserted
// unconditionally before. It is only reached when there is no session (grants
// that have no browser session behind them), no resolver wired, or the session
// row cannot be read — and in the last case it is logged, because silently
// downgrading acr to 1 re-challenges a user who has already completed MFA.
func (s *oauthTokenService) resolveSessionAuthContext(ctx context.Context, sessionUUID *uuid.UUID) tokenAuthFacts {
	fallback := tokenAuthFacts{ACR: jwt.ACRLevel1, AMR: []string{jwt.AMRPassword}}
	if sessionUUID == nil || s.sessionAuthResolver == nil {
		return fallback
	}
	sessionCtx, err := s.sessionAuthResolver.ResolveSessionAuthContext(ctx, *sessionUUID)
	if err != nil {
		slog.Warn("could not read the session's authentication context; the token will assert single-factor password and step-up routes will re-challenge",
			"error", err)
		return fallback
	}
	if sessionCtx == nil {
		return fallback
	}
	facts := fallback
	if sessionCtx.ACR != "" {
		facts.ACR = sessionCtx.ACR
	}
	if len(sessionCtx.AMR) > 0 {
		facts.AMR = sessionCtx.AMR
	}
	facts.AuthTime = sessionCtx.AuthTime
	return facts
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// generateTokens creates an access token, an ID token, and — when
// issueRefreshToken is set and the scope allows it — a brand-new refresh token
// family.
//
// issueRefreshToken MUST be false on the refresh_token grant. That path owns
// rotation: it revokes the presented token and re-issues into the SAME family.
// Letting this helper also mint here inserted a second row in a NEW family on
// every refresh whose raw value was then discarded by the caller — unreachable
// rows that still counted against the per-client active-token limit and were
// written outside the rotation transaction.
// apiAudience, when non-empty, is the identifier of a registered API the client
// asked this token to be addressed to (already validated against client_apis by
// resolveRequestedAudience). It replaces the access token's `aud`; the ID token
// keeps aud = the client, because an ID token is always for the client (OIDC
// Core §2).
func (s *oauthTokenService) generateTokens(ctx context.Context, sub string, user *User, client *Client, scope string, nonce *string, dpopThumbprint string, issueRefreshToken bool, sessionUUID *uuid.UUID, apiAudience string) (*OAuthTokenResult, *apperror.OAuthError) {
	issuer := ""
	audience := ""
	identifier := ""
	providerID := tokenRealm(client)
	// The issuer identifies the authorization server, not the client — see
	// jwt.TokenIssuer. Stamping the client's domain made `iss` disagree with the
	// `issuer` in the discovery document, which a compliant RP must reject.
	issuer = jwt.TokenIssuerPtr(client.Domain)
	if client.Identifier != nil {
		audience = *client.Identifier
		identifier = *client.Identifier
	}
	if apiAudience != "" {
		audience = apiAudience
	}

	accessTokenOpts := s.clientAccessTokenOpts(client)
	if dpopThumbprint != "" {
		accessTokenOpts.DPoPThumbprint = dpopThumbprint
	}
	// Carry the session's REAL authentication facts. Hardcoding pwd/acr=1 threw
	// away what the session row two lines below already records, with two
	// consequences: a user who had just completed TOTP or a passkey got acr=1, so
	// every RequireStepUp route re-challenged them immediately after a full MFA
	// login; and the token ASSERTED amr:["pwd"] for magic-link, SMS and passkey
	// logins in which no password was ever entered — a false claim about how the
	// subject authenticated (RFC 8176, OIDC Core §2).
	authCtx := s.resolveSessionAuthContext(ctx, sessionUUID)
	accessTokenOpts.AMR = authCtx.AMR
	accessTokenOpts.ACR = authCtx.ACR
	// Stamp the originating browser session. This is what lets logout revoke a
	// single session: without a sid the OAuth token is unattributable and logout
	// can only revoke everything or nothing.
	if sessionUUID != nil {
		accessTokenOpts.SessionID = sessionUUID.String()
	}

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
	if idTokenParams == nil {
		idTokenParams = &jwt.IDTokenParams{}
	}
	idTokenParams.AMR = authCtx.AMR
	idTokenParams.ACR = authCtx.ACR
	// auth_time must be the authentication event, not this issuance, or an RP
	// enforcing max_age can never detect a stale session (OIDC Core §2). Zero
	// leaves the jwt layer's "now" fallback in place.
	idTokenParams.AuthTime = authCtx.AuthTime
	// at_hash binds the ID token to the access token delivered with it
	// (OIDC Core §3.1.3.6).
	idTokenParams.AccessToken = accessToken
	// Same session the access token above is stamped with, so a back-channel
	// logout token's sid resolves to something the RP actually holds.
	if sessionUUID != nil {
		idTokenParams.SessionID = sessionUUID.String()
	}

	idToken, err := oauthTokenGenerateIDTokenWithContext(ctx, sub, issuer, identifier, providerID, profile, nonceStr, idTokenParams)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Refresh tokens are only issued when offline_access scope is requested
	// (RFC 6749 §1.5) or for authorization_code grant with a valid DPoP binding.
	var rawRT string
	if issueRefreshToken && hasOfflineAccess(scope) {
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
			// RFC 9449 §5: bind the refresh token to the proofing key so a stolen
			// token is useless without the private key that minted it.
			DPoPJKT:         ptr.PtrOrNil(dpopThumbprint),
			UserSessionUUID: sessionUUID,
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
	identity, err := s.userIdentityRepo.FindByUserIDAndClientReachable(userID, clientID)
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
			// Stamp the tenant's opaque UUID, never the internal PK (least-disclosure
			// per RFC 9068). Resolver is a cached, ctx-agnostic lookup.
			if client != nil && client.TenantID > 0 {
				if s := shared.TenantUUIDStringByID(context.Background(), client.TenantID); s != "" {
					if params.ExtraClaims == nil {
						params.ExtraClaims = map[string]any{}
					}
					params.ExtraClaims["tenant_id"] = s
				}
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
