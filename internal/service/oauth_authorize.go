package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/dto"
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
	// authorizationCodeTTL is the lifetime of an authorization code (RFC 6749 §4.1.2).
	authorizationCodeTTL = 10 * time.Minute
	// consentChallengeTTL is the lifetime of a consent challenge.
	consentChallengeTTL = 10 * time.Minute
	// authorizationCodeLength is the byte length of the raw authorization code.
	authorizationCodeLength = 32
)

// OAuthAuthorizeService handles the OAuth 2.0 authorization endpoint logic.
type OAuthAuthorizeService interface {
	// Authorize processes an authorization request. It validates the client,
	// redirect URI, and PKCE parameters. Depending on whether consent is needed,
	// it either issues an authorization code immediately or creates a consent
	// challenge for the frontend to resolve.
	Authorize(ctx context.Context, req dto.OAuthAuthorizeRequestDTO, userID int64) (*dto.OAuthAuthorizeResult, *apperror.OAuthError)

	// GetConsentChallenge retrieves a pending consent challenge by its UUID.
	GetConsentChallenge(ctx context.Context, challengeUUID uuid.UUID, userID int64) (*dto.OAuthConsentChallengeResponseDTO, error)

	// HandleConsent processes the user's consent decision. On approval, it
	// persists the consent grant and issues an authorization code. On denial,
	// it returns a redirect with an error.
	HandleConsent(ctx context.Context, decision dto.OAuthConsentDecisionDTO, userID int64) (*dto.OAuthConsentDecisionResult, *apperror.OAuthError)
}

type oauthAuthorizeService struct {
	db               *gorm.DB
	clientRepo       repository.ClientRepository
	clientURIRepo    repository.ClientURIRepository
	authCodeRepo     repository.OAuthAuthorizationCodeRepository
	consentGrantRepo repository.OAuthConsentGrantRepository
	consentChallRepo repository.OAuthConsentChallengeRepository
	authEventService AuthEventService
}

// NewOAuthAuthorizeService creates a new OAuthAuthorizeService.
func NewOAuthAuthorizeService(
	db *gorm.DB,
	clientRepo repository.ClientRepository,
	clientURIRepo repository.ClientURIRepository,
	authCodeRepo repository.OAuthAuthorizationCodeRepository,
	consentGrantRepo repository.OAuthConsentGrantRepository,
	consentChallRepo repository.OAuthConsentChallengeRepository,
	authEventService AuthEventService,
) OAuthAuthorizeService {
	return &oauthAuthorizeService{
		db:               db,
		clientRepo:       clientRepo,
		clientURIRepo:    clientURIRepo,
		authCodeRepo:     authCodeRepo,
		consentGrantRepo: consentGrantRepo,
		consentChallRepo: consentChallRepo,
		authEventService: authEventService,
	}
}

// Authorize implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) Authorize(ctx context.Context, req dto.OAuthAuthorizeRequestDTO, userID int64) (*dto.OAuthAuthorizeResult, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_authorize.authorize")
	defer span.End()
	span.SetAttributes(
		attribute.String("oauth.client_id", req.ClientID),
		attribute.String("oauth.response_type", req.ResponseType),
		attribute.Int64("user.id", userID),
	)

	// Look up the client by its public identifier.
	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(req.ClientID, "")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client lookup failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// If FindByClientIDAndIdentityProvider requires both params, use a direct
	// lookup by identifier instead.
	if client == nil {
		client, err = s.findClientByIdentifier(req.ClientID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "client lookup failed")
			return nil, apperror.NewOAuthServerError("an unexpected error occurred")
		}
	}

	if client == nil || client.Status != model.StatusActive {
		span.SetStatus(codes.Error, "client not found or inactive")
		return nil, apperror.NewOAuthInvalidRequest("unknown or inactive client_id")
	}

	// Validate that the client supports the authorization_code grant.
	if !s.clientSupportsGrant(client, model.GrantTypeAuthorizationCode) {
		span.SetStatus(codes.Error, "grant type not allowed")
		return nil, apperror.NewOAuthUnauthorizedClient("client is not authorized for authorization_code grant")
	}

	// Validate response_type against client configuration.
	if !s.clientSupportsResponseType(client, req.ResponseType) {
		span.SetStatus(codes.Error, "response type not supported")
		return nil, apperror.NewOAuthUnsupportedResponseType("response_type 'code' is not enabled for this client")
	}

	// Validate redirect_uri against registered URIs.
	if oerr := s.validateRedirectURI(client, req.RedirectURI); oerr != nil {
		span.SetStatus(codes.Error, "invalid redirect_uri")
		return nil, oerr
	}

	// Check if user has already consented to the requested scopes for this client.
	needsConsent, err := s.needsConsent(client, userID, req.Scope)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "consent check failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	if needsConsent {
		// Create a consent challenge for the frontend.
		challenge, oerr := s.createConsentChallenge(ctx, client, userID, req)
		if oerr != nil {
			span.SetStatus(codes.Error, "consent challenge creation failed")
			return nil, oerr
		}
		span.SetStatus(codes.Ok, "consent required")
		return &dto.OAuthAuthorizeResult{
			ConsentChallenge: challenge.OAuthConsentChallengeUUID.String(),
		}, nil
	}

	// User has already consented — issue authorization code directly.
	redirectURI, oerr := s.issueAuthorizationCode(ctx, client, userID, req)
	if oerr != nil {
		span.SetStatus(codes.Error, "authorization code issuance failed")
		return nil, oerr
	}

	s.authEventService.Log(ctx, AuthEventInput{
		TenantID:    client.TenantID,
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    model.AuthEventCategoryAuthn,
		EventType:   model.AuthEventTypeOAuthAuthorize,
		Severity:    model.AuthEventSeverityInfo,
		Result:      model.AuthEventResultSuccess,
		Description: ptr.Ptr("Authorization code issued"),
	})

	span.SetStatus(codes.Ok, "")
	return &dto.OAuthAuthorizeResult{
		RedirectURI: redirectURI,
	}, nil
}

