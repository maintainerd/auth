package oauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	clientpkg "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
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
	// authorizationCodeTTL is the lifetime of an authorization code (RFC 6749 §4.1.2).
	authorizationCodeTTL = 10 * time.Minute
	// consentChallengeTTL is the lifetime of a consent challenge.
	consentChallengeTTL = 10 * time.Minute
	// authorizationCodeLength is the byte length of the raw authorization code.
	authorizationCodeLength = 32
)

// TenantResolver resolves a subdomain tenant slug (the DNS name) to its tenant
// ID. It lets the authorize endpoint make the REQUEST's own host authoritative
// for the client_id ↔ tenant binding without importing internal/tenant: the
// oauth package declares this consumer interface and internal/server wires an
// app-layer adapter over the tenant service via SetTenantResolver. This mirrors
// authn.TenantResolver so /oauth/authorize and /login agree on tenant binding.
type TenantResolver interface {
	ResolveTenantIDByName(ctx context.Context, name string) (tenantID int64, isSystem bool, err error)
	// ResolveSystemTenantID returns the ID of the unique system tenant so the
	// bare system host can be bound as strictly as a subdomain tenant.
	ResolveSystemTenantID(ctx context.Context) (tenantID int64, err error)
}

// authorizeTenantResolver is the injected resolver that maps a request-host slug
// to a tenant ID. It is set once at wiring time via SetTenantResolver. When nil
// (e.g. unit tests that do not exercise the subdomain path) the authorize
// endpoint preserves the legacy session-tenant binding, so the change is
// strictly additive.
var authorizeTenantResolver TenantResolver

// SetTenantResolver injects the subdomain tenant resolver used to make the
// request host authoritative on the OAuth authorize endpoint. Mirrors
// authn.SetTenantResolver; an app-layer adapter over the tenant service is wired
// in internal/server.
func SetTenantResolver(r TenantResolver) { authorizeTenantResolver = r }

// resolveRequestTenantID reports the tenant the browser is actually on, derived
// server-side from the request host (Origin → X-Forwarded-Host → Host, resolved
// by middleware.RequestTenantMiddleware) — NEVER from a caller-supplied
// tenant_id. It returns ok=true, and the request is bound strictly, in two
// cases:
//
//   - a recognized subdomain tenant whose slug resolves to a real tenant ID; or
//   - the bare system host (rt.IsSystem), resolved to the system tenant's ID.
//
// Only a genuinely unrecognized host (not OK), or a missing/failing resolver,
// yields ok=false so the caller falls back to the existing session-tenant
// binding.
func resolveRequestTenantID(ctx context.Context) (int64, bool) {
	rt, ok := middleware.RequestTenantFromContext(ctx)
	if !ok || !rt.OK {
		return 0, false
	}
	if authorizeTenantResolver == nil {
		return 0, false
	}

	// Bare system host: the client_id must belong to the system tenant, so a
	// regular tenant's client cannot be driven on the system surface.
	if rt.IsSystem {
		tenantID, err := authorizeTenantResolver.ResolveSystemTenantID(ctx)
		if err != nil || tenantID == 0 {
			return 0, false
		}
		return tenantID, true
	}

	if rt.Slug == "" {
		return 0, false
	}
	tenantID, _, err := authorizeTenantResolver.ResolveTenantIDByName(ctx, rt.Slug)
	if err != nil || tenantID == 0 {
		return 0, false
	}
	return tenantID, true
}

// OAuthAuthorizeResult is the internal result returned by the authorize service
// method. One of RedirectURI or ConsentChallenge will be set.
type OAuthAuthorizeResult struct {
	// RedirectURI is the full redirect (including ?code=...&state=...) when
	// the authorization code was issued immediately.
	RedirectURI string
	// ConsentChallenge is set when user consent is required. The frontend
	// must redirect the user to the consent page.
	ConsentChallenge string
}

// OAuthConsentDecisionResult is the internal result from processing consent.
type OAuthConsentDecisionResult struct {
	RedirectURI string
}

