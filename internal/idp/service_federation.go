package idp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	oidclib "github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/platform/apperror"
	jwtlib "github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const DefaultOIDCUserinfoEndpoint = "/userinfo"

var (
	idpValidateOIDCToken = (*federationService).validateOIDCToken
	idpOAuth2Exchange    = func(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
		return cfg.Exchange(ctx, code)
	}
	idpOAuth2GetUserinfo = func(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token, url string) (*http.Response, error) {
		return cfg.Client(ctx, tok).Get(url)
	}
	idpGenerateAccessTokenWithOptionsContext = jwtlib.GenerateAccessTokenWithOptionsContext
	idpGenerateIDTokenWithContext            = jwtlib.GenerateIDTokenWithContext
	idpGenerateRefreshTokenWithContext       = jwtlib.GenerateRefreshTokenWithContext
)

// FederationService handles external identity provider flows:
// OIDC token exchange, JIT user provisioning, identity linking/unlinking,
// and home-realm discovery.
type FederationService interface {
	// ExchangeExternalToken validates an external OIDC ID token, finds or
	// provisions the user, and returns our own JWT. This is the entry point
	// for frontends that authenticate entirely with an upstream provider
	// (Google, Cognito, Auth0, …) and want to check permissions via our system.
	ExchangeExternalToken(ctx context.Context, req FederationTokenRequestDTO) (*LoginResponseDTO, error)

	// ExchangeOAuth2Code handles the OAuth2 authorization code flow for
	// generic OAuth2 providers. It exchanges the code for an access token,
	// fetches user info from the provider's userinfo endpoint, and
	// provisions/links the user.
	ExchangeOAuth2Code(ctx context.Context, req FederationOAuth2CallbackDTO) (*LoginResponseDTO, error)

	// LinkIdentity attaches an external provider identity to an already
	// authenticated user. If the identity is already linked to a different user,
	// the call fails.
	LinkIdentity(ctx context.Context, userID int64, req LinkIdentityRequestDTO) (*IdentityDTO, error)

	// UnlinkIdentity removes an external provider identity from a user.
	// The built-in "default" identity cannot be removed.
	UnlinkIdentity(ctx context.Context, userID int64, identityUUIDStr string) error

	// GetUserIdentities returns all identities (builtin + external) linked to a user.
	GetUserIdentities(ctx context.Context, userID int64) ([]IdentityDTO, error)

	// HomeRealmDiscovery returns the identity provider to use for the given
	// email address, based on the email-domain list stored in each provider's config.
	HomeRealmDiscovery(ctx context.Context, tenantID int64, email string) (*HRDResponseDTO, error)
}

type federationService struct {
	db               *gorm.DB
	userRepo         UserRepository
	userIdentityRepo UserIdentityRepository
	idpRepo          IdentityProviderRepository
	clientRepo       ClientRepository
	userRoleRepo     UserRoleRepository
	roleRepo         RoleRepository
	authEventService authevent.AuthEventService
	sessionService   authn.SessionService
}

func NewFederationService(
	db *gorm.DB,
	userRepo UserRepository,
	userIdentityRepo UserIdentityRepository,
	idpRepo IdentityProviderRepository,
	clientRepo ClientRepository,
	userRoleRepo UserRoleRepository,
	roleRepo RoleRepository,
	authEventService authevent.AuthEventService,
	sessionService ...authn.SessionService,
) FederationService {
	var sessions authn.SessionService
	if len(sessionService) > 0 {
		sessions = sessionService[0]
	}
	return &federationService{
		db:               db,
		userRepo:         userRepo,
		userIdentityRepo: userIdentityRepo,
		idpRepo:          idpRepo,
		clientRepo:       clientRepo,
		userRoleRepo:     userRoleRepo,
		roleRepo:         roleRepo,
		authEventService: authEventService,
		sessionService:   sessions,
	}
}

// Token exchange

