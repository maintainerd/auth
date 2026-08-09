package idp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	oidclib "github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	jwtlib "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const DefaultOIDCUserinfoEndpoint = "/userinfo"

// oidcClockSkewLeeway is the allowed clock skew when verifying upstream provider
// id_tokens. go-oidc's expiry check (t.Expiry.Before(now)) has no built-in
// tolerance, so a small clock difference between us and the IdP would spuriously
// reject freshly-issued tokens. We shift the verifier's notion of "now" slightly
// into the past via oidc.Config.Now, which the library supports natively. This
// only tolerates a bounded skew — it never disables the expiry check.
const oidcClockSkewLeeway = 1 * time.Minute

// oidcSkewedNow returns the clock function passed to oidc.Config.Now so token
// verification tolerates a small clock skew without weakening expiry enforcement.
func oidcSkewedNow() func() time.Time {
	return func() time.Time { return time.Now().Add(-oidcClockSkewLeeway) }
}

// errEmailCollision is returned by provisionUser when a verified email already
// belongs to an existing user and the account-link service is wired. The
// ResolveBrokerUser caller catches it and creates a confirmation request.
type errEmailCollision struct {
	tenantID           int64
	existingUserID     int64
	identityProviderID int64
	providerName       string
	providerSub        string
	providerEmail      string
	providerClaims     []byte
}

func (e *errEmailCollision) Error() string {
	return "account link required: email collision detected"
}

// handleEmailCollision converts an errEmailCollision surfaced from provisionUser
// into a caller-appropriate error for the direct-token federation flows
// (ExchangeExternalToken, ExchangeOAuth2Code, HandleSAMLResponse). Those flows
// return tokens directly and have no redirect surface for the account-link
// confirmation UI, so they must NOT silently merge. When the account-link
// service is wired we still create the pending confirmation request (so the user
// can later confirm and link); either way we return a Conflict so the collision
// is surfaced for explicit resolution rather than silently linked. Returns nil
// when err is not a collision (the caller then handles the original error).
func (s *federationService) handleEmailCollision(ctx context.Context, err error) error {
	var collision *errEmailCollision
	if !errors.As(err, &collision) {
		return nil
	}
	if s.accountLinkSvc != nil {
		if _, initErr := s.accountLinkSvc.Initiate(ctx, authn.InitiateAccountLinkInput{
			TenantID:           collision.tenantID,
			ExistingUserID:     collision.existingUserID,
			IdentityProviderID: collision.identityProviderID,
			ProviderName:       collision.providerName,
			ProviderSubject:    collision.providerSub,
			ProviderEmail:      collision.providerEmail,
			ProviderClaims:     collision.providerClaims,
			IPAddress:          middleware.ClientIPFromContext(ctx),
		}); initErr != nil {
			return initErr
		}
	}
	return apperror.NewConflict("this email is already associated with an existing account; sign in and link this provider to continue")
}