// GetConsentChallenge implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) GetConsentChallenge(ctx context.Context, challengeUUID uuid.UUID, userID int64) (*dto.OAuthConsentChallengeResponseDTO, error) {
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
	return &dto.OAuthConsentChallengeResponseDTO{
		ChallengeID: challenge.OAuthConsentChallengeUUID.String(),
		ClientName:  clientName,
		ClientUUID:  clientUUID,
		Scopes:      scopes,
		RedirectURI: challenge.RedirectURI,
		ExpiresAt:   challenge.ExpiresAt.Unix(),
	}, nil
}

// HandleConsent implements OAuthAuthorizeService.
func (s *oauthAuthorizeService) HandleConsent(ctx context.Context, decision dto.OAuthConsentDecisionDTO, userID int64) (*dto.OAuthConsentDecisionResult, *apperror.OAuthError) {
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

		s.authEventService.Log(ctx, AuthEventInput{
			TenantID:    challenge.TenantID,
			ActorUserID: &userID,
			IPAddress:   middleware.ClientIPFromContext(ctx),
			UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
			Category:    model.AuthEventCategoryAuthn,
			EventType:   model.AuthEventTypeOAuthConsentDeny,
			Severity:    model.AuthEventSeverityInfo,
			Result:      model.AuthEventResultFailure,
			Description: ptr.Ptr("User denied consent"),
		})

		oauthErr := apperror.NewOAuthAccessDenied("the resource owner denied the request")
		span.SetStatus(codes.Ok, "consent denied")
		return &dto.OAuthConsentDecisionResult{
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
		if _, err := txConsentGrantRepo.Upsert(&model.OAuthConsentGrant{
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

		authCode := &model.OAuthAuthorizationCode{
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

	s.authEventService.Log(ctx, AuthEventInput{
		TenantID:    challenge.TenantID,
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    model.AuthEventCategoryAuthn,
		EventType:   model.AuthEventTypeOAuthConsent,
		Severity:    model.AuthEventSeverityInfo,
		Result:      model.AuthEventResultSuccess,
		Description: ptr.Ptr("User approved consent and authorization code issued"),
	})

	span.SetStatus(codes.Ok, "")
	return &dto.OAuthConsentDecisionResult{
		RedirectURI: redirectURI,
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

// findClientByIdentifier looks up a client by its OAuth identifier (the string
// stored in the clients.identifier column).
func (s *oauthAuthorizeService) findClientByIdentifier(identifier string) (*model.Client, error) {
	var client model.Client
	err := s.db.
		Preload("IdentityProvider").
		Preload("IdentityProvider.Tenant").
		Preload("ClientURIs").
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

// clientSupportsGrant checks whether the client has the given grant_type in
// its configuration.
func (s *oauthAuthorizeService) clientSupportsGrant(client *model.Client, grantType string) bool {
	for _, g := range client.GrantTypes {
		if g == grantType {
			return true
		}
	}
	return false
}

// clientSupportsResponseType checks whether the client has the given
// response_type in its configuration.
func (s *oauthAuthorizeService) clientSupportsResponseType(client *model.Client, responseType string) bool {
	for _, rt := range client.ResponseTypes {
		if rt == responseType {
			return true
		}
	}
	return false
}

// validateRedirectURI checks the redirect_uri against the client's registered
// redirect URIs. Exact match is required per RFC 6749 §3.1.2.3.
func (s *oauthAuthorizeService) validateRedirectURI(client *model.Client, redirectURI string) *apperror.OAuthError {
	if client.ClientURIs == nil {
		return apperror.NewOAuthInvalidRequest("no redirect URIs registered for this client")
	}

	for _, uri := range *client.ClientURIs {
		if uri.Type == model.ClientURITypeRedirect && uri.URI == redirectURI {
			return nil
		}
	}

	return apperror.NewOAuthInvalidRequest("redirect_uri does not match any registered redirect URIs")
}

// needsConsent determines whether the user needs to provide consent for the
// requested scopes. Consent is not required if the client has require_consent
// set to false or if the user has already consented to all requested scopes.
func (s *oauthAuthorizeService) needsConsent(client *model.Client, userID int64, requestedScope string) (bool, error) {
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
func (s *oauthAuthorizeService) createConsentChallenge(ctx context.Context, client *model.Client, userID int64, req dto.OAuthAuthorizeRequestDTO) (*model.OAuthConsentChallenge, *apperror.OAuthError) {
	challenge := &model.OAuthConsentChallenge{
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
func (s *oauthAuthorizeService) issueAuthorizationCode(ctx context.Context, client *model.Client, userID int64, req dto.OAuthAuthorizeRequestDTO) (string, *apperror.OAuthError) {
	rawCode, err := crypto.GenerateRandomString(authorizationCodeLength)
	if err != nil {
		return "", apperror.NewOAuthServerError("an unexpected error occurred")
	}
	codeHash := crypto.HashAuthorizationCode(rawCode)

	authCode := &model.OAuthAuthorizationCode{
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