func (s *federationService) ExchangeExternalToken(ctx context.Context, req FederationTokenRequestDTO) (*LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "federation.exchange_external_token")
	defer span.End()
	span.SetAttributes(attribute.String("provider_identifier", req.ProviderIdentifier))

	// 1. Look up the configured external IDP.
	idp, err := s.idpRepo.FindByIdentifier(req.ProviderIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	decCfg := decryptIdpConfig(idp.Config)
	var cfg OIDCProviderConfig
	if err := json.Unmarshal(decCfg, &cfg); err != nil || cfg.Issuer == "" {
		return nil, apperror.NewValidation("identity provider is not configured for OIDC")
	}

	// 2. Validate the external ID token using OIDC discovery.
	claims, err := idpValidateOIDCToken(s, ctx, cfg, req.ExternalToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "oidc validation failed")
		return nil, apperror.NewUnauthorized("external token validation failed: " + err.Error())
	}

	externalSub := stringClaim(claims, "sub")
	if externalSub == "" {
		return nil, apperror.NewValidation("external token missing 'sub' claim")
	}
	span.SetAttributes(attribute.String("external_sub", externalSub))

	// 3. Build metadata from claims with attribute mapping.
	meta := extractMetadata(claims, cfg.AttributeMapping)
	email := meta.Email
	if email == "" {
		email = stringClaim(claims, "email")
	}

	// 4. Find or provision the user.
	var user *User
	var internalSub string
	var isNew bool

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		// Try to find by the external identity (provider + sub).
		existing, err := txUserIdentityRepo.FindByProviderAndSub(idp.Provider, externalSub)
		if err != nil {
			return apperror.NewInternal("identity lookup failed", err)
		}

		if existing != nil {
			// Known user — load and refresh metadata.
			user, err = txUserRepo.FindByID(existing.UserID)
			if err != nil || user == nil {
				return apperror.NewInternal("user not found for existing identity", err)
			}
			_ = s.refreshMetadata(tx, existing, meta)
		} else {
			// Unknown external sub. Optionally JIT-provision.
			if !cfg.AllowJITProvisioning {
				return apperror.NewUnauthorized("user not found and JIT provisioning is disabled for this provider")
			}
			var provisionErr error
			user, isNew, provisionErr = s.provisionUser(ctx, tx, idp, externalSub, email, meta)
			if provisionErr != nil {
				return provisionErr
			}
		}

		// Retrieve the internal (default) identity sub for JWT generation.
		defaultIdentity, err := txUserIdentityRepo.FindByUserIDAndProvider(user.UserID, shared.ProviderDefault)
		if err != nil {
			return apperror.NewInternal("default identity lookup failed", err)
		}
		if defaultIdentity == nil {
			return apperror.NewInternal("user has no default identity", nil)
		}
		internalSub = defaultIdentity.Sub
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 5. Find the client used to generate our tokens.
	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(req.ClientID, req.ProviderIdentifier)
	if err != nil || client == nil {
		// Fallback: any client associated with the tenant's default IDP.
		defaultIDP, _ := s.idpRepo.FindDefaultByTenantID(idp.TenantID)
		if defaultIDP != nil {
			client, _ = s.clientRepo.FindByClientIDAndIdentityProvider(req.ClientID, defaultIDP.Identifier)
		}
	}
	if client == nil {
		return nil, apperror.NewNotFound("client not found for this provider")
	}

	// 6. authevent.Log auth event.
	userID := user.UserID
	desc := fmt.Sprintf("external login via %s", idp.Provider)
	if isNew {
		desc = fmt.Sprintf("JIT-provisioned via %s", idp.Provider)
	}
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(desc),
	})

	span.SetStatus(codes.Ok, "")
	return s.generateTokens(ctx, internalSub, user, client)
}

// Generic OAuth2 authorization code flow

