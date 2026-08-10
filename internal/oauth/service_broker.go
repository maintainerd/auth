package oauth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const brokerSessionTTL = 10 * time.Minute

// brokerProviderResolver resolves upstream provider authorize parameters for the
// broker flow. Set once at startup via SetBrokerProviderResolver so the oauth
// package never imports the idp domain or handles provider secrets.
var brokerProviderResolver BrokerProviderResolver

// SetBrokerProviderResolver installs the provider resolver that StartBroker uses
// to obtain the upstream authorize endpoint, client_id, and scopes. Call once
// during app startup before serving requests.
func SetBrokerProviderResolver(r BrokerProviderResolver) {
	brokerProviderResolver = r
}

// brokerCallbackResolver resolves the maintainerd user for the broker callback
// by exchanging the upstream provider's authorization code and provisioning the
// identity. Set once at startup.
var brokerCallbackResolver BrokerCallbackResolver

// SetBrokerCallbackResolver installs the callback resolver. Call once during
// app startup before serving requests.
func SetBrokerCallbackResolver(r BrokerCallbackResolver) {
	brokerCallbackResolver = r
}

// brokerAccountLinkVerifier checks whether a confirmation token has been
// confirmed and returns the linked user's ID. Set once at startup.
var brokerAccountLinkVerifier AccountLinkVerifier

// SetBrokerAccountLinkVerifier installs the account-link verifier used by
// BrokerResume. Call once during app startup before serving requests.
func SetBrokerAccountLinkVerifier(v AccountLinkVerifier) {
	brokerAccountLinkVerifier = v
}

// HandleCallback implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) StartBroker(ctx context.Context, req OAuthAuthorizeRequestDTO) (*OAuthAuthorizeResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.start_broker")
	defer span.End()
	span.SetAttributes(
		attribute.String("oauth.client_id", req.ClientID),
		attribute.String("oauth.idp_hint", req.IdpHint),
	)

	if brokerProviderResolver == nil {
		span.SetStatus(codes.Error, "broker resolver not configured")
		return nil, apperror.NewOAuthServerError("identity provider brokering is not configured")
	}

	// Resolve and validate the downstream OAuth client + its redirect_uri.
	client, err := s.resolveAuthorizeClient(req)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil || client.Status != shared.StatusActive {
		return nil, apperror.NewOAuthInvalidRequest("unknown or inactive client context")
	}
	if oerr := s.validateRedirectURI(client, req.RedirectURI); oerr != nil {
		return nil, oerr
	}

	// idp_hint must be an enabled connection on this client.
	conn, oerr := s.findEnabledConnection(client, req.IdpHint)
	if oerr != nil {
		return nil, oerr
	}
	idp := conn.IdentityProvider

	// SAML providers do not speak OAuth2 and have no provider_client_id, so the
	// broker leg cannot start one. Reject with an actionable error instead of
	// letting it round-trip into an opaque server_error deep in provider
	// resolution — a SAML connection is reached through the SAML initiate
	// endpoint, not idp_hint on /authorize.
	if idp != nil && idp.ProviderType == shared.IDPTypeSAML {
		return nil, apperror.NewOAuthInvalidRequest("this identity provider uses SAML; start it from the SAML sign-in endpoint, not idp_hint")
	}

	// Resolve the upstream provider's authorize endpoint + client_id + scopes
	// (decrypts the provider config — secrets never leave this call).
	provider, perr := brokerProviderResolver.ResolveBrokerProvider(ctx, req.IdpHint)
	if perr != nil {
		span.RecordError(perr)
		span.SetStatus(codes.Error, "provider resolution failed")
		return nil, apperror.NewOAuthServerError("failed to resolve identity provider configuration")
	}

	// Per-attempt state, PKCE, and nonce for the upstream (OAuth #2) leg.
	idpState, err := crypto.GenerateRandomString(32)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	idpNonce, err := crypto.GenerateRandomString(24)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	// PKCE for the upstream (OAuth #2) leg — only for providers that support it.
	// LinkedIn does NOT support PKCE: presenting a code_challenge/verifier together
	// with the client_secret makes its token endpoint reject the exchange as
	// invalid_client ("Client authentication failed"). So LinkedIn's broker leg
	// runs as a plain confidential-client authorization_code flow (no challenge,
	// no verifier). An empty verifier signals the exchange to omit code_verifier.
	var verifier, challenge string
	if upstreamSupportsPKCE(idp.Provider) {
		verifier, err = crypto.GenerateRandomString(48)
		if err != nil {
			return nil, apperror.NewOAuthServerError("an unexpected error occurred")
		}
		challenge = crypto.ComputeS256Challenge(verifier)
	}

	// Persist the broker session that correlates OAuth #1 ↔ OAuth #2.
	session := &OAuthBrokerSession{
		TenantID:                   client.TenantID,
		ClientID:                   client.ClientID,
		IdentityProviderID:         idp.IdentityProviderID,
		IdentityProviderIdentifier: idp.Identifier,
		AppRedirectURI:             req.RedirectURI,
		AppState:                   ptr.PtrOrNil(req.State),
		AppScope:                   parseScopeFields(req.Scope),
		AppNonce:                   ptr.PtrOrNil(req.Nonce),
		AppCodeChallenge:           ptr.PtrOrNil(req.CodeChallenge),
		AppCodeChallengeMethod:     ptr.PtrOrNil(req.CodeChallengeMethod),
		IdpState:                   idpState,
		IdpPKCEVerifier:            verifier,
		IdpNonce:                   ptr.Ptr(idpNonce),
		ExpiresAt:                  time.Now().Add(brokerSessionTTL),
	}
	if _, err := NewOAuthBrokerSessionRepository(s.db).Create(session); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "broker session create failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Build the upstream provider's authorize URL.
	redirectURL := buildBrokerAuthorizeURL(provider, req.IdpHint, idpState, challenge, idpNonce)

	span.SetStatus(codes.Ok, "")
	return &OAuthAuthorizeResult{RedirectURI: redirectURL}, nil
}