var (
	errIdentityCreatedConcurrently = errors.New("external identity was created concurrently")

	idpValidateOIDCToken = (*federationService).validateOIDCToken
	idpOAuth2Exchange    = func(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
		octx := context.WithValue(ctx, oauth2.HTTPClient, idpHTTPClientFactory())
		return cfg.Exchange(octx, code)
	}
	idpOAuth2GetUserinfo = func(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token, url string) (*http.Response, error) {
		octx := context.WithValue(ctx, oauth2.HTTPClient, idpHTTPClientFactory())
		return cfg.Client(octx, tok).Get(url)
	}
	idpOAuth2ExchangeWithPKCE = func(ctx context.Context, cfg *oauth2.Config, code, verifier string) (*oauth2.Token, error) {
		octx := context.WithValue(ctx, oauth2.HTTPClient, idpHTTPClientFactory())
		return cfg.Exchange(octx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	}
	// idpOIDCDiscover resolves a provider's authorize/token endpoints via OIDC
	// discovery. It is a var so tests can stub it without network access.
	idpOIDCDiscover = func(ctx context.Context, issuer string) (authorize, token string, err error) {
		octx := oidclib.ClientContext(ctx, idpHTTPClientFactory())
		provider, perr := oidclib.NewProvider(octx, issuer)
		if perr != nil {
			return "", "", perr
		}
		ep := provider.Endpoint()
		return ep.AuthURL, ep.TokenURL, nil
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

	// AdminUnlinkIdentity removes an external provider identity from another
	// user on behalf of a tenant admin. The target user is resolved by UUID and
	// strictly scoped to the admin's tenant (a cross-tenant target is reported
	// as NotFound to avoid leaking existence); the actor recorded in the audit
	// trail is the admin (actorUserID), not the target user. The built-in
	// "default" identity cannot be removed.
	AdminUnlinkIdentity(ctx context.Context, tenantID int64, actorUserID int64, userUUID uuid.UUID, identityUUID string) error

	// GetUserIdentities returns all identities (builtin + external) linked to a user.
	GetUserIdentities(ctx context.Context, userID int64) ([]IdentityDTO, error)

	// HomeRealmDiscovery returns the identity provider to use for the given
	// email address, based on the email-domain list stored in each provider's config.
	HomeRealmDiscovery(ctx context.Context, tenantID int64, email string) (*HRDResponseDTO, error)

	// HomeRealmDiscoveryByClient resolves the tenant from the public client_id and
	// then performs home-realm discovery. This is the public-surface entry point —
	// the public API never accepts tenant_id, only client_id.
	HomeRealmDiscoveryByClient(ctx context.Context, clientID string, email string) (*HRDResponseDTO, error)

	// ResolveBrokerProvider returns the upstream OAuth2 authorize parameters for a
	// brokered identity provider (its authorization endpoint, client_id, and
	// scopes), decrypting the provider config and resolving the endpoint via the
	// explicit authorization_endpoint or OIDC discovery. No secrets are returned.
	ResolveBrokerProvider(ctx context.Context, idpIdentifier string) (*BrokerProviderInfo, error)

	// SetIdentityLinkStore wires the in-flight account-link request store.
	SetIdentityLinkStore(store IdentityLinkRequestStore)

	// StartIdentityLink begins attaching a provider identity to an already
	// signed-in account; CompleteIdentityLink finishes it. Neither issues a
	// session — see service_identity_link.go.
	StartIdentityLink(ctx context.Context, userID, tenantID, clientID int64, providerIdentifier, redirectURI string) (*StartIdentityLinkResult, error)
	CompleteIdentityLink(ctx context.Context, userID int64, state, code, redirectURI string) (*IdentityDTO, error)

	// ResolveBrokerUser exchanges an upstream provider authorization code (with
	// PKCE verifier), validates the returned id_token (nonce-checked when
	// present), provisions the user if needed, and returns the authenticated
	// maintainerd user and their identity sub. It does NOT mint tokens — the
	// caller (the broker flow) uses the resolved user to issue its own
	// authorization code.
	ResolveBrokerUser(ctx context.Context, idpID int64, code, pkceVerifier, nonce, redirectURI string, clientID int64) (*BrokerResolvedUser, error)

	// TestConnection probes the unsaved IdP configuration: OIDC discovery and
	// JWKS endpoint validation. Reuses the SSRF-safe idpHTTPClient so external
	// probes cannot reach local network resources.
	TestConnection(ctx context.Context, req TestConnectionRequestDTO) (*TestConnectionResultDTO, error)

	// InitiateSAMLSSO generates a SAML AuthnRequest and returns the IdP redirect URL.
	// The returned URL includes the SAMLRequest query parameter (deflate+base64) and
	// a signed RelayState that carries the client context back to the ACS endpoint.
	InitiateSAMLSSO(ctx context.Context, in SAMLInitiateInput) (*SAMLInitiateResult, error)

	// HandleSAMLResponse validates the IdP-POST SAML Response, provisions or
	// authenticates the user, and returns a short-lived exchange code that the
	// frontend can use to obtain tokens without putting them in the URL.
	HandleSAMLResponse(ctx context.Context, r *http.Request, relayState string) (*SAMLCallbackResult, error)

	// ExchangeSAMLCode redeems a one-time SAML exchange code (issued by HandleSAMLResponse)
	// and returns the full login response (access token, refresh token, id token).
	ExchangeSAMLCode(ctx context.Context, code string) (*LoginResponseDTO, error)

	// SAMLMetadata returns the SP metadata XML for the IdP identified by identifier.
	// IdPs import this XML to configure the trust relationship.
	SAMLMetadata(ctx context.Context, identifier string) ([]byte, error)

	// InitiateSAMLLogout starts SP-initiated SAML Single Logout: it revokes the
	// subject's local sessions and returns the IdP SLO URL (SAMLRequest + signed
	// RelayState) the browser should follow.
	InitiateSAMLLogout(ctx context.Context, in SAMLLogoutInitiateInput) (*SAMLLogoutInitiateResult, error)

	// HandleSAMLSingleLogout serves the SLO endpoint published in our SP
	// metadata. It consumes the IdP's LogoutResponse (completing an SP-initiated
	// logout) and honours an IdP-initiated LogoutRequest, answering it with a
	// LogoutResponse. Both directions require a valid signature from the
	// provider's configured certificate.
	HandleSAMLSingleLogout(ctx context.Context, r *http.Request, providerIdentifier string) (*SAMLSingleLogoutResult, error)

	// SetAccountLinkService injects the account-link service so the broker
	// provisioning path can create a confirmation request instead of silently
	// merging identities on an email collision.
	SetAccountLinkService(svc authn.AccountLinkRequestService)
}

type federationService struct {
	db                  *gorm.DB
	userRepo            UserRepository
	userIdentityRepo    UserIdentityRepository
	idpRepo             IdentityProviderRepository
	emailDomainRepo     IdentityProviderEmailDomainRepository
	clientRepo          ClientRepository
	userRoleRepo        UserRoleRepository
	roleRepo            RoleRepository
	authEventService    authevent.AuthEventService
	sessionService      authn.SessionService
	accountLinkSvc      authn.AccountLinkRequestService
	eventService        event.EventService
	securitySettingRepo secpolicy.SecuritySettingRepository
	samlStore           cache.WebAuthnSessionStore
	// linkStore holds in-flight account-link requests. Nil disables linking.
	linkStore IdentityLinkRequestStore

	providerCache sync.Map
}

// SetIdentityLinkStore wires the store that holds in-flight account-link
// requests. A setter rather than another constructor parameter: the signature
// is already long, and internal/app must build the adapter after both the idp
// and oauth repositories exist.
func (s *federationService) SetIdentityLinkStore(store IdentityLinkRequestStore) {
	s.linkStore = store
}

func NewFederationService(
	db *gorm.DB,
	userRepo UserRepository,
	userIdentityRepo UserIdentityRepository,
	idpRepo IdentityProviderRepository,
	emailDomainRepo IdentityProviderEmailDomainRepository,
	clientRepo ClientRepository,
	userRoleRepo UserRoleRepository,
	roleRepo RoleRepository,
	authEventService authevent.AuthEventService,
	eventService event.EventService,
	securitySettingRepo secpolicy.SecuritySettingRepository,
	samlStore cache.WebAuthnSessionStore,
	sessionService ...authn.SessionService,
) FederationService {
	var sessions authn.SessionService
	if len(sessionService) > 0 {
		sessions = sessionService[0]
	}
	return &federationService{
		db:                  db,
		userRepo:            userRepo,
		userIdentityRepo:    userIdentityRepo,
		idpRepo:             idpRepo,
		emailDomainRepo:     emailDomainRepo,
		clientRepo:          clientRepo,
		userRoleRepo:        userRoleRepo,
		roleRepo:            roleRepo,
		authEventService:    authEventService,
		sessionService:      sessions,
		eventService:        eventService,
		securitySettingRepo: securitySettingRepo,
		samlStore:           samlStore,
	}
}

// SetAccountLinkService implements FederationService.
func (s *federationService) SetAccountLinkService(svc authn.AccountLinkRequestService) {
	s.accountLinkSvc = svc
}

// buildOIDCConfig unmarshals the non-secret JSONB config (endpoints / scopes /
// attribute_mapping). Issuer, provider client_id and the client secret are NOT
// part of this struct — read them off the model columns (idp.IssuerOrEmpty(),
// idp.ProviderClientIDOrEmpty(), idp.DecryptedProviderClientSecret()).
func buildOIDCConfig(idp *IdentityProvider) (OIDCProviderConfig, error) {
	var cfg OIDCProviderConfig
	if len(idp.Config) > 0 {
		if err := json.Unmarshal(idp.Config, &cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
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

	cfg, err := buildOIDCConfig(idp)
	if err != nil || idp.IssuerOrEmpty() == "" {
		return nil, apperror.NewValidation("identity provider is not configured for OIDC")
	}

	// 2. Validate the external ID token using OIDC discovery.
	// Uses the same idpValidateOIDCToken shared with resolveFederatedPrincipal.
	claims, err := idpValidateOIDCToken(s, ctx, idp.IssuerOrEmpty(), idp.ProviderClientIDOrEmpty(), req.ExternalToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "oidc validation failed")
		return nil, apperror.NewUnauthorized("external token validation failed")
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

	// 4. Resolve the client lazily. Unknown identities should still honor the
	// JIT-disabled branch without requiring a client lookup, but JIT writes must
	// never happen before the client/provider connection is proven.
	var client *Client
	resolveClient := func() (*Client, error) {
		if client != nil {
			return client, nil
		}
		if s.clientRepo == nil {
			return nil, apperror.NewNotFound("client not found for this provider")
		}

		found, lookupErr := s.clientRepo.FindByClientIDAndIdentityProvider(req.ClientID, req.ProviderIdentifier)
		if lookupErr != nil {
			return nil, apperror.NewInternal("client lookup failed", lookupErr)
		}
		// There is deliberately no fallback here. This used to retry the lookup
		// against the tenant's DEFAULT identity provider and accept whatever it
		// found, which meant a client not connected to the requesting provider
		// still authenticated as long as it was connected to the default one —
		// and, once the lookup started rejecting disabled connections, it became
		// the way to route around a connection an admin had just disabled.
		// A client is reachable from a provider only via an enabled connection.
		if found == nil {
			return nil, apperror.NewNotFound("client not found for this provider")
		}
		client = found
		return client, nil
	}

	// 5. Find or provision the user — same logic as resolveFederatedPrincipal
	// (shared validation/JIT path: idpValidateOIDCToken → extractMetadata →
	// find-or-provision via provisionUser/refreshMetadata).
	var user *User
	var internalSub string
	var isNew bool

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		existing, err := txUserIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, externalSub)
		if err != nil {
			return apperror.NewInternal("identity lookup failed", err)
		}

		if existing != nil {
			user, err = txUserRepo.FindByID(existing.UserID)
			if err != nil || user == nil {
				return apperror.NewInternal("user not found for existing identity", err)
			}
			_ = s.refreshMetadata(tx, existing, meta)
		} else {
			if !idp.AllowJITProvisioning {
				return apperror.NewUnauthorized("user not found and JIT provisioning is disabled for this provider")
			}
			resolvedClient, clientErr := resolveClient()
			if clientErr != nil {
				return clientErr
			}
			var provisionErr error
			user, isNew, provisionErr = s.provisionUser(ctx, tx, idp, externalSub, email, meta, &resolvedClient.ClientID)
			if provisionErr != nil {
				return provisionErr
			}
		}

		systemIDPID, sysErr := s.systemIdentityProviderID(idp.TenantID)
		if sysErr != nil {
			return sysErr
		}
		defaultIdentity, err := txUserIdentityRepo.FindByUserIDAndIdentityProviderID(user.UserID, systemIDPID)
		if err != nil {
			return apperror.NewInternal("default identity lookup failed", err)
		}
		if defaultIdentity == nil {
			return apperror.NewInternal("user has no default identity", nil)
		}
		internalSub = defaultIdentity.Sub
		return nil
	})
	if errors.Is(err, errIdentityCreatedConcurrently) {
		user, internalSub, err = s.resolveExistingUserIdentity(idp, externalSub, true)
	}
	if collisionErr := s.handleEmailCollision(ctx, err); collisionErr != nil {
		return nil, collisionErr
	}
	if err != nil {
		return nil, err
	}
	if client == nil {
		client, err = resolveClient()
		if err != nil {
			return nil, err
		}
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

// resolveTokenEndpoint returns the provider's token endpoint: explicit
// token_endpoint → OIDC discovery → issuer-based legacy default. Providers
// like Google/Facebook/Cognito whose token endpoint is not at the issuer root
// are now resolved automatically via discovery.
func resolveTokenEndpoint(ctx context.Context, issuer string, cfg OIDCProviderConfig) string {
	if e := strings.TrimSpace(cfg.TokenEndpoint); e != "" {
		return e
	}
	if iss := strings.TrimSpace(issuer); iss != "" {
		if _, token, err := idpOIDCDiscover(ctx, iss); err == nil && strings.TrimSpace(token) != "" {
			return token
		}
	}
	return strings.TrimRight(issuer, "/") + "/oauth/token"
}

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

	cfg, err := buildOIDCConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation("identity provider configuration is invalid")
	}
	// Fail closed: a key/corruption failure must abort the exchange, never POST the
	// raw ciphertext blob upstream as if it were the client secret.
	clientSecret, err := idp.DecryptedProviderClientSecretStrict()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "provider client secret unavailable")
		return nil, apperror.NewInternal("provider client secret unavailable", err)
	}
	if idp.ProviderClientIDOrEmpty() == "" || clientSecret == "" {
		return nil, apperror.NewValidation("identity provider missing OAuth2 client credentials")
	}

	userinfoURL := cfg.UserinfoEndpoint
	if userinfoURL == "" && idp.IssuerOrEmpty() != "" {
		userinfoURL = strings.TrimRight(idp.IssuerOrEmpty(), "/") + DefaultOIDCUserinfoEndpoint
	}
	if userinfoURL == "" {
		return nil, apperror.NewValidation("identity provider missing userinfo endpoint")
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     idp.ProviderClientIDOrEmpty(),
		ClientSecret: clientSecret,
		RedirectURL:  req.RedirectURI,
		Endpoint: oauth2.Endpoint{
			TokenURL:  resolveTokenEndpoint(ctx, idp.IssuerOrEmpty(), cfg),
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
		Scopes: cfg.Scopes,
	}

	tok, err := idpOAuth2Exchange(ctx, oauth2Cfg, req.Code)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "oauth2 code exchange failed")
		return nil, apperror.NewUnauthorized("failed to exchange authorization code")
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

	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(req.ClientID, idp.Identifier)
	if err != nil || client == nil {
		return nil, apperror.NewNotFound("client not found")
	}

	var user *User
	var internalSub string

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		existing, txErr := txUserIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, externalSub)
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
			user, isNew, txErr = s.provisionUser(ctx, tx, idp, externalSub, email, meta, &client.ClientID)
			if txErr != nil {
				return txErr
			}
			_ = isNew
		}

		identity, txErr := txUserIdentityRepo.FindByUserIDAndIdentityProviderID(user.UserID, idp.IdentityProviderID)
		if txErr != nil || identity == nil {
			return apperror.NewInternal("identity resolution failed", txErr)
		}
		internalSub = identity.Sub
		return nil
	})
	if errors.Is(err, errIdentityCreatedConcurrently) {
		user, internalSub, err = s.resolveExistingUserIdentity(idp, externalSub, false)
	}
	if collisionErr := s.handleEmailCollision(ctx, err); collisionErr != nil {
		return nil, collisionErr
	}
	if err != nil {
		return nil, err
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

	cfg, err := buildOIDCConfig(idp)
	if err != nil || idp.IssuerOrEmpty() == "" {
		return nil, apperror.NewValidation("identity provider is not configured for OIDC")
	}

	claims, err := idpValidateOIDCToken(s, ctx, idp.IssuerOrEmpty(), idp.ProviderClientIDOrEmpty(), req.ExternalToken)
	if err != nil {
		return nil, apperror.NewUnauthorized("external token validation failed: " + err.Error())
	}

	externalSub := stringClaim(claims, "sub")
	if externalSub == "" {
		return nil, apperror.NewValidation("external token missing 'sub' claim")
	}

	// Ensure this external identity isn't already claimed by another user.
	existing, err := s.userIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, externalSub)
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
		IdentityProviderID: idpID,
		Provider:           idp.Provider,
		Sub:                externalSub,
		Metadata:           datatypes.JSON(metaJSON),
	}

	var created *UserIdentity
	err = s.db.Transaction(func(tx *gorm.DB) error {
		c, e := s.userIdentityRepo.WithTx(tx).Create(identity)
		if e != nil {
			return e
		}
		created = c
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeIdentityLinked, 1, idp.TenantID,
			).SetActor(&userID).SetSubject(&created.UserIdentityUUID, "identity")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})
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
	isBuiltin, err := s.isSystemBuiltinIdentity(target)
	if err != nil {
		return err
	}
	if isBuiltin {
		return apperror.NewValidation("the built-in identity cannot be unlinked")
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if e := s.userIdentityRepo.WithTx(tx).DeleteByID(target.UserIdentityID); e != nil {
			return e
		}
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeIdentityUnlinked, 1, target.TenantID,
			).SetActor(&userID).SetSubject(&target.UserIdentityUUID, "identity")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	}); err != nil {
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

func (s *federationService) AdminUnlinkIdentity(ctx context.Context, tenantID int64, actorUserID int64, userUUID uuid.UUID, identityUUID string) error {
	_, span := otel.Tracer("service").Start(ctx, "federation.admin_unlink_identity")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("tenant.id", tenantID),
		attribute.Int64("actor.user.id", actorUserID),
		attribute.String("user.uuid", userUUID.String()),
		attribute.String("identity.uuid", identityUUID),
	)

	// Resolve the target user, strictly scoped to the admin's tenant. A missing
	// or cross-tenant target is reported as NotFound so we never leak whether a
	// user exists in another tenant.
	target, err := s.userRepo.FindByUUID(userUUID)
	if err != nil {
		return apperror.NewInternal("user lookup failed", err)
	}
	if target == nil || target.TenantID != tenantID {
		return apperror.NewNotFound("user not found")
	}

	identities, err := s.userIdentityRepo.FindByUserID(target.UserID)
	if err != nil {
		return apperror.NewInternal("identity lookup failed", err)
	}

	var targetIdentity *UserIdentity
	for i := range identities {
		if identities[i].UserIdentityUUID.String() == identityUUID {
			targetIdentity = &identities[i]
			break
		}
	}
	if targetIdentity == nil {
		return apperror.NewNotFound("identity not found")
	}
	// Defense-in-depth: the identity itself must belong to the admin's tenant.
	if targetIdentity.TenantID != tenantID {
		return apperror.NewNotFound("identity not found")
	}
	isBuiltin, err := s.isSystemBuiltinIdentity(targetIdentity)
	if err != nil {
		return err
	}
	if isBuiltin {
		return apperror.NewValidation("the built-in identity cannot be unlinked")
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if e := s.userIdentityRepo.WithTx(tx).DeleteByID(targetIdentity.UserIdentityID); e != nil {
			return e
		}
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeIdentityUnlinked, 1, targetIdentity.TenantID,
			).SetActor(&actorUserID).SetSubject(&targetIdentity.UserIdentityUUID, "identity")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	}); err != nil {
		return apperror.NewInternal("failed to unlink identity", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &actorUserID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr(fmt.Sprintf("admin unlinked external identity %s from user %s", targetIdentity.Provider, target.UserUUID)),
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

	// Single indexed lookup on the child table (uq_idp_email_domain) replaces the
	// former full-scan-and-JSON-parse over every provider in the tenant.
	match, err := s.emailDomainRepo.FindByTenantAndDomain(tenantID, domain)
	if err != nil {
		return nil, apperror.NewInternal("provider lookup failed", err)
	}
	if match != nil {
		idp, lookupErr := s.idpRepo.FindByID(match.IdentityProviderID)
		if lookupErr == nil && idp != nil {
			span.SetStatus(codes.Ok, "")
			return hrdResponseFrom(idp), nil
		}
	}

	// No domain mapped to an external provider — return the default (maintainerd) IDP.
	defaultIDP, _ := s.idpRepo.FindDefaultByTenantID(tenantID)
	if defaultIDP == nil {
		return nil, apperror.NewNotFound("no identity provider found for this tenant")
	}
	span.SetStatus(codes.Ok, "")
	return hrdResponseFrom(defaultIDP), nil
}

// HomeRealmDiscoveryByClient resolves the tenant from the public client_id and
// delegates to HomeRealmDiscovery. The public surface only ever accepts client_id.
func (s *federationService) HomeRealmDiscoveryByClient(ctx context.Context, clientID string, email string) (*HRDResponseDTO, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, apperror.NewValidation("client_id is required")
	}

	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientID, "")
	if err != nil {
		return nil, apperror.NewInternal("client lookup failed", err)
	}
	if client == nil {
		return nil, apperror.NewNotFound("unknown client")
	}

	return s.HomeRealmDiscovery(ctx, client.TenantID, email)
}