func (s *federationService) ExchangeOAuth2Code(ctx context.Context, req FederationOAuth2CallbackDTO) (*LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "federation.exchange_oauth2_code")
	defer span.End()
	span.SetAttributes(attribute.String("provider_identifier", req.ProviderIdentifier))

	idp, err := s.idpRepo.FindByIdentifier(req.ProviderIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	decCfg := decryptIdpConfig(idp.Config)
	var cfg OIDCProviderConfig
	if err := json.Unmarshal(decCfg, &cfg); err != nil {
		return nil, apperror.NewValidation("identity provider configuration is invalid")
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, apperror.NewValidation("identity provider missing OAuth2 client credentials")
	}

	userinfoURL := cfg.UserinfoEndpoint
	if userinfoURL == "" && cfg.Issuer != "" {
		userinfoURL = strings.TrimRight(cfg.Issuer, "/") + DefaultOIDCUserinfoEndpoint
	}
	if userinfoURL == "" {
		return nil, apperror.NewValidation("identity provider missing userinfo endpoint")
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  req.RedirectURI,
		Endpoint: oauth2.Endpoint{
			TokenURL:  strings.TrimRight(cfg.Issuer, "/") + "/oauth/token",
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
		Scopes: cfg.Scopes,
	}

	tok, err := idpOAuth2Exchange(ctx, oauth2Cfg, req.Code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "oauth2 code exchange failed")
		return nil, apperror.NewUnauthorized("failed to exchange authorization code: " + err.Error())
	}

	resp, err := idpOAuth2GetUserinfo(ctx, oauth2Cfg, tok, userinfoURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "userinfo fetch failed")
		return nil, apperror.NewUnauthorized("failed to fetch user info from provider")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperror.NewUnauthorized("failed to read user info response")
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, apperror.NewUnauthorized("failed to parse user info response")
	}

	externalSub := stringClaim(claims, "sub")
	if externalSub == "" {
		externalSub = stringClaim(claims, "id")
	}
	if externalSub == "" {
		return nil, apperror.NewValidation("external token missing user identifier")
	}
	span.SetAttributes(attribute.String("external_sub", externalSub))

	meta := extractMetadata(claims, cfg.AttributeMapping)
	email := meta.Email
	if email == "" {
		email = stringClaim(claims, "email")
	}

	var user *User
	var internalSub string

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		existing, txErr := txUserIdentityRepo.FindByProviderAndSub(idp.Provider, externalSub)
		if txErr != nil {
			return apperror.NewInternal("identity lookup failed", txErr)
		}

		if existing != nil {
			user, txErr = txUserRepo.FindByID(existing.UserID)
			if txErr != nil || user == nil {
				return apperror.NewInternal("user lookup failed", txErr)
			}
			_ = s.refreshMetadata(tx, existing, meta)
		} else {
			var isNew bool
			user, isNew, txErr = s.provisionUser(ctx, tx, idp, externalSub, email, meta)
			if txErr != nil {
				return txErr
			}
			_ = isNew
		}

		identity, txErr := txUserIdentityRepo.FindByUserIDAndProvider(user.UserID, idp.Provider)
		if txErr != nil || identity == nil {
			return apperror.NewInternal("identity resolution failed", txErr)
		}
		internalSub = identity.Sub
		return nil
	})
	if err != nil {
		return nil, err
	}

	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(req.ClientID, idp.Identifier)
	if err != nil || client == nil {
		return nil, apperror.NewNotFound("client not found")
	}

	return s.generateTokens(ctx, internalSub, user, client)
}

// Identity linking / unlinking

func (s *federationService) LinkIdentity(ctx context.Context, userID int64, req LinkIdentityRequestDTO) (*IdentityDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "federation.link_identity")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("user.id", userID),
		attribute.String("provider_identifier", req.ProviderIdentifier),
	)

	idp, err := s.idpRepo.FindByIdentifier(req.ProviderIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}

	decCfg := decryptIdpConfig(idp.Config)
	var cfg OIDCProviderConfig
	if err := json.Unmarshal(decCfg, &cfg); err != nil || cfg.Issuer == "" {
		return nil, apperror.NewValidation("identity provider is not configured for OIDC")
	}

	claims, err := idpValidateOIDCToken(s, ctx, cfg, req.ExternalToken)
	if err != nil {
		return nil, apperror.NewUnauthorized("external token validation failed: " + err.Error())
	}

	externalSub := stringClaim(claims, "sub")
	if externalSub == "" {
		return nil, apperror.NewValidation("external token missing 'sub' claim")
	}

	// Ensure this external identity isn't already claimed by another user.
	existing, err := s.userIdentityRepo.FindByProviderAndSub(idp.Provider, externalSub)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.UserID != userID {
			return nil, apperror.NewValidation("this external identity is already linked to a different account")
		}
		// Already linked to this user — idempotent success.
		return identityToDTO(existing), nil
	}

	meta := extractMetadata(claims, cfg.AttributeMapping)
	metaJSON, _ := json.Marshal(meta)
	idpID := idp.IdentityProviderID

	identity := &UserIdentity{
		UserID:             userID,
		TenantID:           idp.TenantID,
		ClientID:           0, // no specific client context for linked identities
		IdentityProviderID: &idpID,
		Provider:           idp.Provider,
		Sub:                externalSub,
		Metadata:           datatypes.JSON(metaJSON),
	}

	created, err := s.userIdentityRepo.Create(identity)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to link identity", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("linked external identity: %s", idp.Provider)),
	})

	span.SetStatus(codes.Ok, "")
	return identityToDTO(created), nil
}