// findEnabledConnection returns the client_identity_providers row (with its
// IdentityProvider) for the given client and idp_hint when the connection is
// enabled and the provider is active. Returns an OAuth error when none match.
func (s *oauthAuthorizeService) findEnabledConnection(client *Client, idpHint string) (*ClientIdentityProvider, *apperror.OAuthError) {
	var conns []ClientIdentityProvider
	if err := s.db.
		Preload("IdentityProvider").
		Where("client_id = ? AND enabled = ?", client.ClientID, true).
		Find(&conns).Error; err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	for i := range conns {
		idp := conns[i].IdentityProvider
		if idp != nil && idp.Status == shared.StatusActive && idp.Identifier == idpHint {
			if idp.TenantID != client.TenantID || conns[i].TenantID != client.TenantID {
				return nil, apperror.NewOAuthInvalidRequest("the requested identity provider is not enabled for this client")
			}
			return &conns[i], nil
		}
	}
	return nil, apperror.NewOAuthInvalidRequest("the requested identity provider is not enabled for this client")
}

// upstreamSupportsPKCE reports whether the upstream provider supports the PKCE
// extension (code_challenge/code_verifier) on its authorization-code flow.
// LinkedIn does not: it rejects a code_challenge/verifier presented alongside a
// client_secret with invalid_client. Every other supported provider does.
func upstreamSupportsPKCE(providerSlug string) bool {
	return providerSlug != shared.IDPProviderLinkedIn
}

// buildBrokerAuthorizeURL constructs the upstream provider's authorize URL with
// the resolved parameters. The callback is /api/v1/oauth/callback/{idp} on the
// public host — the identity app will mount its handler there (B5). A blank
// challenge omits PKCE (for providers that don't support it, e.g. LinkedIn).
func buildBrokerAuthorizeURL(provider *BrokerProvider, idpIdentifier, idpState, challenge, idpNonce string) string {
	callback := buildBrokerCallbackURL(idpIdentifier)
	q := url.Values{}
	q.Set("client_id", provider.ClientID)
	q.Set("redirect_uri", callback)
	q.Set("response_type", "code")
	if len(provider.Scopes) > 0 {
		q.Set("scope", strings.Join(provider.Scopes, " "))
	}
	q.Set("state", idpState)
	if challenge != "" {
		q.Set("code_challenge", challenge)
		q.Set("code_challenge_method", "S256")
	}
	q.Set("nonce", idpNonce)

	sep := "?"
	if strings.Contains(provider.AuthorizationEndpoint, "?") {
		sep = "&"
	}
	return provider.AuthorizationEndpoint + sep + q.Encode()
}