// OAuthAuthorizeService handles the OAuth 2.0 authorization endpoint logic.
type OAuthAuthorizeService interface {
	// Authorize processes an authorization request. It validates the client,
	// redirect URI, and PKCE parameters. Depending on whether consent is needed,
	// it either issues an authorization code immediately or creates a consent
	// challenge for the frontend to resolve.
	Authorize(ctx context.Context, req OAuthAuthorizeRequestDTO, userID int64, tenantID int64) (*OAuthAuthorizeResult, *apperror.OAuthError)

	// PrepareAuthorize validates an unauthenticated authorization request enough
	// to safely render the hosted login page (the client must exist and be active
	// and the redirect_uri must be registered). Full request validation happens
	// in Authorize once the user has a session.
	PrepareAuthorize(ctx context.Context, req OAuthAuthorizeRequestDTO) *apperror.OAuthError

	// StartBroker begins the upstream (OAuth #2) leg of a brokered login: it
	// validates that idp_hint is an enabled connection for the client, persists a
	// broker session correlating the original request, and returns the redirect
	// URL to the upstream provider's authorize endpoint.
	StartBroker(ctx context.Context, req OAuthAuthorizeRequestDTO) (*OAuthAuthorizeResult, *apperror.OAuthError)

	// HandleCallback completes a brokered login: it looks up the broker session by
	// idp_state, marks it consumed, exchanges the upstream code (validate +
	// provision), and resumes the original OAuth #1 request by issuing a
	// maintainerd authorization code back to the downstream app.
	HandleCallback(ctx context.Context, idpIdentifier, code, state string) (redirectURL string, accessToken string, oerr *apperror.OAuthError)

	// ResolveBrokerErrorRedirect maps a failed brokered login — an `error` from
	// the upstream IdP, a missing authorization code, or a failed exchange — to a
	// URL back into the identity login UI, so the browser is never left stranded
	// on the API callback endpoint. Best-effort; it always returns a URL.
	ResolveBrokerErrorRedirect(ctx context.Context, idpIdentifier, state, errCode, errDesc string) string

	// GetConsentChallenge retrieves a pending consent challenge by its UUID.
	GetConsentChallenge(ctx context.Context, challengeUUID uuid.UUID, userID int64) (*OAuthConsentChallengeResponseDTO, error)

	// HandleConsent processes the user's consent decision. On approval, it
	// persists the consent grant and issues an authorization code. On denial,
	// it returns a redirect with an error.
	HandleConsent(ctx context.Context, decision OAuthConsentDecisionDTO, userID int64) (*OAuthConsentDecisionResult, *apperror.OAuthError)

	// PrepareAuthorizeSignup persists the authorize request for a signup flow
	// (screen_hint=signup, no session) and returns the request_id for the SPA
	// plus an opaque browser-binding secret the caller stores in an httpOnly
	// cookie. Only the secret's hash is persisted, binding the request to the
	// initiating browser (Ory login_challenge / Keycloak auth-session model).
	PrepareAuthorizeSignup(ctx context.Context, req OAuthAuthorizeRequestDTO) (requestID string, bindingSecret string, oerr *apperror.OAuthError)

	// ContinueAuthorize resumes a persisted authorize request after registration,
	// issuing an authorization code bound to the authenticated user. bindingSecret
	// (from the initiating browser's cookie) must match the hash stored at prepare
	// time, so a leaked request_id cannot be continued from another session.
	ContinueAuthorize(ctx context.Context, requestID string, bindingSecret string, userID int64, tenantID int64) (*OAuthAuthorizeResult, *apperror.OAuthError)

	// BrokerResume completes a brokered OAuth flow after a user has confirmed an
	// account link. It validates the link token, finds the pending broker session,
	// issues an authorization code for the linked user, and returns the downstream
	// redirect URL.
	BrokerResume(ctx context.Context, req BrokerResumeRequestDTO, userID int64) (*BrokerResumeResult, *apperror.OAuthError)
}

// BrokerResumeRequestDTO is the request body for POST /oauth/broker/resume.
type BrokerResumeRequestDTO struct {
	BrokerSessionUUID string `json:"broker_session_uuid"`
	AccountLinkToken  string `json:"account_link_token"`
}

// BrokerResumeResult carries the downstream redirect URL and optional SSO
// access token issued after a successful broker resume.
type BrokerResumeResult struct {
	RedirectURL string
	AccessToken string
}

type oauthAuthorizeService struct {
	db                  *gorm.DB
	clientRepo          ClientRepository
	clientURIRepo       ClientURIRepository
	authCodeRepo        OAuthAuthorizationCodeRepository
	consentGrantRepo    OAuthConsentGrantRepository
	consentChallRepo    OAuthConsentChallengeRepository
	authEventService    authevent.AuthEventService
	securitySettingRepo secpolicy.SecuritySettingRepository
	authReqRepo         OAuthAuthorizeRequestRepository
}

// NewOAuthAuthorizeService creates a new OAuthAuthorizeService.
func NewOAuthAuthorizeService(
	db *gorm.DB,
	clientRepo ClientRepository,
	clientURIRepo ClientURIRepository,
	authCodeRepo OAuthAuthorizationCodeRepository,
	consentGrantRepo OAuthConsentGrantRepository,
	consentChallRepo OAuthConsentChallengeRepository,
	authEventService authevent.AuthEventService,
	authReqRepo OAuthAuthorizeRequestRepository,
	securitySettingRepo ...secpolicy.SecuritySettingRepository,
) OAuthAuthorizeService {
	var settings secpolicy.SecuritySettingRepository
	if len(securitySettingRepo) > 0 {
		settings = securitySettingRepo[0]
	}
	return &oauthAuthorizeService{
		db:                  db,
		clientRepo:          clientRepo,
		clientURIRepo:       clientURIRepo,
		authCodeRepo:        authCodeRepo,
		consentGrantRepo:    consentGrantRepo,
		consentChallRepo:    consentChallRepo,
		authEventService:    authEventService,
		securitySettingRepo: settings,
		authReqRepo:         authReqRepo,
	}
}

