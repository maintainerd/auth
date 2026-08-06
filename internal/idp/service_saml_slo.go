package idp

import (
	"context"
	"net/http"
	"strings"

	crewsaml "github.com/crewjam/saml"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	jwtlib "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// idpValidateIDTokenHint validates the id_token_hint presented at the SLO
// initiate endpoint. A var so tests can stub it without minting real keys.
var idpValidateIDTokenHint = jwtlib.ValidateTokenWithContext

// samlLogoutRevokeReason is the reason stamped on sessions ended by SAML Single
// Logout. It must be one of the values user_sessions.revoked_reason allows
// (migration 046).
const samlLogoutRevokeReason = "logout"

// InitiateSAMLLogout starts SP-initiated SAML Single Logout.
//
// SLO is global by definition — the IdP is being told "this principal's session
// is over", and SAML 2.0 Core §3.7 has every participating SP terminate the
// session it holds. So unlike an ordinary per-session logout, this revokes every
// local session for the resolved user, and it does so BEFORE handing the browser
// to the IdP: if the user never comes back from the IdP (closed tab, IdP error),
// the sessions here must already be gone.
func (s *federationService) InitiateSAMLLogout(ctx context.Context, in SAMLLogoutInitiateInput) (*SAMLLogoutInitiateResult, error) {
	ctx, span := otel.Tracer("service").Start(ctx, "federation.initiate_saml_logout")
	defer span.End()
	span.SetAttributes(attribute.String("provider_identifier", in.ProviderIdentifier))

	if in.ProviderIdentifier == "" || in.IDTokenHint == "" {
		return nil, apperror.NewValidation("provider_identifier and id_token_hint are required")
	}

	idp, err := s.idpRepo.FindByIdentifier(in.ProviderIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	cfg, err := parseSAMLConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation(err.Error())
	}
	if strings.TrimSpace(cfg.SLOURL) == "" {
		return nil, apperror.NewValidation("single logout is not configured for this identity provider")
	}

	identity, user, err := s.resolveSAMLLogoutSubject(ctx, idp, in.IDTokenHint)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	postLogoutRedirectURI, err := s.resolveSAMLPostLogoutRedirect(idp, in.ClientID, in.PostLogoutRedirectURI)
	if err != nil {
		return nil, err
	}

	if err := s.revokeSAMLLogoutSessions(ctx, user.UserID, "SAML single logout (SP-initiated)"); err != nil {
		return nil, err
	}

	entityID, acsURL, metadataURL, sloURL := samlSPURLs(in.ProviderIdentifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL, sloURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to build SAML SP", err)
	}
	destination := sp.GetSLOBindingLocation(crewsaml.HTTPRedirectBinding)
	if destination == "" {
		return nil, apperror.NewValidation("single logout is not configured for this identity provider")
	}

	// Build the LogoutRequest first so its ID can be carried in the RelayState:
	// the IdP echoes it back as InResponseTo, which is what lets the SLO endpoint
	// prove the response answers the logout we actually started.
	logoutReq, err := sp.MakeLogoutRequest(destination, identity.Sub)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate SAML LogoutRequest", err)
	}
	relayState, err := newSAMLLogoutRelayState(in.ProviderIdentifier, in.ClientID, postLogoutRedirectURI, idp.TenantID, logoutReq.ID)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate relay state", err)
	}

	span.SetStatus(codes.Ok, "")
	return &SAMLLogoutInitiateResult{RedirectURL: logoutReq.Redirect(relayState).String()}, nil
}

// resolveSAMLLogoutSubject validates the id_token_hint and resolves the identity
// it names WITHIN the given provider.
//
// The hint is the only credential this endpoint has, so it is checked the way
// any other bearer credential is: signature, claims, and token type. Only an ID
// token is accepted (OIDC RP-Initiated Logout §2), and the subject must resolve
// to an identity issued by THIS provider — otherwise a token minted for one
// provider could drive a logout against another.
func (s *federationService) resolveSAMLLogoutSubject(ctx context.Context, idp *IdentityProvider, idTokenHint string) (*UserIdentity, *User, error) {
	claims, err := idpValidateIDTokenHint(ctx, idTokenHint)
	if err != nil {
		return nil, nil, apperror.NewUnauthorized("id_token_hint is invalid or expired")
	}
	if tokenType, _ := claims["token_type"].(string); tokenType != jwtlib.TokenTypeID {
		return nil, nil, apperror.NewUnauthorized("id_token_hint must be an ID token")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, nil, apperror.NewUnauthorized("id_token_hint has no subject")
	}

	identity, err := s.userIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, sub)
	if err != nil {
		return nil, nil, apperror.NewInternal("identity lookup failed", err)
	}
	if identity == nil {
		return nil, nil, apperror.NewUnauthorized("no identity for this provider")
	}

	user, err := s.userRepo.FindByID(identity.UserID)
	if err != nil || user == nil {
		return nil, nil, apperror.NewInternal("user lookup failed", err)
	}
	return identity, user, nil
}