// resolveAuthorizeEndpoint resolves a provider's authorization endpoint: the
// explicit authorization_endpoint when configured, otherwise via OIDC discovery
// from the issuer.
func (s *federationService) resolveAuthorizeEndpoint(ctx context.Context, issuer string, cfg OIDCProviderConfig) (string, error) {
	if e := strings.TrimSpace(cfg.AuthorizationEndpoint); e != "" {
		return e, nil
	}
	if strings.TrimSpace(issuer) == "" {
		return "", apperror.NewValidation("identity provider has no authorization_endpoint or issuer")
	}
	authorize, _, err := idpOIDCDiscover(ctx, issuer)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(authorize) == "" {
		return "", apperror.NewValidation("OIDC discovery returned no authorization endpoint")
	}
	return authorize, nil
}

// ResolveBrokerProvider implements FederationService.
func (s *federationService) ResolveBrokerProvider(ctx context.Context, idpIdentifier string) (*BrokerProviderInfo, error) {
	idp, err := s.idpRepo.FindByIdentifier(idpIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	cfg, err := buildOIDCConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation("identity provider configuration is invalid")
	}
	if strings.TrimSpace(idp.ProviderClientIDOrEmpty()) == "" {
		return nil, apperror.NewValidation("identity provider missing OAuth2 client_id")
	}

	authorize, err := s.resolveAuthorizeEndpoint(ctx, idp.IssuerOrEmpty(), cfg)
	if err != nil {
		return nil, err
	}

	return &BrokerProviderInfo{
		AuthorizationEndpoint: authorize,
		ClientID:              idp.ProviderClientIDOrEmpty(),
		Scopes:                cfg.Scopes,
	}, nil
}