func buildBrokerCallbackURL(idpIdentifier string) string {
	return strings.TrimRight(config.AppPublicHostname, "/") + "/api/v1/oauth/callback/" + url.PathEscape(idpIdentifier)
}

// ResolveBrokerErrorRedirect returns a URL back into the identity login UI for a
// failed brokered login (an `error` from the upstream IdP, a missing code, or a
// failed exchange), so the browser is never left on the API callback endpoint.
// It resolves the tenant's identity host from the broker session when possible
// (and consumes that session so it can't be replayed); on any miss it falls back
// to the system-tenant login. It never returns an error.
func (s *oauthAuthorizeService) ResolveBrokerErrorRedirect(ctx context.Context, idpIdentifier, state, errCode, errDesc string) string {
	_ = ctx
	_ = idpIdentifier

	tenantName := ""
	tenantIsSystem := true

	if st := strings.TrimSpace(state); st != "" {
		sessionRepo := NewOAuthBrokerSessionRepository(s.db)
		if session, err := sessionRepo.FindByIdpState(st); err == nil && session != nil {
			_ = sessionRepo.Consume(session.OAuthBrokerSessionID, time.Now())
			var t Tenant
			if e := s.db.Select("name", "is_system").Where("tenant_id = ?", session.TenantID).First(&t).Error; e == nil {
				tenantName = t.Name
				tenantIsSystem = t.IsSystem
			}
		}
	}

	// Allowlist the error CODE. This endpoint is unauthenticated and the code can
	// originate from an attacker-crafted callback URL, so an unrecognized value is
	// collapsed to access_denied rather than reflected back onto the trusted auth
	// origin. (url.Values.Encode already percent-encodes, so this is defense in
	// depth against content injection, not the only guard.)
	errCode = normalizeBrokerErrorCode(errCode)

	loginURL := shared.FrontendURL(shared.FrontendSurfaceIdentity, tenantName, tenantIsSystem, "/login")
	q := url.Values{}
	q.Set("error", errCode)
	// error_description is only ever set by trusted, developer-authored callers
	// (this file and the handler's canonical upstream-error messages). The raw
	// upstream error_description is deliberately NOT plumbed here — see
	// handler_callback.go, which maps the upstream error CODE to a fixed message.
	if strings.TrimSpace(errDesc) != "" {
		q.Set("error_description", errDesc)
	}
	return loginURL + "?" + q.Encode()
}

// brokerErrorCodes is the set of OAuth2 error codes (RFC 6749 §4.1.2.1 + OIDC)
// this server will echo back to the login UI. Anything else becomes
// access_denied so an attacker cannot reflect arbitrary text into the code slot.
var brokerErrorCodes = map[string]struct{}{
	"invalid_request": {}, "unauthorized_client": {}, "access_denied": {},
	"unsupported_response_type": {}, "invalid_scope": {}, "server_error": {},
	"temporarily_unavailable": {}, "interaction_required": {}, "login_required": {},
	"consent_required": {}, "invalid_grant": {},
}

func normalizeBrokerErrorCode(code string) string {
	code = strings.TrimSpace(code)
	if _, ok := brokerErrorCodes[code]; ok {
		return code
	}
	return "access_denied"
}

// CanonicalUpstreamErrorMessage maps an upstream IdP's OAuth error code to a
// FIXED, safe, user-facing message. The upstream provider's own
// error_description is attacker-influenceable on an unauthenticated callback, so
// it is never forwarded — only this curated text is shown.
func CanonicalUpstreamErrorMessage(code string) string {
	switch normalizeBrokerErrorCode(code) {
	case "access_denied":
		return "You declined or were denied access at the identity provider."
	case "invalid_scope":
		return "The identity provider rejected the requested permissions."
	case "temporarily_unavailable":
		return "The identity provider is temporarily unavailable. Please try again."
	case "interaction_required", "login_required", "consent_required":
		return "The identity provider needs additional interaction to sign you in."
	default:
		return "Sign-in with the identity provider could not be completed."
	}
}