func (s *federationService) UnlinkIdentity(ctx context.Context, userID int64, identityUUIDStr string) error {
	_, span := otel.Tracer("service").Start(ctx, "federation.unlink_identity")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("user.id", userID),
		attribute.String("identity.uuid", identityUUIDStr),
	)

	identities, err := s.userIdentityRepo.FindByUserID(userID)
	if err != nil {
		return apperror.NewInternal("identity lookup failed", err)
	}

	var target *UserIdentity
	for i := range identities {
		if identities[i].UserIdentityUUID.String() == identityUUIDStr {
			target = &identities[i]
			break
		}
	}
	if target == nil {
		return apperror.NewNotFound("identity not found")
	}
	if target.Provider == shared.ProviderDefault {
		return apperror.NewValidation("the built-in identity cannot be unlinked")
	}

	if err := s.userIdentityRepo.DeleteByID(target.UserIdentityID); err != nil {
		return apperror.NewInternal("failed to unlink identity", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("unlinked external identity: %s", target.Provider)),
	})

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *federationService) GetUserIdentities(ctx context.Context, userID int64) ([]IdentityDTO, error) {
	identities, err := s.userIdentityRepo.FindByUserID(userID)
	if err != nil {
		return nil, apperror.NewInternal("identity lookup failed", err)
	}
	result := make([]IdentityDTO, len(identities))
	for i := range identities {
		result[i] = *identityToDTO(&identities[i])
	}
	return result, nil
}

// Home Realm Discovery

func (s *federationService) HomeRealmDiscovery(ctx context.Context, tenantID int64, email string) (*HRDResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "federation.hrd")
	defer span.End()

	domain := emailDomain(email)
	if domain == "" {
		return nil, apperror.NewValidation("invalid email address")
	}

	idps, err := s.idpRepo.FindAllByTenantID(tenantID)
	if err != nil {
		return nil, apperror.NewInternal("provider lookup failed", err)
	}

	for _, idp := range idps {
		decCfg := decryptIdpConfig(idp.Config)
		var cfg OIDCProviderConfig
		if err := json.Unmarshal(decCfg, &cfg); err != nil {
			continue
		}
		for _, d := range cfg.EmailDomains {
			if strings.EqualFold(d, domain) {
				idpCopy := idp
				span.SetStatus(codes.Ok, "")
				return hrdResponseFrom(&idpCopy), nil
			}
		}
	}

	// No external provider matched — return the default (maintainerd) IDP.
	defaultIDP, _ := s.idpRepo.FindDefaultByTenantID(tenantID)
	if defaultIDP == nil {
		return nil, apperror.NewNotFound("no identity provider found for this tenant")
	}
	span.SetStatus(codes.Ok, "")
	return hrdResponseFrom(defaultIDP), nil
}

// Internal helpers

// validateOIDCToken fetches the provider's OIDC discovery doc, verifies the
// token's signature + standard claims, and returns the raw claims map.
func (s *federationService) validateOIDCToken(ctx context.Context, cfg OIDCProviderConfig, rawToken string) (map[string]interface{}, error) {
	provider, err := oidclib.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed for %s: %w", cfg.Issuer, err)
	}

	verifierCfg := &oidclib.Config{ClientID: cfg.ClientID}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OIDC client_id is required")
	}
	verifier := provider.Verifier(verifierCfg)

	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	var claims map[string]interface{}
	_ = idToken.Claims(&claims)
	return claims, nil
}