// exchangeUpstreamCode performs the provider leg of an OAuth2/OIDC round trip:
// redeem the authorization code with PKCE, then validate the returned id_token
// against the provider's issuer and our registered client_id, and confirm the
// nonce matches the one this flow generated.
//
// Shared by brokered sign-in and account linking. Both need exactly these
// checks, and a second copy would be free to drift — a missing nonce comparison
// in either path reopens id_token replay.
func (s *federationService) exchangeUpstreamCode(
	ctx context.Context,
	idp *IdentityProvider,
	cfg OIDCProviderConfig,
	clientSecret, code, pkceVerifier, nonce, redirectURI string,
) (string, map[string]any, error) {
	oauth2Cfg := &oauth2.Config{
		ClientID:     idp.ProviderClientIDOrEmpty(),
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint: oauth2.Endpoint{
			TokenURL:  resolveTokenEndpoint(ctx, idp.IssuerOrEmpty(), cfg),
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
	}

	tok, err := idpOAuth2ExchangeWithPKCE(ctx, oauth2Cfg, code, pkceVerifier)
	if err != nil {
		return "", nil, apperror.NewUnauthorized("failed to exchange authorization code")
	}

	rawIDTok, ok := tok.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDTok) == "" {
		return "", nil, apperror.NewUnauthorized("provider did not return an id_token")
	}
	claims, err := s.validateOIDCToken(ctx, idp.IssuerOrEmpty(), idp.ProviderClientIDOrEmpty(), rawIDTok)
	if err != nil {
		return "", nil, apperror.NewUnauthorized("failed to validate provider token")
	}
	// Every flow that reaches here generated and stored a nonce, so an empty one
	// means a corrupted or forged session — fail closed rather than skipping the
	// id_token replay check.
	if nonce == "" {
		return "", nil, apperror.NewUnauthorized("missing nonce for provider token validation")
	}
	if tokNonce, _ := claims["nonce"].(string); tokNonce != nonce {
		return "", nil, apperror.NewUnauthorized("provider token nonce mismatch")
	}
	return rawIDTok, claims, nil
}

