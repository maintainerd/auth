package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	clientmodel "github.com/maintainerd/maintainerd-auth/internal/client"
	iammodel "github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/setup/seeder"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// This file is the orchestrator-provisioning half of setup: everything an
// orchestrator (core) needs configured before the setup window closes for good.
//
// Every RPC here is get-or-create, never create-or-fail. Provisioning is driven
// by a machine over a network, so a lost response is routine; a Create* that
// answered a retry with a conflict would leave core unable to tell "I already
// did this" from "someone else took the name", and its only recovery would be to
// tear the instance down. Declarative Ensure* makes an interrupted provision
// resumable by replaying it, which is also why none of this needs a server-side
// replay ledger.

// DefaultControlPolicyName is the policy an orchestrator's service principal
// carries. Built during setup from an explicit request, not seeded.
const DefaultControlPolicyName = "auth-control"

// EnsureControlClientRequestDTO registers the machine client an orchestrator
// authenticates as after setup closes.
type EnsureControlClientRequestDTO struct {
	Name        string
	DisplayName string
	ServiceName string
	JWKS        string
	JWKSUri     string
	Audience    string
}

type EnsureControlClientResponseDTO struct {
	ClientUUID              string
	ClientID                string
	TokenEndpointAuthMethod string
	ServiceUUID             string
	AlreadyExisted          bool
}