// HandleCallback implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) HandleCallback(ctx context.Context, idpIdentifier, code, state string) (string, string, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.handle_broker_callback")
	defer span.End()

	state = strings.TrimSpace(state)
	code = strings.TrimSpace(code)
	idpIdentifier = strings.TrimSpace(idpIdentifier)

	// Look up the broker session by the idp_state we sent in the authorize URL.
	sessionRepo := NewOAuthBrokerSessionRepository(s.db)
	session, err := sessionRepo.FindByIdpState(state)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "broker session lookup failed")
		return "", "", apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if session == nil || session.IsExpired() || session.IsConsumed() {
		return "", "", apperror.NewOAuthInvalidRequest("broker session is expired, already used, or invalid")
	}
	if session.IdentityProviderIdentifier == "" || session.IdentityProviderIdentifier != idpIdentifier {
		return "", "", apperror.NewOAuthInvalidRequest("broker callback identity provider does not match the login session")
	}
	// Exchange the provider code and resolve the user.
	if brokerCallbackResolver == nil {
		return "", "", apperror.NewOAuthServerError("broker callback not configured")
	}
	nonce := ""
	if session.IdpNonce != nil {
		nonce = *session.IdpNonce
	}
	callbackURL := buildBrokerCallbackURL(idpIdentifier)
	resolved, perr := brokerCallbackResolver.ResolveBrokerUser(ctx, session.IdentityProviderID, code, session.IdpPKCEVerifier, nonce, callbackURL, session.ClientID)
	if perr != nil {
		span.RecordError(perr)
		span.SetStatus(codes.Error, "provider user resolution failed")
		// User-actionable refusals (JIT provisioning disabled, registration
		// closed, unverifiable identity, already-linked conflicts) surface with
		// their reason so the login page can say WHY the sign-in was refused —
		// a bare server_error reads as an outage and gives the user nothing to
		// act on. Internal failures stay opaque.
		var unauth *apperror.UnauthorizedError
		var validation *apperror.ValidationError
		var conflict *apperror.ConflictError
		switch {
		case errors.As(perr, &unauth):
			return "", "", apperror.NewOAuthAccessDenied(unauth.Reason)
		case errors.As(perr, &validation):
			return "", "", apperror.NewOAuthAccessDenied(validation.Reason)
		case errors.As(perr, &conflict):
			return "", "", apperror.NewOAuthAccessDenied(conflict.Reason)
		}
		return "", "", apperror.NewOAuthServerError("failed to authenticate with identity provider")
	}

	// Account link required: the broker resolved a collision. Redirect to the
	// identity app for explicit user confirmation instead of issuing an auth code.
	if resolved.AccountLinkToken != "" {
		// Derive the identity frontend host for the session's tenant. A regular
		// tenant links accounts on its own subdomain; the system tenant uses the
		// bare host. This must distinguish system vs regular to pick the subdomain.
		linkTenantName := ""
		linkTenantIsSystem := true
		var linkTenant Tenant
		if err := s.db.Select("name", "is_system").Where("tenant_id = ?", session.TenantID).First(&linkTenant).Error; err == nil {
			linkTenantName = linkTenant.Name
			linkTenantIsSystem = linkTenant.IsSystem
		}
		linkURL := shared.FrontendURL(shared.FrontendSurfaceIdentity, linkTenantName, linkTenantIsSystem, "/account-link") +
			"?token=" + url.QueryEscape(resolved.AccountLinkToken) +
			"&provider=" + url.QueryEscape(resolved.AccountLinkProvider) +
			"&email=" + url.QueryEscape(resolved.AccountLinkEmail) +
			"&broker_session=" + url.QueryEscape(session.OAuthBrokerSessionUUID.String())
		span.SetStatus(codes.Ok, "account_link_required")
		return linkURL, "", nil
	}

	// Reconstruct the original OAuth #1 request from the stored session.
	req := OAuthAuthorizeRequestDTO{
		ResponseType:        "code",
		RedirectURI:         session.AppRedirectURI,
		State:               ptrOrEmpty(session.AppState),
		Scope:               strings.Join([]string(session.AppScope), " "),
		Nonce:               ptrOrEmpty(session.AppNonce),
		CodeChallenge:       ptrOrEmpty(session.AppCodeChallenge),
		CodeChallengeMethod: ptrOrEmpty(session.AppCodeChallengeMethod),
	}

	// Load the downstream app client from the broker session.
	appClient, cerr := s.clientRepo.FindByID(session.ClientID)
	if cerr != nil || appClient == nil {
		span.RecordError(cerr)
		span.SetStatus(codes.Error, "app client lookup failed")
		return "", "", apperror.NewOAuthInvalidRequest("unknown client context")
	}
	if appClient.TenantID != session.TenantID {
		return "", "", apperror.NewOAuthInvalidRequest("broker session tenant does not match the client")
	}

	// The broker just authenticated this user and created a real user_sessions
	// row (resolved.SessionID). Bind it to the authorization code explicitly:
	// this request is a bare redirect from the upstream IdP, so there are no
	// JWT claims in context for callerSessionUUID to read, and a code without
	// a session mints `sid`-less tokens that session validation rejects on
	// every authenticated endpoint ("Token is not bound to a session").
	var brokerSessionUUID *uuid.UUID
	if parsed, perr := uuid.Parse(strings.TrimSpace(resolved.SessionID)); perr == nil {
		brokerSessionUUID = &parsed
	}

	// Atomically mark the broker session consumed and issue our own authorization
	// code bound to the app client + resolved user.
	redirectURL := ""
	var issueOErr *apperror.OAuthError
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := sessionRepo.WithTx(tx).Consume(session.OAuthBrokerSessionID, time.Now()); err != nil {
			return err
		}
		txSvc := *s
		txSvc.authCodeRepo = s.authCodeRepo.WithTx(tx)
		var oerr *apperror.OAuthError
		redirectURL, oerr = txSvc.issueAuthorizationCode(ctx, appClient, resolved.UserID, req, brokerSessionUUID)
		if oerr != nil {
			issueOErr = oerr
			return oerr
		}
		return nil
	}); err != nil {
		span.RecordError(err)
		if errors.Is(err, ErrAlreadyConsumed) {
			span.SetStatus(codes.Error, "broker session consume failed")
			return "", "", apperror.NewOAuthInvalidRequest("broker session has already been used")
		}
		if issueOErr != nil {
			span.SetStatus(codes.Error, "authorization code issuance failed")
			return "", "", issueOErr
		}
		span.SetStatus(codes.Error, "broker callback transaction failed")
		return "", "", apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// No SSO cookie is set here. The broker callback runs on the ISSUER host
	// (APP_PUBLIC_HOSTNAME / identity-api), whereas the hosted identity app reads
	// its session same-origin on the identity FRONTEND host with host-only
	// __Host- cookies — a cookie set here could never be read there. The identity
	// app instead establishes its own session same-origin: a downstream app
	// (console) that wants an external-provider login is routed by the identity
	// SPA through the identity app's OWN first-party broker login, whose
	// /callback exchanges the code on the identity host (see OAuthAuthorizePage /
	// OAuthCallbackPage). Minting a token here only produced a valid access token
	// as a cookie on the issuer host that nothing ever read — dead attack surface
	// — so it is deliberately not done.
	span.SetStatus(codes.Ok, "")
	return redirectURL, "", nil
}