// ResolveBrokerUser implements FederationService.
func (s *federationService) ResolveBrokerUser(ctx context.Context, idpID int64, code, pkceVerifier, nonce, redirectURI string, clientID int64) (*BrokerResolvedUser, error) {
	idp, err := s.idpRepo.FindByID(idpID)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	cfg, err := buildOIDCConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation("identity provider configuration is invalid")
	}
	// Fail closed: a key/corruption failure must abort the exchange, never POST the
	// raw ciphertext blob upstream as if it were the client secret.
	clientSecret, err := idp.DecryptedProviderClientSecretStrict()
	if err != nil {
		return nil, apperror.NewInternal("provider client secret unavailable", err)
	}
	if idp.ProviderClientIDOrEmpty() == "" || clientSecret == "" {
		return nil, apperror.NewValidation("identity provider missing OAuth2 client credentials")
	}

	rawIDTok, claims, err := s.exchangeUpstreamCode(ctx, idp, cfg, clientSecret, code, pkceVerifier, nonce, redirectURI)
	if err != nil {
		return nil, err
	}
	_ = rawIDTok

	externalSub, ok := claims["sub"].(string)
	if !ok || externalSub == "" {
		return nil, apperror.NewUnauthorized("provider returned no subject claim")
	}

	email, _ := claims["email"].(string)
	meta := extractMetadata(claims, cfg.AttributeMapping)

	var user *User
	var identitySub string
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)

		existing, txErr := txUserIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, externalSub)
		if txErr != nil {
			return apperror.NewInternal("identity lookup failed", txErr)
		}
		if existing != nil {
			var lookupErr error
			user, lookupErr = txUserRepo.FindByID(existing.UserID)
			if lookupErr != nil || user == nil {
				return apperror.NewInternal("user not found for existing identity", lookupErr)
			}
		} else {
			u, _, txErr := s.provisionUser(ctx, tx, idp, externalSub, email, meta, &clientID)
			if txErr != nil {
				return txErr
			}
			user = u
		}

		identity, txErr := txUserIdentityRepo.FindByUserIDAndIdentityProviderID(user.UserID, idp.IdentityProviderID)
		if txErr != nil || identity == nil {
			return apperror.NewInternal("failed to resolve internal identity", txErr)
		}
		identitySub = identity.Sub
		return nil
	})
	if errors.Is(err, errIdentityCreatedConcurrently) {
		user, identitySub, err = s.resolveExistingUserIdentity(idp, externalSub, false)
	}
	var collision *errEmailCollision
	if errors.As(err, &collision) {
		// Fail closed: without an account-link service we cannot create a
		// confirmation request, and we must never silently merge. Surface the
		// collision as an error instead of dereferencing a nil service.
		if s.accountLinkSvc == nil {
			return nil, apperror.NewConflict("this email is already associated with an existing account and account linking is not available")
		}
		req, initErr := s.accountLinkSvc.Initiate(ctx, authn.InitiateAccountLinkInput{
			TenantID:           collision.tenantID,
			ExistingUserID:     collision.existingUserID,
			IdentityProviderID: collision.identityProviderID,
			ProviderName:       collision.providerName,
			ProviderSubject:    collision.providerSub,
			ProviderEmail:      collision.providerEmail,
			ProviderClaims:     collision.providerClaims,
			IPAddress:          middleware.ClientIPFromContext(ctx),
		})
		if initErr != nil {
			return nil, initErr
		}
		return &BrokerResolvedUser{
			AccountLinkToken:    req.ConfirmationToken,
			AccountLinkProvider: collision.providerName,
			AccountLinkEmail:    collision.providerEmail,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	// Record the upstream session: idp_session_id is how an upstream provider's
	// back-channel logout locates the session it just ended. The column and its
	// index already existed and were never written.
	brokerAttrs := authn.SessionAttributes{
		AMR:                []string{"federated"},
		ACR:                "1",
		IdentityProviderID: &idp.IdentityProviderID,
	}
	if upstreamSID, _ := claims["sid"].(string); strings.TrimSpace(upstreamSID) != "" {
		brokerAttrs.IDPSessionID = &upstreamSID
	}
	sessionID, err := s.createBrokerSession(ctx, user, clientID, brokerAttrs)
	if err != nil {
		return nil, err
	}

	return &BrokerResolvedUser{
		UserID:      user.UserID,
		UserUUID:    user.UserUUID,
		IdentitySub: identitySub,
		SessionID:   sessionID,
	}, nil
}

// validateOIDCToken fetches the provider's OIDC discovery doc, verifies the
// token's signature + standard claims, and returns the raw claims map.
func (s *federationService) validateOIDCToken(ctx context.Context, issuer, clientID, rawToken string) (map[string]interface{}, error) {
	octx := oidclib.ClientContext(ctx, idpHTTPClientFactory())
	provider, err := s.getOrDiscoverProvider(octx, issuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed for %s: %w", issuer, err)
	}

	if clientID == "" {
		return nil, fmt.Errorf("OIDC client_id is required")
	}
	verifierCfg := &oidclib.Config{ClientID: clientID, Now: oidcSkewedNow()}
	verifier := provider.Verifier(verifierCfg)

	idToken, verr := verifier.Verify(ctx, rawToken)
	if verr != nil {
		s.providerCache.Delete(issuer)
		provider, err = s.getOrDiscoverProvider(octx, issuer)
		if err != nil {
			return nil, fmt.Errorf("OIDC re-discovery failed for %s: %w", issuer, err)
		}
		verifier = provider.Verifier(verifierCfg)
		idToken, verr = verifier.Verify(ctx, rawToken)
	}
	if verr != nil {
		return nil, fmt.Errorf("token verification failed: %w", verr)
	}

	var claims map[string]interface{}
	_ = idToken.Claims(&claims)
	return claims, nil
}

func (s *federationService) getOrDiscoverProvider(ctx context.Context, issuer string) (*oidclib.Provider, error) {
	if cached, ok := s.providerCache.Load(issuer); ok {
		return cached.(*oidclib.Provider), nil
	}
	provider, err := oidclib.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	s.providerCache.Store(issuer, provider)
	return provider, nil
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
	clientID *int64,
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
			// Fail closed: a verified-email match with a pre-existing account must
			// NEVER be silently merged. Always surface the collision so the caller
			// resolves it explicitly — the broker flow turns it into a confirmation
			// request, the direct-token flows reject it. Whether the account-link
			// service happens to be wired only changes how the caller reacts; it can
			// never downgrade this into an unconfirmed silent link.
			return nil, false, &errEmailCollision{
				tenantID:           idp.TenantID,
				existingUserID:     existing.UserID,
				identityProviderID: idp.IdentityProviderID,
				providerName:       idp.Provider,
				providerSub:        externalSub,
				providerEmail:      email,
				providerClaims:     func() []byte { b, _ := json.Marshal(meta); return b }(),
			}
		}
	}

	if user == nil {
		// Social/federated JIT provisioning creates a brand-new account, so the
		// tenant's registration policy applies here exactly as it does to
		// self-signup — otherwise these controls are trivially bypassed by
		// logging in through any enabled social IdP. Existing-user federated
		// LOGIN (handled above) is unaffected; only the CREATE is gated.
		regPolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, idp.TenantID)
		if regPolicy != nil {
			// blocked_email_domains is a hard org policy ("this org will never
			// hold an identity at these domains") and must hold regardless of the
			// provisioning mechanism.
			if regPolicy.EmailDomainBlocked(email) {
				return nil, false, apperror.NewValidation("email domain is not permitted")
			}
			// A closed directory (self_registration_enabled=false) must not be
			// openable via social login. The allowlist is intentionally NOT
			// applied — it is a self-signup gate, whereas federated identity is a
			// separate, admin-configured trust path.
			if !regPolicy.SelfRegistrationEnabled {
				return nil, false, apperror.NewUnauthorized("registration is disabled for this tenant")
			}
		}

		// Create a new user from the external profile. Unlike user.User, this
		// package's User struct has no BeforeCreate hook and no column defaults,
		// so the NOT NULL columns (user_uuid, status, metadata) must be set here
		// explicitly — otherwise the insert writes a zero UUID / empty status /
		// NULL metadata and fails the metadata NOT NULL constraint. A user vouched
		// for by a configured, admin-enabled IdP is provisioned active.
		username := deriveUsername(meta, email)
		newUser := &User{
			UserUUID:        uuid.New(),
			TenantID:        idp.TenantID,
			Email:           email,
			Username:        username,
			IsEmailVerified: meta.EmailVerified,
			Status:          shared.StatusActive,
			Metadata:        datatypes.JSON([]byte("{}")),
		}
		created, err := txUserRepo.Create(newUser)
		if err != nil || created == nil {
			return nil, false, apperror.NewInternal("failed to provision user", err)
		}
		user = created
		isNew = true

		// Assign default role if available.
		if defaultRole, _ := s.findDefaultRole(s.roleRepo.WithTx(tx), idp.TenantID); defaultRole != nil {
			if err := tx.Create(&UserRole{UserRoleUUID: uuid.New(), UserID: user.UserID, RoleID: defaultRole.RoleID}).Error; err != nil {
				return nil, false, apperror.NewInternal("failed to assign default role", err)
			}
		}
	}

	// Create the built-in (system maintainerd) identity if it doesn't exist yet.
	// This ensures our RBAC system always has a stable sub for the user. The
	// built-in identity is keyed to the tenant's system IdP — never the provider
	// string — so it stays distinct from any external "maintainerd" identity
	// federated in as an enterprise IdP.
	systemIDP, sysErr := s.idpRepo.FindSystemByTenantID(idp.TenantID)
	if sysErr != nil {
		return nil, false, apperror.NewInternal("system identity provider lookup failed", sysErr)
	}
	if systemIDP == nil {
		return nil, false, apperror.NewInternal("tenant has no system identity provider", nil)
	}
	defaultIdentity, err := txUserIdentityRepo.FindByUserIDAndIdentityProviderID(user.UserID, systemIDP.IdentityProviderID)
	if err != nil {
		return nil, false, apperror.NewInternal("default identity lookup failed", err)
	}
	if defaultIdentity == nil {
		systemIDPID := systemIDP.IdentityProviderID
		defIdentity := &UserIdentity{
			UserIdentityUUID:   uuid.New(),
			TenantID:           idp.TenantID,
			UserID:             user.UserID,
			IdentityProviderID: systemIDPID,
			Provider:           shared.ProviderMaintainerd,
			Sub:                uuid.New().String(),
			Metadata:           datatypes.JSON([]byte(`{}`)),
		}
		if _, err := txUserIdentityRepo.Create(defIdentity); err != nil {
			return nil, false, apperror.NewInternal("failed to create default identity", err)
		}
	}

	// Create the external identity.
	idpID := idp.IdentityProviderID
	metaJSON, _ := json.Marshal(meta)
	jitNow := time.Now()
	provisioningSource := "jit"
	extIdentity := &UserIdentity{
		UserIdentityUUID:   uuid.New(),
		TenantID:           idp.TenantID,
		UserID:             user.UserID,
		IdentityProviderID: idpID,
		Provider:           idp.Provider,
		Sub:                externalSub,
		Metadata:           datatypes.JSON(metaJSON),
		JITProvisionedAt:   &jitNow,
		ProvisioningSource: &provisioningSource,
	}
	existing, created, err := txUserIdentityRepo.CreateByTenantProviderSubIfAbsent(extIdentity)
	if err != nil {
		return nil, false, apperror.NewInternal("failed to create external identity", err)
	}
	if !created {
		// The insert was refused, so something already owns this subject. Never
		// fall through here: the user row was created moments ago and has no
		// external identity, so continuing would mint tokens for a half-built
		// account and do it again on every retry.
		switch {
		case existing == nil:
			return nil, false, apperror.NewInternal("external identity could not be resolved after a conflict", nil)
		case existing.IdentityProviderID != idpID:
			// A DIFFERENT provider in this tenant already issued this subject.
			// `sub` is unique per issuer (OIDC Core §2) and the tenant is the
			// issuer, so this is unresolvable without operator input — refuse
			// rather than guess which human the subject refers to.
			return nil, false, apperror.NewConflict(
				"this provider returned a subject that is already in use by another identity provider in this tenant")
		case existing.UserID != user.UserID:
			// Same provider, same subject, different user: a concurrent request
			// won the race. The caller re-resolves against the winning row.
			return nil, false, errIdentityCreatedConcurrently
		}
	}

	return user, isNew, nil
}