// resolveSAMLPostLogoutRedirect validates the landing page the caller wants the
// browser on once the IdP is done.
//
// It is validated HERE, before it is sealed into the signed RelayState, because
// the SLO endpoint redirects to it without re-deriving it. An unvalidated value
// would be an open redirect on a URL the IdP hands straight back to the browser.
// Only a URI registered on the client as a logout_uri qualifies.
func (s *federationService) resolveSAMLPostLogoutRedirect(idp *IdentityProvider, clientID, candidate string) (string, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", nil
	}
	if clientID == "" {
		return "", apperror.NewValidation("client_id is required when post_logout_redirect_uri is supplied")
	}
	if err := security.ValidateRedirectURI(candidate); err != nil {
		return "", apperror.NewValidation("post_logout_redirect_uri is not allowed: " + err.Error())
	}

	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientID, idp.Identifier)
	if err != nil || client == nil {
		return "", apperror.NewNotFound("client not found for this provider")
	}
	uris, err := s.clientRepo.FindRedirectURIs(client.ClientID)
	if err != nil {
		return "", apperror.NewInternal("logout URI lookup failed", err)
	}
	for _, uri := range uris {
		if uri.Type == shared.ClientURITypeLogout && uri.URI == candidate {
			return candidate, nil
		}
	}
	return "", apperror.NewValidation("post_logout_redirect_uri is not registered for this client")
}

// revokeSAMLLogoutSessions ends every session the user holds here.
//
// A nil session service is a hard failure, never a skip: continuing would send
// the browser off to the IdP and report a completed logout while the local
// session — the thing an attacker actually rides — is still live.
func (s *federationService) revokeSAMLLogoutSessions(ctx context.Context, userID int64, description string) error {
	if s.sessionService == nil {
		return apperror.NewInternal("session service not configured", nil)
	}
	if err := s.sessionService.RevokeAllSessions(ctx, userID, samlLogoutRevokeReason); err != nil {
		return apperror.NewInternal("session revocation failed", err)
	}
	if s.authEventService != nil {
		s.authEventService.Log(ctx, authevent.AuthEventInput{
			ActorUserID: &userID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    authevent.AuthEventCategorySession,
			EventType:   authevent.AuthEventTypeSessionExpired,
			Severity:    authevent.AuthEventSeverityInfo,
			Result:      authevent.AuthEventResultSuccess,
			Description: ptr.Ptr(description),
		})
	}
	return nil
}

// HandleSAMLSingleLogout serves the SLO endpoint advertised in our SP metadata.
// It handles both directions of SAML 2.0 Single Logout:
//
//   - a LogoutResponse, completing a logout this server started; and
//   - a LogoutRequest, an IdP-initiated logout this server must honour and
//     answer.
//
// Every inbound message must be signed by the provider's configured certificate
// (see readSAMLLogoutMessage) — the endpoint is unauthenticated by protocol
// design, so the signature is the whole trust decision.
func (s *federationService) HandleSAMLSingleLogout(ctx context.Context, r *http.Request, providerIdentifier string) (*SAMLSingleLogoutResult, error) {
	ctx, span := otel.Tracer("service").Start(ctx, "federation.saml_single_logout")
	defer span.End()
	span.SetAttributes(attribute.String("provider_identifier", providerIdentifier))

	idp, err := s.idpRepo.FindByIdentifier(providerIdentifier)
	if err != nil || idp == nil {
		return nil, apperror.NewNotFound("identity provider not found")
	}
	if idp.Status != "active" {
		return nil, apperror.NewValidation("identity provider is not active")
	}

	cfg, err := parseSAMLConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation(err.Error())
	}
	if strings.TrimSpace(cfg.SLOURL) == "" {
		return nil, apperror.NewValidation("single logout is not configured for this identity provider")
	}

	cert, err := parsePEMCertificate(cfg.Certificate)
	if err != nil {
		return nil, apperror.NewValidation("identity provider certificate is not usable")
	}

	message, err := readSAMLLogoutMessage(r, cert)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "saml logout message rejected")
		return nil, apperror.NewUnauthorized("SAML logout message rejected: " + err.Error())
	}

	_, _, _, sloURL := samlSPURLs(providerIdentifier)
	if message.IsRequest {
		return s.handleIDPInitiatedSAMLLogout(ctx, idp, cfg, message, sloURL)
	}
	return s.completeSPInitiatedSAMLLogout(ctx, idp, message, sloURL)
}