// Authorize implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) Authorize(ctx context.Context, req OAuthAuthorizeRequestDTO, userID int64, tenantID int64) (*OAuthAuthorizeResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.authorize")
	defer span.End()
	span.SetAttributes(
		attribute.String("oauth.client_id", req.ClientID),
		attribute.String("oauth.response_type", req.ResponseType),
		attribute.Int64("user.id", userID),
	)

	client, err := s.resolveAuthorizeClient(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if client == nil || client.Status != shared.StatusActive {
		span.SetStatus(codes.Error, "client not found or inactive")
		return nil, apperror.NewOAuthInvalidRequest("unknown or inactive client context")
	}

	// Tenant binding (SECURITY-CRITICAL): the client_id must belong to the tenant
	// whose surface the browser is actually on. When the request host resolves to
	// a known subdomain tenant, THAT tenant is authoritative — it is derived
	// server-side from the request (Origin/Host), not from any caller-supplied
	// tenant_id — so an external app cannot drive a client_id from tenant A while
	// the browser is on tenant B's subdomain. This is a hard error: there is NO
	// fallback to the system tenant. When the request host is NOT recognizable
	// (e.g. a trusted internal caller with no tenant surface), we preserve the
	// legacy binding against the authenticated session's tenant. Combined with
	// resolveAuthorizeClient (which rejects seeded system clients as public
	// client_ids, except the allowed hosted-login client), an external client_id
	// caller cannot act as another tenant.
	if requestTenantID, ok := resolveRequestTenantID(ctx); ok {
		if client.TenantID != requestTenantID {
			span.SetStatus(codes.Error, "client request-tenant mismatch")
			return nil, apperror.NewOAuthInvalidRequest("unknown client_id")
		}
	} else if client.TenantID != tenantID {
		span.SetStatus(codes.Error, "client tenant mismatch")
		return nil, apperror.NewOAuthInvalidRequest("unknown client_id")
	}

	if oerr := validateOAuthPKCE(req.CodeChallenge, req.CodeChallengeMethod, oauthEffectiveTokenPolicy(s.securitySettingRepo, client).RequirePKCE); oerr != nil {
		span.SetStatus(codes.Error, "pkce invalid")
		return nil, oerr
	}

	// Enforce MFA / step-up requirement. When the tenant enforces MFA
	// (RequiredACR = "2") or the client overrides require step-up, the
	// user must present a fresh acr=2 token to proceed.
	sessionPolicy := oauthEffectiveSessionPolicy(s.securitySettingRepo, client)
	if sessionPolicy.RequiredACR == "2" {
		claims := middleware.JWTClaimsFromContext(ctx)
		if claims == nil || claims.ACR != "2" {
			span.SetStatus(codes.Error, "step-up required")
			return nil, &apperror.OAuthError{
				Code:        "step_up_required",
				Description: "Step-up authentication required",
				StatusCode:  http.StatusForbidden,
			}
		}
		mfaPolicy := secpolicy.LoadMFAPolicy(s.securitySettingRepo, client.TenantID)
		stepUpTTL := mfaPolicy.StepUpTTLSeconds()
		if claims.Iat > 0 && time.Now().Unix()-claims.Iat > stepUpTTL {
			span.SetStatus(codes.Error, "step-up expired")
			return nil, &apperror.OAuthError{
				Code:        "step_up_required",
				Description: "Step-up authentication has expired; please re-authenticate",
				StatusCode:  http.StatusForbidden,
			}
		}
	}

	// OIDC Core §3.1.2.1: acr_values and max_age are how a relying party asks for
	// step-up authentication and for re-authentication at the PROTOCOL level. They
	// were parsed nowhere, so an RP that asked for either had a code issued off
	// whatever session already existed and no way to tell that its request had
	// been dropped.
	if oerr := s.enforceRequestedAuthContext(ctx, req); oerr != nil {
		span.SetStatus(codes.Error, "requested authentication context not satisfied")
		return nil, oerr
	}

	// Validate that the client supports the authorization_code grant.
	if !clientHasGrant(client, GrantTypeAuthorizationCode) {
		span.SetStatus(codes.Error, "grant type not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for authorization_code grant")
	}

	// Validate response_type against client configuration.
	if !s.clientSupportsResponseType(client, req.ResponseType) {
		span.SetStatus(codes.Error, "response type not supported")
		return nil, apperror.NewOAuthUnsupportedResponseType("response_type 'code' is not enabled for this client")
	}

	if oerr := validateClientAllowedScopes(client, req.Scope); oerr != nil {
		span.SetStatus(codes.Error, "scope not allowed")
		return nil, oerr
	}

	// Validate redirect_uri against registered URIs.
	if oerr := s.validateRedirectURI(client, req.RedirectURI); oerr != nil {
		span.SetStatus(codes.Error, "invalid redirect_uri")
		return nil, oerr
	}

	// Enforce state parameter for CSRF protection (RFC 6749 §10.12).
	if req.State == "" {
		span.SetStatus(codes.Error, "missing state")
		return nil, apperror.NewOAuthInvalidRequest("state parameter is required")
	}

	// Nonce is required when response_type includes id_token (OIDC Core §3.1.2.1).
	if strings.Contains(req.ResponseType, "id_token") && req.Nonce == "" {
		span.SetStatus(codes.Error, "missing nonce")
		return nil, apperror.NewOAuthInvalidRequest("nonce parameter is required when response_type includes id_token")
	}

	// Check if user has already consented to the requested scopes for this client.
	needsConsent, err := s.needsConsent(client, userID, req.Scope)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "consent check failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if needsConsent {
		if req.Prompt == "none" {
			span.SetStatus(codes.Error, "consent required during non-interactive authorization")
			return nil, apperror.NewOAuthConsentRequired("user consent is required")
		}
		// Create a consent challenge for the frontend.
		challenge, oerr := s.createConsentChallenge(ctx, client, userID, req)
		if oerr != nil {
			span.SetStatus(codes.Error, "consent challenge creation failed")
			return nil, oerr
		}
		span.SetStatus(codes.Ok, "consent required")
		return &OAuthAuthorizeResult{
			ConsentChallenge: challenge.OAuthConsentChallengeUUID.String(),
		}, nil
	}

	// User has already consented — issue authorization code directly.
	redirectURI, oerr := s.issueAuthorizationCode(ctx, client, userID, req)
	if oerr != nil {
		span.SetStatus(codes.Error, "authorization code issuance failed")
		return nil, oerr
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    client.TenantID,
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthAuthorize,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Authorization code issued"),
	})

	span.SetStatus(codes.Ok, "")
	return &OAuthAuthorizeResult{
		RedirectURI: redirectURI,
	}, nil
}

