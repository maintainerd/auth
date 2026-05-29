package idp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	oidclib "github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	jwtlib "github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
	authEventService AuthEventService
}

func NewFederationService(
	db *gorm.DB,
	userRepo UserRepository,
	userIdentityRepo UserIdentityRepository,
	idpRepo IdentityProviderRepository,
	clientRepo ClientRepository,
	userRoleRepo UserRoleRepository,
	roleRepo RoleRepository,
	authEventService AuthEventService,
) FederationService {
	return &federationService{
		db:               db,
		userRepo:         userRepo,
		userIdentityRepo: userIdentityRepo,
		idpRepo:          idpRepo,
		clientRepo:       clientRepo,
		userRoleRepo:     userRoleRepo,
		roleRepo:         roleRepo,
		authEventService: authEventService,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Token exchange
// ──────────────────────────────────────────────────────────────────────────────

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

	var cfg OIDCProviderConfig
	if err := json.Unmarshal(idp.Config, &cfg); err != nil || cfg.Issuer == "" {
		return nil, apperror.NewValidation("identity provider is not configured for OIDC")
	}

	// 2. Validate the external ID token using OIDC discovery.
	claims, err := s.validateOIDCToken(ctx, cfg, req.ExternalToken)
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
		defaultIdentity, err := txUserIdentityRepo.FindByUserIDAndProvider(user.UserID, ProviderDefault)
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

	// 6. Log auth event.
	userID := user.UserID
	desc := fmt.Sprintf("external login via %s", idp.Provider)
	if isNew {
		desc = fmt.Sprintf("JIT-provisioned via %s", idp.Provider)
	}
	s.authEventService.Log(ctx, AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeTokenCreated,
		Severity:    AuthEventSeverityInfo,
		Result:      AuthEventResultSuccess,
		Description: ptr.Ptr(desc),
	})

	span.SetStatus(codes.Ok, "")
	return s.generateTokens(internalSub, user, client)
}

// ──────────────────────────────────────────────────────────────────────────────
// Identity linking / unlinking
// ──────────────────────────────────────────────────────────────────────────────

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

	var cfg OIDCProviderConfig
	if err := json.Unmarshal(idp.Config, &cfg); err != nil || cfg.Issuer == "" {
		return nil, apperror.NewValidation("identity provider is not configured for OIDC")
	}

	claims, err := s.validateOIDCToken(ctx, cfg, req.ExternalToken)
	if err != nil {
		return nil, apperror.NewUnauthorized("external token validation failed: " + err.Error())
	}

	externalSub := stringClaim(claims, "sub")
	if externalSub == "" {
		return nil, apperror.NewValidation("external token missing 'sub' claim")
	}

	// Ensure this external identity isn't already claimed by another user.
	existing, _ := s.userIdentityRepo.FindByProviderAndSub(idp.Provider, externalSub)
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

	s.authEventService.Log(ctx, AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeTokenCreated,
		Severity:    AuthEventSeverityInfo,
		Result:      AuthEventResultSuccess,
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
	if target.Provider == ProviderDefault {
		return apperror.NewValidation("the built-in identity cannot be unlinked")
	}

	if err := s.userIdentityRepo.DeleteByID(target.UserIdentityID); err != nil {
		return apperror.NewInternal("failed to unlink identity", err)
	}

	s.authEventService.Log(ctx, AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    AuthEventCategoryAuthn,
		EventType:   AuthEventTypeTokenCreated,
		Severity:    AuthEventSeverityInfo,
		Result:      AuthEventResultSuccess,
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

// ──────────────────────────────────────────────────────────────────────────────
// Home Realm Discovery
// ──────────────────────────────────────────────────────────────────────────────

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
		if idp.ProviderType != IDPTypeSocial {
			continue
		}
		var cfg OIDCProviderConfig
		if err := json.Unmarshal(idp.Config, &cfg); err != nil {
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

// ──────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────────────────────────────────

// validateOIDCToken fetches the provider's OIDC discovery doc, verifies the
// token's signature + standard claims, and returns the raw claims map.
func (s *federationService) validateOIDCToken(ctx context.Context, cfg OIDCProviderConfig, rawToken string) (map[string]interface{}, error) {
	provider, err := oidclib.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed for %s: %w", cfg.Issuer, err)
	}

	verifierCfg := &oidclib.Config{ClientID: cfg.ClientID}
	if cfg.ClientID == "" {
		verifierCfg.SkipClientIDCheck = true
	}
	verifier := provider.Verifier(verifierCfg)

	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("claim extraction failed: %w", err)
	}
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

	// Try to match an existing user by email to merge identities rather than duplicate.
	var user *User
	var isNew bool
	if email != "" {
		existing, _ := txUserRepo.FindByEmail(email)
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
			IsEmailVerified: stringClaim2(meta.Email) != "",
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
	defaultIdentity, _ := txUserIdentityRepo.FindByUserIDAndProvider(user.UserID, ProviderDefault)
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
			Provider:           ProviderDefault,
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
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return tx.Model(identity).Update("metadata", datatypes.JSON(metaJSON)).Error
}

func (s *federationService) generateTokens(sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	accessToken, err := jwtlib.GenerateAccessToken(
		sub,
		"openid profile email",
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		client.IdentityProvider.Identifier,
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
	idToken, err := jwtlib.GenerateIDToken(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", nil)
	if err != nil {
		return nil, apperror.NewInternal("id token generation failed", err)
	}

	refreshToken, err := jwtlib.GenerateRefreshToken(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
	if err != nil {
		return nil, apperror.NewInternal("refresh token generation failed", err)
	}

	return &LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
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
	return roleRepo.FindByNameAndTenantID(RoleRegistered, tenantID)
}

// ──────────────────────────────────────────────────────────────────────────────
// Pure helpers
// ──────────────────────────────────────────────────────────────────────────────

func stringClaim(claims map[string]interface{}, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func stringClaim2(s string) string { return s }

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
		Email:      stringClaim(claims, claimName("email")),
		Name:       stringClaim(claims, claimName("name")),
		GivenName:  stringClaim(claims, claimName("given_name")),
		FamilyName: stringClaim(claims, claimName("family_name")),
		Picture:    stringClaim(claims, claimName("picture")),
		Locale:     stringClaim(claims, claimName("locale")),
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
		IsDefault:    ui.Provider == ProviderDefault,
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
