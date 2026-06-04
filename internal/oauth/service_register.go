package oauth

import (
	"context"
	"time"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const (
	clientIDLength     = 24
	clientSecretLength = 48
)

var (
	oauthRegisterGenerateRandomString = crypto.GenerateRandomString
	oauthRegisterHashClientSecret     = security.HashClientSecret
)

// OAuthRegisterService handles Dynamic Client Registration (RFC 7591).
type OAuthRegisterService interface {
	// Register creates a new OAuth client from the supplied metadata and returns
	// the client_id (and client_secret for confidential clients).
	Register(ctx context.Context, req OAuthClientRegistrationRequestDTO) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError)
}

type oauthRegisterService struct {
	db               *gorm.DB
	clientRepo       ClientRepository
	clientURIRepo    ClientURIRepository
	tenantRepo       TenantRepository
	authEventService authevent.AuthEventService
}

// NewOAuthRegisterService creates a new OAuthRegisterService.
func NewOAuthRegisterService(
	db *gorm.DB,
	clientRepo ClientRepository,
	clientURIRepo ClientURIRepository,
	tenantRepo TenantRepository,
	authEventService authevent.AuthEventService,
) OAuthRegisterService {
	return &oauthRegisterService{
		db:               db,
		clientRepo:       clientRepo,
		clientURIRepo:    clientURIRepo,
		tenantRepo:       tenantRepo,
		authEventService: authEventService,
	}
}

// Register implements OAuthRegisterService.
func (s *oauthRegisterService) Register(ctx context.Context, req OAuthClientRegistrationRequestDTO) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_register.register")
	defer span.End()

	// Require at least one redirect URI per RFC 7591 §2.
	if len(req.RedirectURIs) == 0 {
		return nil, apperror.NewOAuthInvalidRequest("at least one redirect_uri is required")
	}

	// Default grant type is authorization_code when not specified.
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{GrantTypeAuthorizationCode}
	}

	// Default response type mirrors the grant type.
	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	// Resolve target tenant: use system tenant for unauthenticated registration.
	tenant, err := s.tenantRepo.FindSystem()
	if err != nil || tenant == nil {
		span.SetStatus(codes.Error, "tenant resolution failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	rawClientID, err := oauthRegisterGenerateRandomString(clientIDLength)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	clientName := security.SanitizeInput(req.ClientName)
	if clientName == "" {
		clientName = "Registered Client"
	}

	client := &Client{
		TenantID:      tenant.TenantID,
		DisplayName:   clientName,
		Identifier:    ptr.Ptr(rawClientID),
		Status:        shared.StatusActive,
		GrantTypes:    grantTypes,
		ResponseTypes: responseTypes,
		ClientType:    "public",
	}

	// Confidential clients get a generated secret (hashed at rest, returned once).
	clientSecret := ""
	tokenEndpointAuthMethod := req.TokenEndpointAuthMethod
	if tokenEndpointAuthMethod == "" {
		tokenEndpointAuthMethod = TokenAuthMethodSecretBasic
	}
	if tokenEndpointAuthMethod != "none" {
		rawSecret, serr := oauthRegisterGenerateRandomString(clientSecretLength)
		if serr != nil {
			span.RecordError(serr)
			return nil, apperror.NewOAuthServerError("an unexpected error occurred")
		}
		secretHash, herr := oauthRegisterHashClientSecret(ctx, rawSecret)
		if herr != nil {
			span.RecordError(herr)
			return nil, apperror.NewOAuthServerError("an unexpected error occurred")
		}
		secretEncrypted, encErr := crypto.EncryptAtRest(rawSecret)
		if encErr != nil {
			span.RecordError(encErr)
			return nil, apperror.NewOAuthServerError("an unexpected error occurred")
		}
		clientSecret = rawSecret
		client.SecretHash = ptr.Ptr(secretHash)
		client.SecretEncrypted = ptr.Ptr(secretEncrypted)
		client.ClientType = "confidential"
	}

	createdClient, err := s.clientRepo.Create(client)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client creation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Register redirect URIs.
	for _, uri := range req.RedirectURIs {
		safeURI := security.SanitizeInput(uri)
		if safeURI == "" {
			continue
		}
		clientURI := &ClientURI{
			TenantID: tenant.TenantID,
			ClientID: createdClient.ClientID,
			URI:      safeURI,
			Type:     shared.ClientURITypeRedirect,
		}
		if _, uerr := s.clientURIRepo.Create(clientURI); uerr != nil {
			span.RecordError(uerr)
		}
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenant.TenantID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategorySystem,
		EventType:   authevent.AuthEventTypeOAuthAuthorize,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("Dynamic client registration"),
	})

	span.SetAttributes(attribute.Int64("client.id", createdClient.ClientID))
	span.SetStatus(codes.Ok, "")

	resp := &OAuthClientRegistrationResponseDTO{
		ClientID:                rawClientID,
		ClientIDIssuedAt:        time.Now().Unix(),
		ClientSecretExpiresAt:   0, // 0 = does not expire per RFC 7591
		ClientName:              clientName,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
	}
	if clientSecret != "" {
		resp.ClientSecret = clientSecret
	}

	return resp, nil
}