// enforceRequestedAuthContext applies the RP's acr_values and max_age to the
// session the caller already has.
//
// Both fail CLOSED: if the session's authentication facts cannot be read, the
// request is refused rather than approved on the assumption that the session is
// good enough. Refusal is the recoverable outcome — the RP re-runs /authorize
// after the user authenticates — whereas silently issuing a code claims an
// authentication strength the user never demonstrated.
func (s *oauthAuthorizeService) enforceRequestedAuthContext(ctx context.Context, req OAuthAuthorizeRequestDTO) *apperror.OAuthError {
	if req.ACRValues == "" && !req.MaxAgeSet {
		return nil
	}

	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil {
		return apperror.NewOAuthLoginRequired("authentication required")
	}

	if req.ACRValues != "" {
		satisfied := false
		for _, requested := range strings.Fields(req.ACRValues) {
			if requested == claims.ACR {
				satisfied = true
				break
			}
		}
		if !satisfied {
			// step_up_required mirrors what the tenant-policy branch above returns,
			// so the hosted identity app has one code to react to.
			return &apperror.OAuthError{
				Code:        "step_up_required",
				Description: "the requested acr_values are not satisfied by the current session",
				StatusCode:  http.StatusForbidden,
			}
		}
	}

	if req.MaxAgeSet {
		authTime, ok := s.sessionAuthTime(ctx, claims.SessionID)
		if !ok {
			return apperror.NewOAuthLoginRequired("re-authentication required")
		}
		if time.Since(authTime) > time.Duration(req.MaxAgeSeconds)*time.Second {
			return apperror.NewOAuthLoginRequired("re-authentication required")
		}
	}

	return nil
}

// sessionAuthTime reads when the session behind this request last actively
// authenticated. ok=false means "unknown", which callers must treat as a
// failure, not as "recent".
func (s *oauthAuthorizeService) sessionAuthTime(ctx context.Context, sessionID string) (time.Time, bool) {
	if s.db == nil || sessionID == "" {
		return time.Time{}, false
	}
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return time.Time{}, false
	}
	var row struct {
		AuthTime time.Time `gorm:"column:auth_time"`
	}
	if err := s.db.WithContext(ctx).
		Table("user_sessions").
		Select("auth_time").
		Where("user_session_uuid = ? AND revoked_at IS NULL", sessionUUID).
		Take(&row).Error; err != nil {
		return time.Time{}, false
	}
	if row.AuthTime.IsZero() {
		return time.Time{}, false
	}
	return row.AuthTime, true
}