// handleIDPInitiatedSAMLLogout honours a LogoutRequest sent by the IdP and
// returns the LogoutResponse redirect the browser should follow back.
func (s *federationService) handleIDPInitiatedSAMLLogout(
	ctx context.Context,
	idp *IdentityProvider,
	cfg *SAMLProviderConfig,
	message *samlInboundMessage,
	sloURL string,
) (*SAMLSingleLogoutResult, error) {
	request, err := parseSAMLLogoutRequest(message.XML, cfg.EntityID, sloURL)
	if err != nil {
		return nil, apperror.NewUnauthorized(err.Error())
	}
	if err := s.enforceSAMLLogoutRequestSingleUse(ctx, request.ID); err != nil {
		return nil, err
	}

	identity, err := s.userIdentityRepo.FindByTenantProviderAndSub(idp.TenantID, idp.Provider, request.NameID)
	if err != nil {
		return nil, apperror.NewInternal("identity lookup failed", err)
	}

	loggedOut := false
	if identity != nil {
		if err := s.revokeSAMLLogoutSessions(ctx, identity.UserID, "SAML single logout (IdP-initiated)"); err != nil {
			return nil, err
		}
		loggedOut = true
	}
	// An unknown NameID still gets a Success LogoutResponse. There is genuinely
	// no session here to end, and answering differently would turn the SLO
	// endpoint into an oracle for "does this subject have an account here".

	entityID, acsURL, metadataURL, _ := samlSPURLs(idp.Identifier)
	sp, err := buildSAMLServiceProvider(cfg, entityID, acsURL, metadataURL, sloURL)
	if err != nil {
		return nil, apperror.NewInternal("failed to build SAML SP", err)
	}
	responseURL, err := sp.MakeRedirectLogoutResponse(request.ID, message.RelayState)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate SAML LogoutResponse", err)
	}
	return &SAMLSingleLogoutResult{RedirectURL: responseURL.String(), LoggedOut: loggedOut}, nil
}

// completeSPInitiatedSAMLLogout finishes a logout this server started: it binds
// the IdP's LogoutResponse to our signed, single-use RelayState and returns the
// post-logout landing page validated at initiate time.
func (s *federationService) completeSPInitiatedSAMLLogout(
	ctx context.Context,
	idp *IdentityProvider,
	message *samlInboundMessage,
	sloURL string,
) (*SAMLSingleLogoutResult, error) {
	rs, err := verifyRelayStateForPurpose(message.RelayState, samlRelayPurposeSLO)
	if err != nil {
		return nil, apperror.NewUnauthorized("invalid or expired relay state")
	}
	if rs.ProviderIdentifier != idp.Identifier {
		return nil, apperror.NewUnauthorized("relay state belongs to another identity provider")
	}
	if err := s.enforceRelayStateSingleUse(ctx, rs.Nonce); err != nil {
		return nil, err
	}

	cfg, err := parseSAMLConfig(idp)
	if err != nil {
		return nil, apperror.NewValidation(err.Error())
	}
	if _, err := parseSAMLLogoutResponse(message.XML, cfg.EntityID, sloURL, rs.RequestID); err != nil {
		return nil, apperror.NewUnauthorized(err.Error())
	}

	// Sessions were already revoked when the logout was initiated, so a response
	// that never arrives cannot leave the user signed in here.
	return &SAMLSingleLogoutResult{RedirectURL: rs.RedirectURI, LoggedOut: true}, nil
}

// enforceSAMLLogoutRequestSingleUse rejects a LogoutRequest ID that has already
// been processed. Without it a captured, still-fresh LogoutRequest can be
// replayed to terminate the subject's sessions again and again.
func (s *federationService) enforceSAMLLogoutRequestSingleUse(ctx context.Context, requestID string) error {
	seen, err := s.markSAMLMessageSeen(ctx, samlLogoutRequestNoncePrefix+requestID)
	if err != nil {
		return err
	}
	if seen {
		return apperror.NewUnauthorized("SAML logout request has already been processed")
	}
	return nil
}

// markSAMLMessageSeen records a single-use marker and reports whether it was
// already present. A missing store is an error, never a pass: without it there
// is no replay protection at all.
func (s *federationService) markSAMLMessageSeen(ctx context.Context, key string) (bool, error) {
	if s.samlStore == nil {
		return false, apperror.NewInternal("SAML session store not configured", nil)
	}
	var seen bool
	if err := s.samlStore.GetSession(ctx, key, &seen); err == nil {
		return true, nil
	}
	if err := s.samlStore.SetSession(ctx, key, true, samlRelayNonceTTL); err != nil {
		return false, apperror.NewInternal("failed to record SAML message use", err)
	}
	return false, nil
}
