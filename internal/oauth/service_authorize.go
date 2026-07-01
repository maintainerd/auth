package oauth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	clientpkg "github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
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

	// GetConsentChallenge retrieves a pending consent challenge by its UUID.
	GetConsentChallenge(ctx context.Context, challengeUUID uuid.UUID, userID int64) (*OAuthConsentChallengeResponseDTO, error)

	// HandleConsent processes the user's consent decision. On approval, it
	// persists the consent grant and issues an authorization code. On denial,
	// it returns a redirect with an error.
	HandleConsent(ctx context.Context, decision OAuthConsentDecisionDTO, userID int64) (*OAuthConsentDecisionResult, *apperror.OAuthError)

	// PrepareAuthorizeSignup persists the authorize request for a signup flow
	// (screen_hint=signup, no session) and returns the request_id for the SPA.
	PrepareAuthorizeSignup(ctx context.Context, req OAuthAuthorizeRequestDTO) (string, *apperror.OAuthError)

	// ContinueAuthorize resumes a persisted authorize request after registration,
	// issuing an authorization code bound to the authenticated user.
	ContinueAuthorize(ctx context.Context, requestID string, userID int64, tenantID int64) (*OAuthAuthorizeResult, *apperror.OAuthError)
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

	if client.TenantID != tenantID {
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
	if oerr := s.validateRedirectURI(client, req.RedirectURI); oerr != nil {
		span.SetStatus(codes.Error, "invalid redirect_uri")
		return oerr
	}

	if req.ScreenHint == "signup" && req.RegistrationFlow != "" {
		if oerr := s.validateRegistrationFlowForAuthorize(client.ClientID, req.RegistrationFlow); oerr != nil {
			return oerr
		}
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *oauthAuthorizeService) validateRegistrationFlowForAuthorize(clientID int64, identifier string) *apperror.OAuthError {
	var flow struct {
		Status string
	}
	err := s.db.Table("registration_flows").
		Where("client_id = ? AND identifier = ? AND deleted_at IS NULL", clientID, identifier).
		Select("status").
		First(&flow).Error
	if err != nil {
		return apperror.NewOAuthInvalidRequest("unknown registration flow")
	}
	if flow.Status != shared.StatusActive {
		return apperror.NewOAuthInvalidRequest("registration flow is inactive")
	}
	return nil
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

func isPublicOAuthSystemClientAllowed(client *Client) bool {
	return client != nil && client.Name == shared.SystemClientNameAuthConsole
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

	scopes := splitScopes(challenge.Scope)

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
	grantedScopes := splitScopes(grant.Scopes)
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
		Scope:               req.Scope,
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
		Scope:               req.Scope,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(authorizationCodeTTL),
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
func buildAuthCodeRedirect(redirectURI, code, state string) string {
	sep := "?"
	for _, c := range redirectURI {
		if c == '?' {
			sep = "&"
			break
		}
	}
	result := redirectURI + sep + "code=" + code
	if state != "" {
		result += "&state=" + state
	}
	return result
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

func (s *oauthAuthorizeService) PrepareAuthorizeSignup(ctx context.Context, req OAuthAuthorizeRequestDTO) (string, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.prepare_authorize_signup")
	defer span.End()

	client, err := s.resolveAuthorizeClient(req)
	if err != nil {
		span.RecordError(err)
		return "", apperror.NewOAuthServerError("an unexpected error occurred")
	}
	if client == nil || client.Status != shared.StatusActive {
		return "", apperror.NewOAuthInvalidRequest("unknown or inactive client context")
	}
	if oerr := s.validateRedirectURI(client, req.RedirectURI); oerr != nil {
		return "", oerr
	}

	authReq := &OAuthAuthorizeRequest{
		ClientID:            client.ClientID,
		RedirectURI:         req.RedirectURI,
		ResponseType:        req.ResponseType,
		Scope:               ptr.PtrOrNil(req.Scope),
		State:               ptr.PtrOrNil(req.State),
		Nonce:               ptr.PtrOrNil(req.Nonce),
		CodeChallenge:       ptr.PtrOrNil(req.CodeChallenge),
		CodeChallengeMethod: ptr.PtrOrNil(req.CodeChallengeMethod),
		ScreenHint:          ptr.Ptr(req.ScreenHint),
		RegistrationFlow:    ptr.PtrOrNil(req.RegistrationFlow),
		Status:              "pending",
		ExpiresAt:           time.Now().Add(authorizeRequestTTL),
	}
	if _, err := s.authReqRepo.Create(authReq); err != nil {
		span.RecordError(err)
		return "", apperror.NewOAuthServerError("an unexpected error occurred")
	}
	span.SetStatus(codes.Ok, "")
	return authReq.OAuthAuthorizeRequestUUID.String(), nil
}

func (s *oauthAuthorizeService) ContinueAuthorize(ctx context.Context, requestID string, userID int64, tenantID int64) (*OAuthAuthorizeResult, *apperror.OAuthError) {
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
		Scope:               ptrOrEmpty(savedReq.Scope),
		State:               ptrOrEmpty(savedReq.State),
		Nonce:               ptrOrEmpty(savedReq.Nonce),
		CodeChallenge:       ptrOrEmpty(savedReq.CodeChallenge),
		CodeChallengeMethod: ptrOrEmpty(savedReq.CodeChallengeMethod),
		RegistrationFlow:    ptrOrEmpty(savedReq.RegistrationFlow),
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