// resolveExistingUserIdentity is the fallback taken when the external identity
// was created concurrently. It resolves the user by the external identity's
// (tenant, provider, sub) triple — which is unambiguous because sub is unique
// even when two identities share the "maintainerd" provider slug. When
// useSystemIdentity is true the returned sub is that of the user's built-in
// system identity (resolved via the tenant's system IdP id); otherwise it is the
// external identity's own sub.
func (s *federationService) resolveExistingUserIdentity(idp *IdentityProvider, externalSub string, useSystemIdentity bool) (*User, string, error) {
	existing, err := s.userIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, externalSub)
	if err != nil {
		return nil, "", apperror.NewInternal("identity lookup failed", err)
	}
	if existing == nil {
		return nil, "", apperror.NewUnauthorized("unable to resolve provider identity")
	}
	user, err := s.userRepo.FindByID(existing.UserID)
	if err != nil || user == nil {
		return nil, "", apperror.NewInternal("user lookup failed", err)
	}
	if !useSystemIdentity {
		return user, existing.Sub, nil
	}
	systemIDPID, err := s.systemIdentityProviderID(idp.TenantID)
	if err != nil {
		return nil, "", err
	}
	identity, err := s.userIdentityRepo.FindByUserIDAndIdentityProviderID(user.UserID, systemIDPID)
	if err != nil || identity == nil {
		return nil, "", apperror.NewInternal("identity resolution failed", err)
	}
	return user, identity.Sub, nil
}

