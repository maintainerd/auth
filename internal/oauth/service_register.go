package oauth

import (
	"context"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	clientpkg "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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

// dcrAllowedGrantTypes is the set a dynamically registered client may ask for.
//
// The registration endpoint used to accept arbitrary grant_types. Combined with
// token_endpoint_auth_method=none that let a caller create a client that mints
// tokens with no credential, and token-exchange/CIBA/device are administrative
// capabilities that should be granted deliberately, not self-served.
var dcrAllowedGrantTypes = map[string]struct{}{
	GrantTypeAuthorizationCode: {},
	GrantTypeRefreshToken:      {},
	GrantTypeClientCredentials: {},
}

// OAuthRegisterService handles Dynamic Client Registration (RFC 7591).
type OAuthRegisterService interface {
	// Register creates a new OAuth client from the supplied metadata and returns
	// the client_id (and client_secret for confidential clients).
	//
	// tenantID is the CALLER's tenant, taken from the authenticated registration
	// request. It used to be the system tenant unconditionally, so every
	// dynamically registered client landed in the tenant that owns the platform's
	// own seeded clients.
	Register(ctx context.Context, req OAuthClientRegistrationRequestDTO, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError)

	// Read returns the registered metadata for a client (RFC 7592 §2.1), scoped to
	// the caller's tenant. It never returns the client secret: the secret is shown
	// exactly once, at registration.
	Read(ctx context.Context, clientIdentifier string, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError)
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
func (s *oauthRegisterService) Register(ctx context.Context, req OAuthClientRegistrationRequestDTO, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_register.register")
	defer span.End()

	// The tenant is the authenticated caller's, never a default. Falling back to
	// the system tenant would put a caller-created client alongside the platform's
	// own seeded clients.
	if tenantID <= 0 {
		span.SetStatus(codes.Error, "no caller tenant")
		return nil, apperror.NewOAuthInvalidRequest("registration requires an authenticated tenant context")
	}

	// Require at least one redirect URI per RFC 7591 §2.
	if len(req.RedirectURIs) == 0 {
		return nil, apperror.NewOAuthInvalidRequest("at least one redirect_uri is required")
	}

	// Default grant type is authorization_code when not specified.
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{GrantTypeAuthorizationCode}
	}
	for _, g := range grantTypes {
		if _, ok := dcrAllowedGrantTypes[g]; !ok {
			return nil, apperror.NewOAuthInvalidRequest("grant_type is not available through dynamic registration: " + g)
		}
	}

	// Default response type mirrors the grant type.
	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	// Reject dangerous redirect schemes BEFORE anything is written. Registration
	// is what makes a redirect target trusted for the rest of the flow, so the
	// scheme check has to happen here and not only at /authorize.
	redirectURIs := make([]string, 0, len(req.RedirectURIs))
	for _, uri := range req.RedirectURIs {
		safeURI := security.SanitizeInput(uri)
		if safeURI == "" {
			continue
		}
		if err := security.ValidateRedirectURI(safeURI); err != nil {
			return nil, apperror.NewOAuthInvalidRequest("redirect_uri uses a forbidden scheme: " + safeURI)
		}
		redirectURIs = append(redirectURIs, safeURI)
	}
	if len(redirectURIs) == 0 {
		return nil, apperror.NewOAuthInvalidRequest("at least one redirect_uri is required")
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

	tokenEndpointAuthMethod := req.TokenEndpointAuthMethod
	if tokenEndpointAuthMethod == "" {
		tokenEndpointAuthMethod = TokenAuthMethodSecretBasic
	}

	// client_type must be one of the four values chk_clients_client_type admits.
	// The old code wrote "public"/"confidential", which are not in that set, so
	// every registration violated the constraint and 500'd — the endpoint had
	// never actually worked.
	clientType := shared.ClientTypeTraditional
	if tokenEndpointAuthMethod == TokenAuthMethodNone {
		clientType = shared.ClientTypeSPA
	}

	// RFC 7591 `scope` becomes the client's allowed_scopes. It is load-bearing for
	// a machine credential: an empty allowlist means "the baseline OIDC scopes"
	// (see validateClientAllowedScopes), and ValidateClientOAuthMatrix refuses a
	// client_credentials client that declares none at all.
	allowedScopes := []string(parseScopeFields(security.SanitizeInput(req.Scope)))

	client := &Client{
		TenantID:                tenantID,
		Name:                    clientName,
		DisplayName:             clientName,
		Identifier:              ptr.Ptr(rawClientID),
		Status:                  shared.StatusActive,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		ClientType:              clientType,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
		AllowedScopes:           allowedScopes,
	}

	// Confidential clients get a generated secret (hashed at rest, returned once).
	clientSecret := ""
	if tokenEndpointAuthMethod != TokenAuthMethodNone {
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
	}

	// The same client-type / auth-method / grant matrix the console write path
	// enforces. Applying it here is what stops registration creating the one
	// combination authenticateOAuthClient has to refuse at runtime: a
	// credential-less confidential or m2m client.
	if err := clientpkg.ValidateClientOAuthMatrix(
		clientType,
		tokenEndpointAuthMethod,
		grantTypes,
		allowedScopes,
		clientSecret != "",
		false,
	); err != nil {
		span.SetStatus(codes.Error, "client matrix rejected")
		return nil, apperror.NewOAuthInvalidRequest(err.Error())
	}

	createdClient, err := s.clientRepo.Create(client)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client creation failed")
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}

	// Register redirect URIs.
	for _, safeURI := range redirectURIs {
		clientURI := &ClientURI{
			TenantID: tenantID,
			ClientID: createdClient.ClientID,
			URI:      safeURI,
			Type:     shared.ClientURITypeRedirect,
		}
		if _, uerr := s.clientURIRepo.Create(clientURI); uerr != nil {
			span.RecordError(uerr)
		}
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
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
		RedirectURIs:            redirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
	}
	if clientSecret != "" {
		resp.ClientSecret = clientSecret
	}

	return resp, nil
}

// Read implements OAuthRegisterService (RFC 7592 §2.1).
func (s *oauthRegisterService) Read(ctx context.Context, clientIdentifier string, tenantID int64) (*OAuthClientRegistrationResponseDTO, *apperror.OAuthError) {
	_, span := otel.Tracer("service").Start(ctx, "oauth_register.read")
	defer span.End()

	if tenantID <= 0 {
		return nil, apperror.NewOAuthInvalidRequest("registration requires an authenticated tenant context")
	}

	client, err := s.clientRepo.FindByIdentifier(clientIdentifier)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewOAuthServerError("an unexpected error occurred")
	}
	// A cross-tenant hit is reported as "not found", not "forbidden": the
	// difference would confirm that a client_id exists in another tenant.
	if client == nil || client.TenantID != tenantID {
		return nil, apperror.NewOAuthInvalidRequest("client not found")
	}

	redirectURIs := []string{}
	if client.ClientURIs != nil {
		for _, uri := range *client.ClientURIs {
			if uri.Type == shared.ClientURITypeRedirect {
				redirectURIs = append(redirectURIs, uri.URI)
			}
		}
	}

	span.SetStatus(codes.Ok, "")
	// No ClientSecret field is populated: RFC 7592 lets the AS return one, but the
	// secret is only ever stored hashed, and returning the decryptable copy would
	// turn a read endpoint into a secret-exfiltration endpoint.
	return &OAuthClientRegistrationResponseDTO{
		ClientID:                ptrOrEmpty(client.Identifier),
		ClientIDIssuedAt:        client.CreatedAt.Unix(),
		ClientSecretExpiresAt:   0,
		ClientName:              client.DisplayName,
		RedirectURIs:            redirectURIs,
		GrantTypes:              []string(client.GrantTypes),
		ResponseTypes:           []string(client.ResponseTypes),
		TokenEndpointAuthMethod: client.TokenEndpointAuthMethod,
	}, nil
}