// provisionUser creates a new User + default identity + external identity for
// a first-time external login. Runs inside the caller's transaction.
func (s *federationService) provisionUser(
	ctx context.Context,
	tx *gorm.DB,
	idp *IdentityProvider,
	externalSub string,
	email string,
	meta IdentityMetadata,
) (*User, bool, error) {
	txUserRepo := s.userRepo.WithTx(tx)
	txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)

	// Only verified upstream emails may be used to merge identities. An
	// unverified email claim is profile data, not proof of account ownership.
	var user *User
	var isNew bool
	if email != "" && meta.EmailVerified {
		existing, err := txUserRepo.FindByEmailAndTenantID(email, idp.TenantID)
		if err != nil {
			return nil, false, apperror.NewInternal("email lookup failed", err)
		}
		if existing != nil {
			user = existing
		}
	}

	if user == nil {
		// Create a new user from the external profile.
		username := deriveUsername(meta, email)
		newUser := &User{
			Email:           email,
			Username:        username,
			IsEmailVerified: meta.EmailVerified,
		}
		created, err := txUserRepo.Create(newUser)
		if err != nil || created == nil {
			return nil, false, apperror.NewInternal("failed to provision user", err)
		}
		user = created
		isNew = true

		// Assign default role if available.
		if defaultRole, _ := s.findDefaultRole(s.roleRepo.WithTx(tx), idp.TenantID); defaultRole != nil {
			_ = tx.Create(&UserRole{UserID: user.UserID, RoleID: defaultRole.RoleID}).Error
		}
	}

	// Create the default (maintainerd) identity if it doesn't exist yet.
	// This ensures our RBAC system always has a stable sub for the user.
	defaultIdentity, _ := txUserIdentityRepo.FindByUserIDAndProvider(user.UserID, shared.ProviderDefault)
	if defaultIdentity == nil {
		defaultIDP, _ := s.idpRepo.FindDefaultByTenantID(idp.TenantID)
		var defaultIDPID *int64
		if defaultIDP != nil {
			defaultIDPID = &defaultIDP.IdentityProviderID
		}
		defIdentity := &UserIdentity{
			TenantID:           idp.TenantID,
			UserID:             user.UserID,
			ClientID:           0,
			IdentityProviderID: defaultIDPID,
			Provider:           shared.ProviderDefault,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}
		_, _ = txUserIdentityRepo.Create(defIdentity)
	}

	// Create the external identity.
	idpID := idp.IdentityProviderID
	metaJSON, _ := json.Marshal(meta)
	extIdentity := &UserIdentity{
		TenantID:           idp.TenantID,
		UserID:             user.UserID,
		ClientID:           0,
		IdentityProviderID: &idpID,
		Provider:           idp.Provider,
		Sub:                externalSub,
		Metadata:           datatypes.JSON(metaJSON),
	}
	_, err := txUserIdentityRepo.Create(extIdentity)
	if err != nil {
		return nil, false, apperror.NewInternal("failed to create external identity", err)
	}

	return user, isNew, nil
}

func (s *federationService) refreshMetadata(tx *gorm.DB, identity *UserIdentity, meta IdentityMetadata) error {
	metaJSON, _ := json.Marshal(meta)
	return tx.Model(&UserIdentity{}).
		Where("user_identity_id = ?", identity.UserIdentityID).
		Update("metadata", datatypes.JSON(metaJSON)).Error
}