// BrokerResume implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) BrokerResume(ctx context.Context, req BrokerResumeRequestDTO, userID, authTenantID int64) (*BrokerResumeResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.broker_resume")
	defer span.End()

	if brokerAccountLinkVerifier == nil {
		return nil, apperror.NewOAuthServerError("account link verifier not configured")
	}

	// Validate the link token is confirmed and belongs to the authenticated user.
	linkedUserID, found, err := brokerAccountLinkVerifier.FindConfirmedLink(req.AccountLinkToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "link token lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if !found {
		return nil, apperror.NewOAuthInvalidRequest("account link token is not confirmed or has expired")
	}
	if linkedUserID != userID {
		return nil, apperror.NewOAuthInvalidRequest("account link token does not belong to the authenticated user")
	}

	// Load the broker session.
	sessionRepo := NewOAuthBrokerSessionRepository(s.db)
	sessionUUID, uuidErr := uuid.Parse(req.BrokerSessionUUID)
	if uuidErr != nil {
		return nil, apperror.NewOAuthInvalidRequest("invalid broker session uuid")
	}
	session, err := sessionRepo.FindByUUID(sessionUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "broker session lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if session == nil || session.IsExpired() || session.IsConsumed() {
		return nil, apperror.NewOAuthInvalidRequest("broker session is expired, already used, or not found")
	}
	// Cross-tenant guard: the broker session UUID is caller-supplied, so bind it
	// to the authenticated tenant. Without this, an attacker holding a
	// self-confirmed link token in tenant A who learns a tenant-B broker session
	// UUID could mint an auth code for their own user against tenant B's client.
	// Broker session UUIDs are unguessable v4, but that is an accident of
	// generation, not an authorization check — enforce the boundary explicitly.
	if session.TenantID != authTenantID {
		return nil, apperror.NewOAuthInvalidRequest("broker session does not belong to the authenticated tenant")
	}

	// Load the downstream app client.
	appClient, cerr := s.clientRepo.FindByID(session.ClientID)
	if cerr != nil || appClient == nil {
		return nil, apperror.NewOAuthInvalidRequest("unknown client context")
	}
	if appClient.TenantID != authTenantID {
		return nil, apperror.NewOAuthInvalidRequest("client does not belong to the authenticated tenant")
	}

	// Reconstruct the original authorize request from the stored session.
	authorizeReq := OAuthAuthorizeRequestDTO{
		ResponseType:        "code",
		RedirectURI:         session.AppRedirectURI,
		State:               ptrOrEmpty(session.AppState),
		Scope:               strings.Join([]string(session.AppScope), " "),
		Nonce:               ptrOrEmpty(session.AppNonce),
		CodeChallenge:       ptrOrEmpty(session.AppCodeChallenge),
		CodeChallengeMethod: ptrOrEmpty(session.AppCodeChallengeMethod),
	}

	// Issue auth code and consume broker session atomically. BrokerResume runs
	// on an authenticated route (the user just confirmed the link with their
	// existing account), so the caller's own browser session is in context and
	// is exactly the session the code — and every token minted from it — must
	// be bound to.
	var redirectURL string
	var issueOErr *apperror.OAuthError
	if txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := sessionRepo.WithTx(tx).Consume(session.OAuthBrokerSessionID, time.Now()); err != nil {
			return err
		}
		txSvc := *s
		txSvc.authCodeRepo = s.authCodeRepo.WithTx(tx)
		var oerr *apperror.OAuthError
		redirectURL, oerr = txSvc.issueAuthorizationCode(ctx, appClient, userID, authorizeReq, callerSessionUUID(ctx))
		if oerr != nil {
			issueOErr = oerr
			return oerr
		}
		return nil
	}); txErr != nil {
		if errors.Is(txErr, ErrAlreadyConsumed) {
			return nil, apperror.NewOAuthInvalidRequest("broker session has already been used")
		}
		if issueOErr != nil {
			return nil, issueOErr
		}
		span.RecordError(txErr)
		span.SetStatus(codes.Error, "broker resume transaction failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Single-use: retire the confirmed link token now that it has minted a code.
	// The broker session is already consumed (above), so a replayed token is
	// inert, but retiring it makes the single-use property explicit rather than
	// incidental. Best-effort — the redirect has already been produced.
	_, _ = brokerAccountLinkVerifier.ConsumeConfirmedLink(req.AccountLinkToken)

	// No new SSO cookie is minted here: the resume route is only reachable with
	// a live first-party session (the user authenticated to confirm the link),
	// so the browser already holds valid session cookies, and the authorization
	// code above is bound to that same session. The previous mint here could
	// never succeed anyway — it passed an empty client_id, which the JWT layer
	// rejects — so nothing ever consumed the token it pretended to return.
	span.SetStatus(codes.Ok, "")
	return &BrokerResumeResult{RedirectURL: redirectURL}, nil
}
