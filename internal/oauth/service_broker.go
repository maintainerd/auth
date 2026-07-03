package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
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
	verifier, err := crypto.GenerateRandomString(48)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	idpNonce, err := crypto.GenerateRandomString(24)
	if err != nil {
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	challenge := crypto.ComputeS256Challenge(verifier)

	// Persist the broker session that correlates OAuth #1 ↔ OAuth #2.
	session := &OAuthBrokerSession{
		TenantID:                   client.TenantID,
		ClientID:                   client.ClientID,
		IdentityProviderID:         idp.IdentityProviderID,
		IdentityProviderIdentifier: idp.Identifier,
		AppRedirectURI:             req.RedirectURI,
		AppState:                   ptr.PtrOrNil(req.State),
		AppScope:                   ptr.PtrOrNil(req.Scope),
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

// buildBrokerAuthorizeURL constructs the upstream provider's authorize URL with
// the resolved parameters. The callback is /api/v1/oauth/callback/{idp} on the
// public host — the identity app will mount its handler there (B5).
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
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
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
		return "", "", apperror.NewOAuthServerError("failed to authenticate with identity provider")
	}

	// Reconstruct the original OAuth #1 request from the stored session.
	req := OAuthAuthorizeRequestDTO{
		ResponseType:        "code",
		RedirectURI:         session.AppRedirectURI,
		State:               ptrOrEmpty(session.AppState),
		Scope:               ptrOrEmpty(session.AppScope),
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
		redirectURL, oerr = txSvc.issueAuthorizationCode(ctx, appClient, resolved.UserID, req)
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

	// Generate a maintainerd session token so the user has an SSO cookie for
	// subsequent /authorize calls. Soft-fail — the redirect URL always wins.
	accessToken := ""
	if tok, gerr := jwt.GenerateAccessTokenWithOptionsContext(ctx,
		resolved.IdentitySub,
		"openid",
		strings.TrimRight(config.AppPublicHostname, "/"),
		strings.TrimRight(config.AppPublicHostname, "/"),
		ptrOrEmpty(appClient.Identifier),
		fmt.Sprintf("tenant:%d", session.TenantID),
		&jwt.AccessTokenOptions{AccessTokenTTL: 30 * time.Minute, SessionID: resolved.SessionID, ACR: jwt.ACRLevel1},
	); gerr == nil {
		accessToken = tok
	}

	span.SetStatus(codes.Ok, "")
	return redirectURL, accessToken, nil
}