// EnsureControlClient creates (or returns) the orchestrator's machine client.
//
// Authentication is private_key_jwt (RFC 7523) and nothing else is offered. The
// caller registers a PUBLIC key; this service stores no credential for it, so a
// dump of this database yields nothing that can impersonate the orchestrator —
// unlike a client secret, which has to exist in plaintext at least once, travel
// over this RPC, and then live in the orchestrator's configuration forever.
//
// It is also what makes this call idempotent for free: with no
// returned-exactly-once secret, replaying the request returns the same client
// instead of stranding a live credential the caller never received.
func (s *setupService) EnsureControlClient(ctx context.Context, req EnsureControlClientRequestDTO) (*EnsureControlClientResponseDTO, error) {
	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}
	if err := s.requireProvisioningDeps(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperror.NewValidation("control client name is required")
	}
	serviceName := strings.TrimSpace(req.ServiceName)
	if serviceName == "" {
		return nil, apperror.NewValidation("service_name is required: an unbound client authenticates but is then denied every permission-gated method, because its token names no principal for the control policy to attach to")
	}
	jwks := strings.TrimSpace(req.JWKS)
	jwksURI := strings.TrimSpace(req.JWKSUri)
	if jwks == "" && jwksURI == "" {
		// No key means no way to ever authenticate. Registering the client anyway
		// would produce a principal that looks provisioned and cannot be used, and
		// the orchestrator would only discover it after setup had closed.
		return nil, apperror.NewValidation("either jwks or jwks_uri is required: this client authenticates with private_key_jwt and is issued no secret")
	}
	if jwks != "" && !json.Valid([]byte(jwks)) {
		return nil, apperror.NewValidation("jwks is not valid JSON")
	}
	if jwks != "" && jwksURI != "" {
		// Two sources of truth for the verification key is one too many: they can
		// disagree, and which one wins decides who can authenticate.
		return nil, apperror.NewValidation("supply either jwks or jwks_uri, not both")
	}
	if jwksURI != "" && !strings.HasPrefix(strings.ToLower(jwksURI), "https://") {
		// The key set decides who may act as the orchestrator; fetched over plain
		// HTTP it is whatever the network says it is.
		return nil, apperror.NewValidation("jwks_uri must be https")
	}

	sysTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return nil, err
	}
	if sysTenant == nil {
		return nil, apperror.NewValidation("tenant must be created first")
	}

	var out EnsureControlClientResponseDTO
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txServiceRepo := s.serviceRepo.WithTx(tx)

		// The service must already exist. Creating it here would let a client be
		// bound to a principal that carries no policy — authentication would work
		// and authorization would not, which is the harder failure to diagnose.
		boundSvc, err := txServiceRepo.FindByNameAndTenantID(serviceName, sysTenant.TenantID)
		if err != nil {
			return err
		}
		if boundSvc == nil {
			return apperror.NewValidation(fmt.Sprintf("service %q is not registered; call RegisterControlService first so the control policy exists to bind to", serviceName))
		}

		existing, err := txClientRepo.FindByNameAndTenantID(name, sysTenant.TenantID)
		if err != nil {
			return err
		}
		if existing != nil {
			out = EnsureControlClientResponseDTO{
				ClientUUID:              existing.ClientUUID.String(),
				ClientID:                derefStr(existing.Identifier),
				TokenEndpointAuthMethod: existing.TokenEndpointAuthMethod,
				ServiceUUID:             boundSvc.ServiceUUID.String(),
				AlreadyExisted:          true,
			}
			return nil
		}

		identifier, err := crypto.GenerateIdentifier(32)
		if err != nil {
			return apperror.NewInternal("generate control client identifier", err)
		}

		record := &clientmodel.Client{
			ClientUUID:  uuid.New(),
			TenantID:    sysTenant.TenantID,
			Name:        name,
			DisplayName: displayNameOr(req.DisplayName, name),
			ClientType:  shared.ClientTypeM2M,
			Identifier:  ptr.Ptr(identifier),
			// The binding that puts `svc` in this client's tokens; the interceptor
			// resolves the control policy by that claim.
			ServiceID: &boundSvc.ServiceID,
			// No SecretHash: private_key_jwt authenticates with a signature over a
			// key this service never holds.
			SecretHash:              nil,
			TokenEndpointAuthMethod: clientmodel.TokenAuthMethodPrivateKeyJWT,
			GrantTypes:              pq.StringArray{clientmodel.GrantTypeClientCredentials},
			Status:                  shared.StatusActive,
			IsSystem:                true,
			RequireConsent:          boolPtr(false),
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		}
		if jwks != "" {
			record.JWKS = datatypes.JSON([]byte(jwks))
		}
		if jwksURI != "" {
			record.JWKSUri = ptr.Ptr(jwksURI)
		}
		if aud := strings.TrimSpace(req.Audience); aud != "" {
			record.Config = datatypes.JSON([]byte(fmt.Sprintf(`{"audience":%q}`, aud)))
		}

		created, err := txClientRepo.Create(record)
		if err != nil {
			return err
		}
		out = EnsureControlClientResponseDTO{
			ClientUUID:              created.ClientUUID.String(),
			ClientID:                identifier,
			TokenEndpointAuthMethod: created.TokenEndpointAuthMethod,
			ServiceUUID:             boundSvc.ServiceUUID.String(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// EnsureResourceAPIPermissionDTO is one permission an orchestrator's API defines.
type EnsureResourceAPIPermissionDTO struct {
	Name        string
	Description string
}

type EnsureResourceAPIRequestDTO struct {
	ServiceName        string
	ServiceDisplayName string
	Name               string
	DisplayName        string
	Identifier         string
	Permissions        []EnsureResourceAPIPermissionDTO
}

type EnsureResourceAPIResponseDTO struct {
	ServiceUUID     string
	APIUUID         string
	Identifier      string
	PermissionNames []string
	AlreadyExisted  bool
}

// EnsureResourceAPI registers an API the orchestrator owns, plus the permissions
// it defines.
//
// The orchestrator is not only a caller of this service, it is also a resource
// server: its own operators are authorized by permissions that live here. Its
// API has to exist before a token can be minted with that audience, and its
// permissions have to exist before a role can carry them — which is why this
// runs inside the setup window rather than after it.
func (s *setupService) EnsureResourceAPI(ctx context.Context, req EnsureResourceAPIRequestDTO) (*EnsureResourceAPIResponseDTO, error) {
	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}
	if err := s.requireProvisioningDeps(); err != nil {
		return nil, err
	}
	serviceName := strings.TrimSpace(req.ServiceName)
	apiName := strings.TrimSpace(req.Name)
	identifier := strings.TrimSpace(req.Identifier)
	if serviceName == "" || apiName == "" || identifier == "" {
		return nil, apperror.NewValidation("service_name, name and identifier are required")
	}

	sysTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return nil, err
	}
	if sysTenant == nil {
		return nil, apperror.NewValidation("tenant must be created first")
	}

	var out EnsureResourceAPIResponseDTO
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txServiceRepo := s.serviceRepo.WithTx(tx)
		txAPIRepo := s.apiRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)

		service, err := txServiceRepo.FindByNameAndTenantID(serviceName, sysTenant.TenantID)
		if err != nil {
			return err
		}
		if service == nil {
			service, err = txServiceRepo.Create(&iammodel.Service{
				ServiceUUID: uuid.New(),
				TenantID:    sysTenant.TenantID,
				Name:        serviceName,
				DisplayName: displayNameOr(req.ServiceDisplayName, serviceName),
				Version:     "v1",
				Status:      shared.StatusActive,
				IsSystem:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
			if err != nil {
				return err
			}
		}
		out.ServiceUUID = service.ServiceUUID.String()

		api, err := txAPIRepo.FindByIdentifier(identifier, sysTenant.TenantID)
		if err != nil {
			return err
		}
		if api != nil {
			if api.ServiceID != service.ServiceID {
				// The identifier is the audience in issued tokens. Letting a second
				// service claim one already in use would mint tokens accepted by an
				// API the caller did not mean to reach.
				return apperror.NewConflict(fmt.Sprintf("api identifier %q is already registered to another service", identifier))
			}
			out.AlreadyExisted = true
		} else {
			api, err = txAPIRepo.Create(&iammodel.API{
				APIUUID:     uuid.New(),
				TenantID:    sysTenant.TenantID,
				ServiceID:   service.ServiceID,
				Name:        apiName,
				DisplayName: displayNameOr(req.DisplayName, apiName),
				Identifier:  identifier,
				Status:      shared.StatusActive,
				IsSystem:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
			if err != nil {
				return err
			}
		}
		out.APIUUID = api.APIUUID.String()
		out.Identifier = identifier

		for _, p := range req.Permissions {
			permName := strings.TrimSpace(p.Name)
			if permName == "" {
				continue
			}
			existing, err := txPermissionRepo.FindByName(permName, sysTenant.TenantID)
			if err != nil {
				return err
			}
			if existing == nil {
				if _, err := txPermissionRepo.Create(&iammodel.Permission{
					PermissionUUID: uuid.New(),
					TenantID:       sysTenant.TenantID,
					APIID:          api.APIID,
					Name:           permName,
					Description:    p.Description,
					Status:         shared.StatusActive,
					IsSystem:       true,
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}); err != nil {
					return err
				}
			} else if existing.APIID != api.APIID {
				return apperror.NewConflict(fmt.Sprintf("permission %q already belongs to a different api", permName))
			}
			out.PermissionNames = append(out.PermissionNames, permName)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type EnsureRoleRequestDTO struct {
	Name             string
	Description      string
	PermissionNames  []string
	AssignToUserUUID string
}

type EnsureRoleResponseDTO struct {
	RoleUUID        string
	Name            string
	PermissionNames []string
	Assigned        bool
	AlreadyExisted  bool
}

// EnsureRole creates a role carrying the given permissions and optionally grants
// it to a user — how an orchestrator gives its first administrator access to
// itself.
//
// The permissions must already exist: they are looked up, never created. A role
// that silently invented its own permissions would let this one call define both
// the grant and the thing being granted, so a typo would produce a role whose
// permission no guard ever checks — access that looks configured and is not.
func (s *setupService) EnsureRole(ctx context.Context, req EnsureRoleRequestDTO) (*EnsureRoleResponseDTO, error) {
	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}
	if err := s.requireProvisioningDeps(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperror.NewValidation("role name is required")
	}

	sysTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return nil, err
	}
	if sysTenant == nil {
		return nil, apperror.NewValidation("tenant must be created first")
	}

	var out EnsureRoleResponseDTO
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txRoleRepo := s.roleRepo.WithTx(tx)
		txPermissionRepo := s.permissionRepo.WithTx(tx)
		txRolePermissionRepo := s.rolePermissionRepo.WithTx(tx)
		txUserRepo := s.userRepo.WithTx(tx)
		txUserRoleRepo := s.userRoleRepo.WithTx(tx)

		role, err := txRoleRepo.FindByNameAndTenantID(name, sysTenant.TenantID)
		if err != nil {
			return err
		}
		if role != nil {
			out.AlreadyExisted = true
		} else {
			role, err = txRoleRepo.Create(&iammodel.Role{
				RoleUUID:    uuid.New(),
				TenantID:    sysTenant.TenantID,
				Name:        name,
				Description: req.Description,
				IsSystem:    true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
			if err != nil {
				return err
			}
		}
		out.RoleUUID = role.RoleUUID.String()
		out.Name = role.Name

		for _, permName := range req.PermissionNames {
			permName = strings.TrimSpace(permName)
			if permName == "" {
				continue
			}
			permission, err := txPermissionRepo.FindByName(permName, sysTenant.TenantID)
			if err != nil {
				return err
			}
			if permission == nil {
				return apperror.NewValidation(fmt.Sprintf("permission %q does not exist; register it with EnsureResourceAPI first", permName))
			}
			existing, err := txRolePermissionRepo.FindByRoleAndPermission(role.RoleID, permission.PermissionID)
			if err != nil {
				return err
			}
			if existing == nil {
				if _, err := txRolePermissionRepo.Assign(&iammodel.RolePermission{
					RolePermissionUUID: uuid.New(),
					RoleID:             role.RoleID,
					PermissionID:       permission.PermissionID,
					CreatedAt:          time.Now(),
				}); err != nil {
					return err
				}
			}
			out.PermissionNames = append(out.PermissionNames, permName)
		}

		if raw := strings.TrimSpace(req.AssignToUserUUID); raw != "" {
			userUUID, err := uuid.Parse(raw)
			if err != nil {
				return apperror.NewValidation("assign_to_user_uuid is not a valid UUID")
			}
			target, err := txUserRepo.FindByUUID(userUUID)
			if err != nil {
				return err
			}
			if target == nil {
				return apperror.NewNotFoundWithReason("user to assign the role to was not found")
			}
			if target.TenantID != sysTenant.TenantID {
				// Setup provisions THIS instance's system tenant. Reaching across a
				// tenant boundary here would grant a role to someone the orchestrator
				// does not administer.
				return apperror.NewForbidden("the user to assign the role to does not belong to the system tenant")
			}
			link, err := txUserRoleRepo.FindByUserIDAndRoleID(target.UserID, role.RoleID)
			if err != nil {
				return err
			}
			if link == nil {
				if _, err := txUserRoleRepo.Create(&UserRole{
					UserRoleUUID: uuid.New(),
					UserID:       target.UserID,
					RoleID:       role.RoleID,
					CreatedAt:    time.Now(),
				}); err != nil {
					return err
				}
			}
			out.Assigned = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type EnsureConsoleClientRequestDTO struct {
	Name                   string
	DisplayName            string
	Domain                 string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
}

type EnsureConsoleClientResponseDTO struct {
	ClientUUID             string
	ClientID               string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	AlreadyExisted         bool
}

// EnsureConsoleClient registers the browser application an orchestrator's
// operators sign in to.
//
// It is a PUBLIC client: authorization_code with PKCE (S256) and no credential.
// A single-page app cannot keep a secret — issuing one would ship it in the
// bundle to every visitor and produce a client that is confidential in the
// database and public in reality.
func (s *setupService) EnsureConsoleClient(ctx context.Context, req EnsureConsoleClientRequestDTO) (*EnsureConsoleClientResponseDTO, error) {
	if err := s.ensureSetupOpen(); err != nil {
		return nil, err
	}
	if err := s.requireProvisioningDeps(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperror.NewValidation("console client name is required")
	}
	redirects, err := validateHTTPSURIs(req.RedirectURIs, "redirect_uris")
	if err != nil {
		return nil, err
	}
	if len(redirects) == 0 {
		return nil, apperror.NewValidation("at least one redirect_uri is required")
	}
	logouts, err := validateHTTPSURIs(req.PostLogoutRedirectURIs, "post_logout_redirect_uris")
	if err != nil {
		return nil, err
	}

	sysTenant, err := s.tenantRepo.FindSystem()
	if err != nil {
		return nil, err
	}
	if sysTenant == nil {
		return nil, apperror.NewValidation("tenant must be created first")
	}

	var out EnsureConsoleClientResponseDTO
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txClientRepo := s.clientRepo.WithTx(tx)
		txClientURIRepo := s.clientURIRepo.WithTx(tx)

		record, err := txClientRepo.FindByNameAndTenantID(name, sysTenant.TenantID)
		if err != nil {
			return err
		}
		if record != nil {
			out.AlreadyExisted = true
		} else {
			identifier, err := crypto.GenerateIdentifier(32)
			if err != nil {
				return apperror.NewInternal("generate console client identifier", err)
			}
			record, err = txClientRepo.Create(&clientmodel.Client{
				ClientUUID:  uuid.New(),
				TenantID:    sysTenant.TenantID,
				Name:        name,
				DisplayName: displayNameOr(req.DisplayName, name),
				ClientType:  shared.ClientTypeSPA,
				Domain:      ptr.Ptr(strings.TrimSpace(req.Domain)),
				Identifier:  ptr.Ptr(identifier),
				SecretHash:  nil,
				Config: datatypes.JSON([]byte(`{
					"grant_types": ["authorization_code", "refresh_token"],
					"response_type": "code",
					"pkce": true
				}`)),
				TokenEndpointAuthMethod: clientmodel.TokenAuthMethodNone,
				GrantTypes:              pq.StringArray{clientmodel.GrantTypeAuthorizationCode, clientmodel.GrantTypeRefreshToken},
				ResponseTypes:           pq.StringArray{clientmodel.ResponseTypeCode},
				Status:                  shared.StatusActive,
				IsSystem:                true,
				RequireConsent:          boolPtr(false),
				CreatedAt:               time.Now(),
				UpdatedAt:               time.Now(),
			})
			if err != nil {
				return err
			}
		}
		out.ClientUUID = record.ClientUUID.String()
		out.ClientID = derefStr(record.Identifier)

		for _, spec := range []struct {
			uris    []string
			uriType string
			into    *[]string
		}{
			{redirects, shared.ClientURITypeRedirect, &out.RedirectURIs},
			{logouts, shared.ClientURITypeLogout, &out.PostLogoutRedirectURIs},
		} {
			for _, uri := range spec.uris {
				existing, err := txClientURIRepo.FindByURIAndType(uri, spec.uriType, record.ClientID, sysTenant.TenantID)
				if err != nil {
					return err
				}
				if existing == nil {
					if _, err := txClientURIRepo.Create(&clientmodel.ClientURI{
						ClientURIUUID: uuid.New(),
						ClientID:      record.ClientID,
						URI:           uri,
						Type:          spec.uriType,
						CreatedAt:     time.Now(),
						UpdatedAt:     time.Now(),
					}); err != nil {
						return err
					}
				}
				*spec.into = append(*spec.into, uri)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// validateHTTPSURIs rejects a redirect target that is not an absolute https URL.
//
// A redirect URI is where this service sends an authorization code, so a
// relative or non-https value is an open redirect with a credential attached.
// http://localhost is allowed because a native or local development client has
// no other option and the loopback interface is not on the network.
func validateHTTPSURIs(uris []string, field string) ([]string, error) {
	out := make([]string, 0, len(uris))
	for _, raw := range uris {
		uri := strings.TrimSpace(raw)
		if uri == "" {
			continue
		}
		lower := strings.ToLower(uri)
		isLoopback := strings.HasPrefix(lower, "http://localhost") || strings.HasPrefix(lower, "http://127.0.0.1")
		if !strings.HasPrefix(lower, "https://") && !isLoopback {
			return nil, apperror.NewValidation(fmt.Sprintf("%s must be absolute https URLs (http is permitted only for loopback): %q", field, uri))
		}
		if strings.Contains(uri, "*") {
			// A wildcard redirect matches hosts the operator never enumerated, which
			// is how authorization codes end up delivered to an attacker's origin.
			return nil, apperror.NewValidation(fmt.Sprintf("%s must not contain wildcards: %q", field, uri))
		}
		out = append(out, uri)
	}
	return out, nil
}

func displayNameOr(displayName string, fallback string) string {
	if d := strings.TrimSpace(displayName); d != "" {
		return d
	}
	return fallback
}

// requireProvisioningDeps fails closed when the orchestrator-provisioning repos
// were not wired, rather than half-completing a provision.
func (s *setupService) requireProvisioningDeps() error {
	if s.db == nil || s.serviceRepo == nil || s.apiRepo == nil || s.permissionRepo == nil ||
		s.rolePermissionRepo == nil || s.clientURIRepo == nil {
		return apperror.NewInternal("orchestrator provisioning dependencies are not configured", nil)
	}
	return nil
}

func derefStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func boolPtr(v bool) *bool { return &v }

// ensureControlPolicy creates or returns the orchestrator control policy for
// this tenant, built from an explicit action list.
//
// An existing policy is returned UNCHANGED rather than widened to match the
// request. Setup is reachable with only the bootstrap credential, so letting a
// later call rewrite the policy would make "re-run registration" a way to
// escalate an already-registered principal without going through the reviewed
// policy-management path.
func (s *setupService) ensureControlPolicy(repo PolicyRepository, tenantID int64, allowedActions []string) (*Policy, error) {
	existing, err := repo.FindByNameAndVersion(DefaultControlPolicyName, "v1", tenantID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	actions := normalizeControlActions(allowedActions)
	document := iammodel.PolicyDocument{
		Version: "v1",
		Statement: []iammodel.PolicyStatement{
			{Effect: "allow", Action: actions, Resource: []string{"*"}},
		},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, apperror.NewInternal("encode control policy document", err)
	}

	return repo.Create(&Policy{
		PolicyUUID:  uuid.New(),
		TenantID:    tenantID,
		Name:        DefaultControlPolicyName,
		Description: ptr.Ptr("Service-to-service control policy for orchestrator-managed auth administration."),
		Document:    datatypes.JSON(raw),
		Version:     "v1",
		Status:      shared.StatusActive,
		IsSystem:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})
}

// normalizeControlActions trims and de-duplicates the requested actions, falling
// back to the documented default set when none are supplied.
//
// A bare "*" is refused. An orchestrator asking for every action on every
// resource is asking for a grant that cannot be reviewed and that silently
// absorbs every permission family added later — exactly the failure the seeded
// policy had.
func normalizeControlActions(requested []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(requested))
	for _, a := range requested {
		a = strings.TrimSpace(a)
		if a == "" || a == "*" || a == "*:*" {
			continue
		}
		if _, dup := seen[a]; dup {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	if len(out) == 0 {
		return append([]string(nil), seeder.DefaultControlActions...)
	}
	return out
}