// PrepareAuthorize implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) PrepareAuthorize(ctx context.Context, req OAuthAuthorizeRequestDTO) *apperror.OAuthError {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.prepare")
	defer span.End()
	span.SetAttributes(attribute.String("oauth.client_id", req.ClientID))

	client, err := s.resolveAuthorizeClient(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client lookup failed")
		return apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil || client.Status != shared.StatusActive {
		span.SetStatus(codes.Error, "client not found or inactive")
		return apperror.NewOAuthInvalidRequest("unknown or inactive client context")
	}
	// Tenant binding (SECURITY-CRITICAL): reject a cross-tenant client_id BEFORE
	// any login page is shown. When the request host resolves to a known subdomain
	// tenant (derived server-side from Origin/Host, not from a caller-supplied
	// tenant_id), the client_id MUST belong to that tenant — hard reject with NO
	// fallback to the system tenant. When the request host is not recognizable we
	// preserve the existing pre-login behavior (no tenant binding here).
	if requestTenantID, ok := resolveRequestTenantID(ctx); ok {
		if client.TenantID != requestTenantID {
			span.SetStatus(codes.Error, "client request-tenant mismatch")
			return apperror.NewOAuthInvalidRequest("unknown client_id")
		}
	}
	if oerr := s.validateRedirectURI(client, req.RedirectURI); oerr != nil {
		span.SetStatus(codes.Error, "invalid redirect_uri")
		return oerr
	}

	if req.ScreenHint == "signup" && req.RegistrationFlow != "" {
		if _, oerr := s.validateRegistrationFlowForAuthorize(client.ClientID, client.TenantID, req.RegistrationFlow); oerr != nil {
			return oerr
		}
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// validateRegistrationFlowForAuthorize pre-validates the registration flow named
// in an /authorize?screen_hint=signup request, so the hosted UI fails fast rather
// than at the end of the signup form.
//
// It must stay in lockstep with authn.registrationFlowByName — this
// endpoint is unauthenticated, so any check missing here becomes a probe for a
// flow the register endpoint would refuse: hence the tenant predicate, the
// system-flow refusal, and a single undifferentiated error for every failure
// (an "inactive" vs "unknown" split would confirm which flow names exist).
func (s *oauthAuthorizeService) validateRegistrationFlowForAuthorize(clientID, tenantID int64, name string) (int64, *apperror.OAuthError) {
	var flow struct {
		RegistrationFlowID int64
		Status             string
		IsSystem           bool
	}
	err := s.db.Table("registration_flows").
		Where("client_id = ? AND tenant_id = ? AND name = ? AND deleted_at IS NULL", clientID, tenantID, name).
		Select("registration_flow_id, status, is_system").
		First(&flow).Error
	if err != nil {
		return 0, apperror.NewOAuthInvalidRequest("unknown registration flow")
	}
	// System flows are invite-only; a self-service signup link must never select one.
	if flow.IsSystem || flow.Status != shared.StatusActive {
		return 0, apperror.NewOAuthInvalidRequest("unknown registration flow")
	}
	return flow.RegistrationFlowID, nil
}

func (s *oauthAuthorizeService) resolveAuthorizeClient(req OAuthAuthorizeRequestDTO) (*Client, error) {
	if req.ClientID != "" {
		client, err := s.clientRepo.FindByClientIDAndIdentityProvider(req.ClientID, "")
		if err != nil || client != nil {
			if client != nil && client.IsSystem && !isPublicOAuthSystemClientAllowed(client) {
				return nil, nil
			}
			return client, err
		}

		client, err = s.clientRepo.FindByIdentifier(req.ClientID)
		if err != nil || client == nil {
			return client, err
		}
		if client.IsSystem && !isPublicOAuthSystemClientAllowed(client) {
			return nil, nil
		}
		return client, nil
	}

	return nil, nil
}

// isPublicOAuthSystemClientAllowed reports whether a system client may drive a
// public authorize request. Only the two seeded surface clients qualify — the
// identity app needs it to broker its own first-party federated logins, and the
// console to sign its operators in. Their registered redirect_uris (plus PKCE)
// remain the gate on where any resulting code can be delivered.
func isPublicOAuthSystemClientAllowed(client *Client) bool {
	return isSeededSurfaceClient(client)
}

// GetConsentChallenge implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) GetConsentChallenge(ctx context.Context, challengeUUID uuid.UUID, userID int64) (*OAuthConsentChallengeResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.get_consent_challenge")
	defer span.End()
	span.SetAttributes(attribute.String("consent.challenge_uuid", challengeUUID.String()))

	challenge, err := s.consentChallRepo.FindChallengeByUUID(challengeUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "consent challenge lookup failed")
		return nil, apperror.NewInternal("failed to retrieve consent challenge", err)
	}

	if challenge == nil {
		span.SetStatus(codes.Error, "consent challenge not found")
		return nil, apperror.NewNotFoundWithReason("consent challenge not found")
	}

	if challenge.UserID != userID {
		span.SetStatus(codes.Error, "consent challenge user mismatch")
		return nil, apperror.NewForbidden("consent challenge does not belong to the authenticated user")
	}

	if challenge.IsExpired() {
		span.SetStatus(codes.Error, "consent challenge expired")
		return nil, apperror.NewValidation("consent challenge has expired")
	}

	scopes := []string(challenge.Scope)

	clientName := ""
	clientUUID := ""
	if challenge.Client != nil {
		clientName = challenge.Client.DisplayName
		clientUUID = challenge.Client.ClientUUID.String()
	}

	span.SetStatus(codes.Ok, "")
	return &OAuthConsentChallengeResponseDTO{
		ChallengeID: challenge.OAuthConsentChallengeUUID.String(),
		ClientName:  clientName,
		ClientUUID:  clientUUID,
		Scopes:      scopes,
		RedirectURI: challenge.RedirectURI,
		ExpiresAt:   challenge.ExpiresAt.Unix(),
	}, nil
}