func (s *federationService) generateTokens(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	var sessionID string
	if s.sessionService != nil {
		if err := s.sessionService.EnforceConcurrentLimit(ctx, user.UserUUID, user.UserID); err != nil {
			return nil, apperror.NewInternal("session limit enforcement failed", err)
		}
		sess, err := s.sessionService.CreateSession(ctx, user.UserID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx))
		if err != nil {
			return nil, apperror.NewInternal("session creation failed", err)
		}
		sessionID = sess.UserTokenUUID.String()
	}

	accessToken, err := idpGenerateAccessTokenWithOptionsContext(
		ctx,
		sub,
		shared.DefaultTokenScope,
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		client.IdentityProvider.Identifier,
		&jwtlib.AccessTokenOptions{
			AMR:       []string{jwtlib.AMRMFA},
			ACR:       jwtlib.ACRLevel1,
			SessionID: sessionID,
		},
	)
	if err != nil {
		return nil, apperror.NewInternal("access token generation failed", err)
	}

	profile := &jwtlib.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}
	idToken, err := idpGenerateIDTokenWithContext(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", &jwtlib.IDTokenParams{
		RequestedScopes: strings.Fields(shared.DefaultTokenScope),
		AMR:             []string{jwtlib.AMRMFA},
		ACR:             jwtlib.ACRLevel1,
	})
	if err != nil {
		return nil, apperror.NewInternal("id token generation failed", err)
	}

	refreshToken, err := idpGenerateRefreshTokenWithContext(ctx, sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
	if err != nil {
		return nil, apperror.NewInternal("refresh token generation failed", err)
	}

	resp := &LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    shared.DefaultAccessTokenExpiresIn,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}
	if sessionID != "" {
		resp.SessionID = &sessionID
	}
	return resp, nil
}

// findDefaultRole mirrors the logic in registerService to locate the tenant's
// default role without requiring a separate repository method.
func (s *federationService) findDefaultRole(roleRepo RoleRepository, tenantID int64) (*Role, error) {
	isDefault := true
	result, err := roleRepo.FindPaginated(RoleRepositoryGetFilter{
		IsDefault: &isDefault,
		TenantID:  tenantID,
		Page:      1,
		Limit:     1,
	})
	if err == nil && len(result.Data) > 0 {
		return &result.Data[0], nil
	}
	return roleRepo.FindByNameAndTenantID(shared.RoleRegistered, tenantID)
}

// Pure helpers

func stringClaim(claims map[string]interface{}, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func boolClaim(claims map[string]interface{}, key string) bool {
	v, ok := claims[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func extractMetadata(claims map[string]interface{}, mapping map[string]string) IdentityMetadata {
	claimName := func(field string) string {
		if mapping != nil {
			if v, ok := mapping[field]; ok {
				return v
			}
		}
		return field
	}
	return IdentityMetadata{
		Email:         stringClaim(claims, claimName("email")),
		EmailVerified: boolClaim(claims, claimName("email_verified")),
		Name:          stringClaim(claims, claimName("name")),
		GivenName:     stringClaim(claims, claimName("given_name")),
		FamilyName:    stringClaim(claims, claimName("family_name")),
		Picture:       stringClaim(claims, claimName("picture")),
		Locale:        stringClaim(claims, claimName("locale")),
	}
}

func deriveUsername(meta IdentityMetadata, email string) string {
	if meta.Name != "" {
		return strings.ReplaceAll(strings.ToLower(meta.Name), " ", "_")
	}
	if email != "" {
		parts := strings.SplitN(email, "@", 2)
		return parts[0]
	}
	return "user_" + uuid.New().String()[:8]
}

func emailDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[1] == "" {
		return ""
	}
	return strings.ToLower(parts[1])
}

func identityToDTO(ui *UserIdentity) *IdentityDTO {
	d := &IdentityDTO{
		IdentityUUID: ui.UserIdentityUUID.String(),
		Provider:     ui.Provider,
		Sub:          ui.Sub,
		IsDefault:    ui.Provider == shared.ProviderDefault,
		CreatedAt:    ui.CreatedAt.Format(time.RFC3339),
	}

	var meta IdentityMetadata
	if len(ui.Metadata) > 0 {
		_ = json.Unmarshal(ui.Metadata, &meta)
	}
	if meta.Email != "" {
		d.Email = &meta.Email
	}
	if meta.Name != "" {
		d.Name = &meta.Name
	}
	if meta.Picture != "" {
		d.Picture = &meta.Picture
	}
	return d
}

func hrdResponseFrom(idp *IdentityProvider) *HRDResponseDTO {
	return &HRDResponseDTO{
		ProviderIdentifier: idp.Identifier,
		Provider:           idp.Provider,
		DisplayName:        idp.DisplayName,
	}
}