// systemIdentityProviderID resolves the identity_provider_id of the tenant's
// built-in system IdP. Built-in identity lookups key off this id rather than the
// "maintainerd" provider string so they never collide with an external
// (enterprise) maintainerd IdP federated into the same tenant.
func (s *federationService) systemIdentityProviderID(tenantID int64) (int64, error) {
	systemIDP, err := s.idpRepo.FindSystemByTenantID(tenantID)
	if err != nil {
		return 0, apperror.NewInternal("system identity provider lookup failed", err)
	}
	if systemIDP == nil {
		return 0, apperror.NewInternal("tenant has no system identity provider", nil)
	}
	return systemIDP.IdentityProviderID, nil
}

// isSystemBuiltinIdentity reports whether an identity is the tenant's built-in
// system identity, which may never be unlinked. It decides via the identity's
// IdentityProviderID → the tenant's system IdP id, NOT the provider string, so
// an external federated "maintainerd" identity is correctly treated as
// unlinkable. IdentityProviderID is NOT NULL (migration 030), so the provider
// is always known and the comparison is exact — no provider-name guessing.
func (s *federationService) isSystemBuiltinIdentity(identity *UserIdentity) (bool, error) {
	if identity == nil {
		return false, nil
	}
	if s.idpRepo == nil {
		// Fail closed: with no way to resolve the tenant's system provider, treat
		// a built-in-looking identity as built-in rather than allowing the user to
		// unlink their only means of signing in.
		return identity.Provider == shared.ProviderMaintainerd, nil
	}
	systemIDP, err := s.idpRepo.FindSystemByTenantID(identity.TenantID)
	if err != nil {
		return false, apperror.NewInternal("system identity provider lookup failed", err)
	}
	if systemIDP == nil {
		return identity.Provider == shared.ProviderMaintainerd, nil
	}
	return identity.IdentityProviderID == systemIDP.IdentityProviderID, nil
}

func (s *federationService) createBrokerSession(ctx context.Context, user *User, clientID int64, attrs authn.SessionAttributes) (string, error) {
	if s.sessionService == nil {
		return "", nil
	}
	var client *Client
	if clientID > 0 && s.clientRepo != nil {
		client, _ = s.clientRepo.FindByID(clientID)
	}
	var clientTenantID int64
	if client != nil {
		clientTenantID = client.TenantID
	}
	policy := s.resolveFederationSessionPolicy(client)
	if svc, ok := s.sessionService.(policyAwareSessionService); ok {
		if err := svc.EnforceConcurrentLimitWithPolicy(ctx, user.UserUUID, user.UserID, policy); err != nil {
			return "", apperror.NewInternal("session limit enforcement failed", err)
		}
		if client != nil && client.ClientID > 0 {
			cid := client.ClientID
			attrs.ClientID = &cid
		}
		sess, err := svc.CreateSessionWithPolicy(ctx, user.UserID, clientTenantID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), policy, attrs)
		if err != nil {
			return "", apperror.NewInternal("session creation failed", err)
		}
		return sess.UserSessionUUID.String(), nil
	}
	if err := s.sessionService.EnforceConcurrentLimit(ctx, user.UserUUID, user.UserID); err != nil {
		return "", apperror.NewInternal("session limit enforcement failed", err)
	}
	sess, err := s.sessionService.CreateSession(ctx, user.UserID, clientTenantID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx))
	if err != nil {
		return "", apperror.NewInternal("session creation failed", err)
	}
	return sess.UserSessionUUID.String(), nil
}

func (s *federationService) refreshMetadata(tx *gorm.DB, identity *UserIdentity, meta IdentityMetadata) error {
	metaJSON, _ := json.Marshal(meta)
	return tx.Model(&UserIdentity{}).
		Where("user_identity_id = ?", identity.UserIdentityID).
		Update("metadata", datatypes.JSON(metaJSON)).Error
}