// HandleConsent implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) HandleConsent(ctx context.Context, decision OAuthConsentDecisionDTO, userID int64) (*OAuthConsentDecisionResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.handle_consent")
	defer span.End()
	span.SetAttributes(
		attribute.String("consent.challenge_id", decision.ChallengeID),
		attribute.Bool("consent.approved", decision.Approved),
	)

	challengeUUID, _ := uuid.Parse(decision.ChallengeID) // validated by DTO

	challenge, err := s.consentChallRepo.FindChallengeByUUID(challengeUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "challenge lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if challenge == nil || challenge.IsExpired() {
		span.SetStatus(codes.Error, "challenge not found or expired")
		return nil, apperror.NewOAuthInvalidRequest("consent challenge not found or expired")
	}

	if challenge.UserID != userID {
		span.SetStatus(codes.Error, "challenge user mismatch")
		return nil, apperror.NewOAuthAccessDenied("consent challenge does not belong to the authenticated user")
	}

	state := ""
	if challenge.State != nil {
		state = *challenge.State
	}

	// User denied consent.
	if !decision.Approved {
		// Delete the challenge.
		if err := s.consentChallRepo.DeleteChallengeByUUID(challengeUUID); err != nil {
			span.RecordError(err)
		}

		s.authEventService.Log(ctx, authevent.AuthEventInput{
			TenantID:    challenge.TenantID,
			ActorUserID: &userID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategoryAuthn,
			EventType:   authevent.AuthEventTypeOAuthConsentDeny,
			Severity:    authevent.AuthEventSeverityInfo,
			Result:      authevent.AuthEventResultFailure,
			Description: ptr.Ptr("User denied consent"),
		})

		oauthErr := apperror.NewOAuthAccessDenied("the resource owner denied the request")
		span.SetStatus(codes.Ok, "consent denied")
		return &OAuthConsentDecisionResult{
			RedirectURI: oauthErr.RedirectURI(challenge.RedirectURI, state),
		}, nil
	}

	// User approved consent — save grant and issue code in a transaction.
	var redirectURI string
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		txConsentGrantRepo := s.consentGrantRepo.WithTx(tx)
		txAuthCodeRepo := s.authCodeRepo.WithTx(tx)
		txConsentChallRepo := s.consentChallRepo.WithTx(tx)

		// Persist the consent grant (upsert to handle scope expansion).
		if _, err := txConsentGrantRepo.Upsert(&OAuthConsentGrant{
			UserID:   userID,
			ClientID: challenge.ClientID,
			TenantID: challenge.TenantID,
			Scopes:   challenge.Scope,
		}); err != nil {
			return err
		}

		// Generate and store authorization code.
		rawCode, err := crypto.GenerateRandomString(authorizationCodeLength)
		if err != nil {
			return err
		}
		codeHash := crypto.HashAuthorizationCode(rawCode)

		authCode := &OAuthAuthorizationCode{
			CodeHash:            codeHash,
			ClientID:            challenge.ClientID,
			UserID:              userID,
			TenantID:            challenge.TenantID,
			RedirectURI:         challenge.RedirectURI,
			Scope:               challenge.Scope,
			State:               challenge.State,
			Nonce:               challenge.Nonce,
			CodeChallenge:       challenge.CodeChallenge,
			CodeChallengeMethod: challenge.CodeChallengeMethod,
			ExpiresAt:           time.Now().Add(authorizationCodeTTL),
			UserSessionUUID:     callerSessionUUID(ctx),
		}
		if _, err := txAuthCodeRepo.Create(authCode); err != nil {
			return err
		}

		// Build the redirect URI with the authorization code.
		redirectURI = buildAuthCodeRedirect(challenge.RedirectURI, rawCode, state)

		// Remove the challenge now that it has been resolved.
		return txConsentChallRepo.DeleteChallengeByUUID(challengeUUID)
	})

	if txErr != nil {
		span.RecordError(txErr)
		span.SetStatus(codes.Error, "consent grant transaction failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    challenge.TenantID,
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeOAuthConsent,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("User approved consent and authorization code issued"),
	})

	span.SetStatus(codes.Ok, "")
	return &OAuthConsentDecisionResult{
		RedirectURI: redirectURI,
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// findClientByIdentifier looks up a client by its OAuth identifier (the string
// stored in the clients.identifier column).
func (s *oauthAuthorizeService) findClientByIdentifier(identifier string) (*Client, error) {
	var client Client
	err := s.db.
		Preload("Tenant").
		Preload("ClientURIs").
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

// clientSupportsResponseType checks whether the client has the given
// response_type in its configuration.
func (s *oauthAuthorizeService) clientSupportsResponseType(client *Client, responseType string) bool {
	for _, rt := range client.ResponseTypes {
		if rt == responseType {
			return true
		}
	}
	return false
}

// validateRedirectURI checks the redirect_uri against the client's registered
// redirect URIs. Delegates to the shared client.MatchClientRedirectURI.
func (s *oauthAuthorizeService) validateRedirectURI(client *Client, redirectURI string) *apperror.OAuthError {
	var uris []clientpkg.RedirectURIMatch
	if client.ClientURIs != nil {
		uris = make([]clientpkg.RedirectURIMatch, len(*client.ClientURIs))
		for i, u := range *client.ClientURIs {
			uris[i] = clientpkg.RedirectURIMatch{URI: u.URI, Type: u.Type}
		}
	}
	if err := clientpkg.MatchClientRedirectURI(uris, redirectURI); err != nil {
		return apperror.NewOAuthInvalidRequest(err.Error())
	}
	return nil
}

// needsConsent determines whether the user needs to provide consent for the
// requested scopes. Consent is not required if the client has require_consent
// set to false or if the user has already consented to all requested scopes.
func (s *oauthAuthorizeService) needsConsent(client *Client, userID int64, requestedScope string) (bool, error) {
	if !client.RequireConsent {
		return false, nil
	}

	grant, err := s.consentGrantRepo.FindByUserAndClient(userID, client.ClientID)
	if err != nil {
		return false, err
	}

	if grant == nil {
		return true, nil
	}

	// Check that all requested scopes are covered by the existing grant.
	grantedScopes := []string(grant.Scopes)
	grantedSet := make(map[string]struct{}, len(grantedScopes))
	for _, s := range grantedScopes {
		grantedSet[s] = struct{}{}
	}

	for _, requested := range splitScopes(requestedScope) {
		if _, ok := grantedSet[requested]; !ok {
			return true, nil
		}
	}

	return false, nil
}

// createConsentChallenge persists a new consent challenge so the frontend can
// display the consent screen.
func (s *oauthAuthorizeService) createConsentChallenge(ctx context.Context, client *Client, userID int64, req OAuthAuthorizeRequestDTO) (*OAuthConsentChallenge, *apperror.OAuthError) {
	challenge := &OAuthConsentChallenge{
		ClientID:            client.ClientID,
		UserID:              userID,
		TenantID:            client.TenantID,
		RedirectURI:         req.RedirectURI,
		Scope:               parseScopeFields(req.Scope),
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ResponseType:        req.ResponseType,
		ExpiresAt:           time.Now().Add(consentChallengeTTL),
	}

	if req.State != "" {
		challenge.State = &req.State
	}
	if req.Nonce != "" {
		challenge.Nonce = &req.Nonce
	}

	if _, err := s.consentChallRepo.Create(challenge); err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	_ = ctx // used by authEventService.Log via caller
	return challenge, nil
}

// callerSessionUUID extracts the browser session the current request is
// authenticated with.
//
// OptionalUserContextMiddleware has already validated the caller's access-token
// cookie and parsed its `sid` claim into the request context, so the session is
// simply sitting there — it was never read. Carrying it onto the authorization
// code is what makes OAuth-minted tokens session-attributable: without it every
// console/identity token has no `sid`, and logout is forced to choose between
// revoking ALL of the user's sessions (signing them out of other browsers and
// their phone) or none at all.
//
// Returns nil when the request carried no session, which is legitimate for
// non-browser callers; the token then simply has no `sid`.
func callerSessionUUID(ctx context.Context) *uuid.UUID {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil || strings.TrimSpace(claims.SessionID) == "" {
		return nil
	}
	parsed, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil
	}
	return &parsed
}

// issueAuthorizationCode creates an authorization code and returns the full
// redirect URI with the code and state appended.
func (s *oauthAuthorizeService) issueAuthorizationCode(ctx context.Context, client *Client, userID int64, req OAuthAuthorizeRequestDTO) (string, *apperror.OAuthError) {
	rawCode, err := crypto.GenerateRandomString(authorizationCodeLength)
	if err != nil {
		return "", apperror.NewOAuthServerError("an unexpected error occurred")
	}
	codeHash := crypto.HashAuthorizationCode(rawCode)

	authCode := &OAuthAuthorizationCode{
		CodeHash:            codeHash,
		ClientID:            client.ClientID,
		UserID:              userID,
		TenantID:            client.TenantID,
		RedirectURI:         req.RedirectURI,
		Scope:               parseScopeFields(req.Scope),
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(authorizationCodeTTL),
		UserSessionUUID:     callerSessionUUID(ctx),
	}

	if req.State != "" {
		authCode.State = &req.State
	}
	if req.Nonce != "" {
		authCode.Nonce = &req.Nonce
	}

	if _, err := s.authCodeRepo.Create(authCode); err != nil {
		return "", apperror.NewOAuthServerError("an unexpected error occurred")
	}

	_ = ctx // used by authEventService.Log via caller
	return buildAuthCodeRedirect(req.RedirectURI, rawCode, req.State), nil
}

// buildAuthCodeRedirect appends code and optional state to a redirect URI.
//
// The parameters are percent-encoded rather than concatenated raw. `state` is
// client-controlled and arrives here already URL-DECODED, so a value containing
// `&` or `#` used to re-partition the callback query: state "x&code=other"
// was pasted in verbatim, injecting an extra query parameter into the redirect,
// and a `#` truncated everything after it into a fragment. RFC 6749 §4.1.2
// requires these to be added as properly encoded query parameters.
func buildAuthCodeRedirect(redirectURI, code, state string) string {
	params := url.Values{}
	params.Set("code", code)
	if state != "" {
		params.Set("state", state)
	}
	return appendQueryParams(redirectURI, params)
}

// appendQueryParams merges encoded parameters onto a redirect URI, preserving
// any query string the registered URI already carries.
func appendQueryParams(redirectURI string, params url.Values) string {
	if len(params) == 0 {
		return redirectURI
	}
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		// The redirect URI was matched against the client's registered set before
		// reaching here, so a parse failure is not expected. Fall back to simple
		// appending — with the values still encoded, which is the part that
		// matters.
		sep := "?"
		if strings.Contains(redirectURI, "?") {
			sep = "&"
		}
		return redirectURI + sep + params.Encode()
	}
	query := parsed.Query()
	for key, values := range params {
		for _, value := range values {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// splitScopes splits a space-delimited scope string into a slice.
func splitScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	parts := strings.Fields(scope)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

const authorizeRequestTTL = 10 * time.Minute

func (s *oauthAuthorizeService) PrepareAuthorizeSignup(ctx context.Context, req OAuthAuthorizeRequestDTO) (string, string, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.prepare_authorize_signup")
	defer span.End()

	client, err := s.resolveAuthorizeClient(req)
	if err != nil {
		span.RecordError(err)
		return "", "", apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil || client.Status != shared.StatusActive {
		return "", "", apperror.NewOAuthInvalidRequest("unknown or inactive client context")
	}
	if oerr := s.validateRedirectURI(client, req.RedirectURI); oerr != nil {
		return "", "", oerr
	}

	var flowID int64
	if req.ScreenHint == "signup" && req.RegistrationFlow != "" {
		id, oerr := s.validateRegistrationFlowForAuthorize(client.ClientID, client.TenantID, req.RegistrationFlow)
		if oerr != nil {
			return "", "", oerr
		}
		flowID = id
	}

	var regFlowID *int64
	if flowID != 0 {
		regFlowID = &flowID
	}

	// Browser-binding secret: the raw value is returned to the caller for storage
	// in an httpOnly cookie; only its hash is persisted. ContinueAuthorize later
	// requires the matching secret, so a leaked request_id cannot be continued
	// from a different session in the same tenant.
	bindingSecret, err := crypto.GenerateRandomString(32)
	if err != nil {
		span.RecordError(err)
		return "", "", apperror.NewOAuthServerError("an unexpected error occurred")
	}

	authReq := &OAuthAuthorizeRequest{
		ClientID:            client.ClientID,
		TenantID:            client.TenantID,
		RedirectURI:         req.RedirectURI,
		ResponseType:        req.ResponseType,
		Scope:               parseScopeFields(req.Scope),
		State:               ptr.PtrOrNil(req.State),
		Nonce:               ptr.PtrOrNil(req.Nonce),
		CodeChallenge:       ptr.PtrOrNil(req.CodeChallenge),
		CodeChallengeMethod: ptr.PtrOrNil(req.CodeChallengeMethod),
		BindingHash:         ptr.Ptr(crypto.HashOAuthBindingToken(bindingSecret)),
		ScreenHint:          ptr.Ptr(req.ScreenHint),
		RegistrationFlowID:  regFlowID,
		Status:              "pending",
		ExpiresAt:           time.Now().Add(authorizeRequestTTL),
	}
	if _, err := s.authReqRepo.Create(authReq); err != nil {
		span.RecordError(err)
		return "", "", apperror.NewOAuthServerError("an unexpected error occurred")
	}
	span.SetStatus(codes.Ok, "")
	return authReq.OAuthAuthorizeRequestUUID.String(), bindingSecret, nil
}

func (s *oauthAuthorizeService) ContinueAuthorize(ctx context.Context, requestID string, bindingSecret string, userID int64, tenantID int64) (*OAuthAuthorizeResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.continue_authorize")
	defer span.End()

	requestUUID, err := uuid.Parse(requestID)
	if err != nil {
		return nil, apperror.NewOAuthInvalidRequest("invalid request_id")
	}

	savedReq, err := s.authReqRepo.FindByUUID(requestUUID)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if savedReq == nil {
		return nil, apperror.NewOAuthInvalidRequest("authorize request not found or already used")
	}
	if savedReq.IsExpired() {
		return nil, apperror.NewOAuthInvalidRequest("authorize request has expired")
	}

	// Session binding (SECURITY): the request is bound to the browser that
	// initiated it via an httpOnly cookie set at prepare time. Require the
	// matching secret so a different authenticated session in the same tenant
	// cannot consume another user's pending request with a leaked request_id.
	if savedReq.BindingHash == nil || *savedReq.BindingHash == "" {
		span.SetStatus(codes.Error, "authorize request missing session binding")
		return nil, apperror.NewOAuthInvalidRequest("authorize request is not bound to a session")
	}
	if subtle.ConstantTimeCompare(
		[]byte(*savedReq.BindingHash),
		[]byte(crypto.HashOAuthBindingToken(bindingSecret)),
	) != 1 {
		span.SetStatus(codes.Error, "authorize request session-binding mismatch")
		return nil, apperror.NewOAuthInvalidRequest("authorize request binding mismatch")
	}

	client, err := s.clientRepo.FindByID(savedReq.ClientID)
	if err != nil || client == nil {
		return nil, apperror.NewOAuthInvalidRequest("unknown client context")
	}
	if client.Status != shared.StatusActive {
		return nil, apperror.NewOAuthInvalidRequest("client is inactive")
	}

	req := OAuthAuthorizeRequestDTO{
		ResponseType:        savedReq.ResponseType,
		ClientID:            ptrOrEmpty(client.Identifier),
		RedirectURI:         savedReq.RedirectURI,
		Scope:               strings.Join([]string(savedReq.Scope), " "),
		State:               ptrOrEmpty(savedReq.State),
		Nonce:               ptrOrEmpty(savedReq.Nonce),
		CodeChallenge:       ptrOrEmpty(savedReq.CodeChallenge),
		CodeChallengeMethod: ptrOrEmpty(savedReq.CodeChallengeMethod),
	}

	var result *OAuthAuthorizeResult
	var authOErr *apperror.OAuthError
	if txErr := s.db.Transaction(func(tx *gorm.DB) error {
		txConsumeRepo := s.authReqRepo.WithTx(tx)
		if err := txConsumeRepo.Consume(savedReq.OAuthAuthorizeRequestID, tenantID, time.Now()); err != nil {
			return err
		}
		txSvc := *s
		txSvc.clientRepo = s.clientRepo.WithTx(tx)
		txSvc.clientURIRepo = s.clientURIRepo.WithTx(tx)
		txSvc.authCodeRepo = s.authCodeRepo.WithTx(tx)
		txSvc.consentGrantRepo = s.consentGrantRepo.WithTx(tx)
		txSvc.consentChallRepo = s.consentChallRepo.WithTx(tx)
		txSvc.authReqRepo = txConsumeRepo
		result, authOErr = txSvc.Authorize(ctx, req, userID, tenantID)
		if authOErr != nil {
			return authOErr
		}
		return nil
	}); txErr != nil {
		span.RecordError(txErr)
		if errors.Is(txErr, ErrAlreadyConsumed) {
			return nil, apperror.NewOAuthInvalidRequest("authorize request has already been used")
		}
		if authOErr != nil {
			return nil, authOErr
		}
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}