func (s *federationService) generateTokens(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	var sessionID string
	var genClientTenantID int64
	if client != nil {
		genClientTenantID = client.TenantID
	}
	policy := s.resolveFederationSessionPolicy(client)
	if s.sessionService != nil {
		if svc, ok := s.sessionService.(policyAwareSessionService); ok {
			if err := svc.EnforceConcurrentLimitWithPolicy(ctx, user.UserUUID, user.UserID, policy); err != nil {
				return nil, apperror.NewInternal("session limit enforcement failed", err)
			}
			genAttrs := authn.SessionAttributes{AMR: []string{"federated"}, ACR: "1"}
			if client != nil && client.ClientID > 0 {
				cid := client.ClientID
				genAttrs.ClientID = &cid
			}
			sess, err := svc.CreateSessionWithPolicy(ctx, user.UserID, genClientTenantID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), policy, genAttrs)
			if err != nil {
				return nil, apperror.NewInternal("session creation failed", err)
			}
			sessionID = sess.UserSessionUUID.String()
		} else {
			if err := s.sessionService.EnforceConcurrentLimit(ctx, user.UserUUID, user.UserID); err != nil {
				return nil, apperror.NewInternal("session limit enforcement failed", err)
			}
			sess, err := s.sessionService.CreateSession(ctx, user.UserID, genClientTenantID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx))
			if err != nil {
				return nil, apperror.NewInternal("session creation failed", err)
			}
			sessionID = sess.UserSessionUUID.String()
		}
	}

	accessOpts := &jwtlib.AccessTokenOptions{
		AMR:       []string{jwtlib.AMRMFA},
		ACR:       jwtlib.ACRLevel1,
		SessionID: sessionID,
	}
	if policy.AccessTokenTTLSeconds > 0 {
		accessOpts.AccessTokenTTL = time.Duration(policy.AccessTokenTTLSeconds) * time.Second
	}

	accessToken, err := idpGenerateAccessTokenWithOptionsContext(
		ctx,
		sub,
		shared.DefaultTokenScope,
		jwtlib.TokenIssuerPtr(client.Domain),
		*client.Identifier,
		*client.Identifier,
		federationTokenRealm(client),
		accessOpts,
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
	idToken, err := idpGenerateIDTokenWithContext(ctx, sub, jwtlib.TokenIssuerPtr(client.Domain), *client.Identifier, federationTokenRealm(client), profile, "", &jwtlib.IDTokenParams{
		RequestedScopes: strings.Fields(shared.DefaultTokenScope),
		AMR:             []string{jwtlib.AMRMFA},
		ACR:             jwtlib.ACRLevel1,
	})
	if err != nil {
		return nil, apperror.NewInternal("id token generation failed", err)
	}

	// Matches the access and ID tokens minted just above: `iss` identifies this
	// authorization server, never the client's domain. Left raw, the refresh token
	// was the only member of the set whose issuer depended on the legacy
	// client-domain entries in the issuer allowlist still being present.
	refreshToken, err := idpGenerateRefreshTokenWithContext(ctx, sub, jwtlib.TokenIssuerPtr(client.Domain), *client.Identifier, federationTokenRealm(client))
	if err != nil {
		return nil, apperror.NewInternal("refresh token generation failed", err)
	}

	var expiresIn int64 = shared.DefaultAccessTokenExpiresIn
	if policy.AccessTokenTTLSeconds > 0 {
		expiresIn = int64(policy.AccessTokenTTLSeconds)
	}
	resp := &LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}
	if sessionID != "" {
		resp.SessionID = &sessionID
	}
	return resp, nil
}

func federationTokenRealm(client *Client) string {
	if client == nil {
		return ""
	}
	if client.Tenant != nil && strings.TrimSpace(client.Tenant.Name) != "" {
		return client.Tenant.Name
	}
	return fmt.Sprintf("tenant:%d", client.TenantID)
}

func (s *federationService) resolveFederationSessionPolicy(client *Client) secpolicy.EffectiveSessionPolicy {
	if s.securitySettingRepo == nil || client == nil || client.TenantID <= 0 {
		policy, _ := secpolicy.ResolveEffectiveSessionPolicy(nil, nil, secpolicy.SecuritySettingClientOverrides{})
		return policy
	}
	ss, err := s.securitySettingRepo.FindByTenantID(client.TenantID)
	if err != nil || ss == nil {
		policy, _ := secpolicy.ResolveEffectiveSessionPolicy(nil, nil, secpolicy.SecuritySettingClientOverrides{})
		return policy
	}
	sessionConfig := mapFromJSON(ss.SessionConfig)
	mfaConfig := mapFromJSON(ss.MFAConfig)
	policy, err := secpolicy.ResolveEffectiveSessionPolicy(sessionConfig, mfaConfig, secpolicy.SecuritySettingClientOverrides{})
	if err != nil {
		policy, _ = secpolicy.ResolveEffectiveSessionPolicy(nil, nil, secpolicy.SecuritySettingClientOverrides{})
	}
	return policy
}

func mapFromJSON(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

type policyAwareSessionService interface {
	EnforceConcurrentLimitWithPolicy(ctx context.Context, userUUID uuid.UUID, userID int64, policy secpolicy.EffectiveSessionPolicy) error
	CreateSessionWithPolicy(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string, policy secpolicy.EffectiveSessionPolicy, attrs authn.SessionAttributes) (*authn.UserSession, error)
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
		IsDefault:    ui.Provider == shared.ProviderMaintainerd,
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

// ── IdP Test Connection ──────────────────────────────────────────────────────

// TestConnection probes an unsaved IdP configuration by performing OIDC
// discovery and a JWKS endpoint validation. It uses the SSRF-safe
// idpHTTPClient (http_client.go) so external probes cannot reach local
// network resources.
func (s *federationService) TestConnection(_ context.Context, req TestConnectionRequestDTO) (*TestConnectionResultDTO, error) {
	client := idpHTTPClient()
	result := &TestConnectionResultDTO{Success: true}

	addCheck := func(step, url string, ok bool, err error) {
		c := TestCheckDTO{Step: step, URL: url, OK: ok}
		if err != nil {
			c.Error = err.Error()
			result.Success = false
		}
		result.Checks = append(result.Checks, c)
	}

	// 1) OIDC Discovery
	wellKnownURL := strings.TrimRight(req.DiscoveryURL, "/") + "/.well-known/openid-configuration"
	resp, err := client.Get(wellKnownURL)
	if err != nil {
		addCheck("OIDC discovery", wellKnownURL, false, fmt.Errorf("GET failed: %w", err))
		return result, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		addCheck("OIDC discovery", wellKnownURL, false,
			fmt.Errorf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode)))
		return result, nil
	}
	addCheck("OIDC discovery", wellKnownURL, true, nil)

	// 2) Parse discovery document
	var metadata struct {
		Issuer                string `json:"issuer"`
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		UserinfoEndpoint      string `json:"userinfo_endpoint"`
		JWKSURI               string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		addCheck("Parse discovery JSON", wellKnownURL, false,
			fmt.Errorf("JSON decode: %w", err))
		return result, nil
	}
	addCheck("Parse discovery JSON", wellKnownURL, true, nil)

	if metadata.JWKSURI == "" {
		addCheck("JWKS endpoint", wellKnownURL, false,
			fmt.Errorf("jwks_uri is empty in discovery document"))
		return result, nil
	}

	// 3) JWKS probe
	jwksResp, err := client.Get(metadata.JWKSURI)
	if err != nil {
		addCheck("JWKS probe", metadata.JWKSURI, false, fmt.Errorf("GET failed: %w", err))
		return result, nil
	}
	defer func() { _ = jwksResp.Body.Close() }()

	if jwksResp.StatusCode >= 400 {
		addCheck("JWKS probe", metadata.JWKSURI, false,
			fmt.Errorf("HTTP %d %s", jwksResp.StatusCode, http.StatusText(jwksResp.StatusCode)))
		return result, nil
	}

	var jwks json.RawMessage
	if err := json.NewDecoder(jwksResp.Body).Decode(&jwks); err != nil {
		addCheck("JWKS probe", metadata.JWKSURI, false,
			fmt.Errorf("JSON decode: %w", err))
		return result, nil
	}
	addCheck("JWKS probe", metadata.JWKSURI, true, nil)

	return result, nil
}
