package client

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type ClientGRPCHandler struct {
	authv1.UnimplementedClientServiceServer
	tenantResolver TenantResolver
	clientService  ClientService
}

func NewClientGRPCHandler(tenantResolver TenantResolver, clientService ClientService) *ClientGRPCHandler {
	return &ClientGRPCHandler{tenantResolver: tenantResolver, clientService: clientService}
}

func (h *ClientGRPCHandler) ListClients(ctx context.Context, req *authv1.ListClientsRequest) (*authv1.ListClientsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	dto := ClientFilterDTO{
		Name:                 optionalStr(req.GetName()),
		DisplayName:          optionalStr(req.GetDisplayName()),
		ClientType:           req.GetClientType(),
		IdentityProviderUUID: optionalStr(req.GetIdentityProviderId()),
		Status:               req.GetStatus(),
		IsDefault:            req.IsDefault,
		IsSystem:             req.IsSystem,
		PaginationRequestDTO: grpcPagination(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.clientService.Get(ctx, ClientServiceGetFilter{
		TenantID:             tenant.TenantID,
		Name:                 dto.Name,
		DisplayName:          dto.DisplayName,
		ClientType:           dto.ClientType,
		IdentityProviderUUID: dto.IdentityProviderUUID,
		Status:               dto.Status,
		IsDefault:            dto.IsDefault,
		IsSystem:             dto.IsSystem,
		Page:                 dto.Page,
		Limit:                dto.Limit,
		SortBy:               dto.SortBy,
		SortOrder:            dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.Client, len(result.Data))
	for i := range result.Data {
		rows[i] = toClientProto(&result.Data[i])
	}
	return &authv1.ListClientsResponse{Clients: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *ClientGRPCHandler) GetClient(ctx context.Context, req *authv1.GetClientRequest) (*authv1.GetClientResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.GetByUUID(ctx, clientUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetClientResponse{Client: toClientProto(result)}, nil
}

// GetClientSecret exists only because the generated ClientServiceServer
// interface requires it; the HTTP twin has been removed outright. Secrets are
// bcrypt hashed at rest, so there is nothing to return.
//
// It answers UNIMPLEMENTED rather than OK-with-a-message: the old response was a
// success, so a generated client would hand its caller an empty ClientSecret and
// no error, which reads as "this client has no secret".
func (h *ClientGRPCHandler) GetClientSecret(ctx context.Context, req *authv1.GetClientSecretRequest) (*authv1.GetClientSecretResponse, error) {
	return nil, status.Error(codes.Unimplemented,
		"client secrets cannot be retrieved after creation; use RotateClientSecret to issue a new one")
}

func (h *ClientGRPCHandler) RotateClientSecret(ctx context.Context, req *authv1.RotateClientSecretRequest) (*authv1.RotateClientSecretResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	newSecret, err := h.clientService.RotateSecret(ctx, clientUUID, tenant.TenantID, actorUUID, int(req.GetGracePeriodHours()))
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RotateClientSecretResponse{ClientSecret: newSecret}, nil
}

func (h *ClientGRPCHandler) GetClientConfig(ctx context.Context, req *authv1.GetClientConfigRequest) (*authv1.GetClientConfigResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	config, err := h.clientService.GetConfigByUUID(ctx, clientUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	var configMap map[string]any
	if len(config) > 0 {
		if err := json.Unmarshal(config, &configMap); err != nil {
			configMap = nil
		}
	}
	configProto, _ := structpb.NewStruct(configMap)
	if configProto == nil {
		configProto = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return &authv1.GetClientConfigResponse{Config: configProto}, nil
}

// CreateClient is replay-guarded: the response carries a client secret that
// exists in plaintext exactly once, so an un-de-duplicated retry does not just
// create a spare client — it strands a live credential the caller never sees
// again and cannot rotate, because it does not know the client is there.
func (h *ClientGRPCHandler) CreateClient(ctx context.Context, req *authv1.CreateClientRequest) (*authv1.CreateClientResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	actor, err := h.clientMutationActor(ctx, tenant.TenantID, req.GetClientType(), req.GetServiceId())
	if err != nil {
		return nil, err
	}
	config := structProtoToMap(req.GetConfig())
	configJSON, err := mapToJSON(config)
	if err != nil {
		return nil, err
	}
	allowReg := true
	if req.AllowRegistration != nil {
		allowReg = *req.AllowRegistration
	}

	// An omitted config is legitimate here — the column default is '{}'. Normalize
	// before validating so the Required rule catches genuine omissions only.
	if len(configJSON) == 0 {
		configJSON = []byte("{}")
	}

	// Validate with the same DTO the REST handler uses: the two transports must
	// not disagree about what a valid client is.
	if err := (ClientCreateRequestDTO{
		Name:                 req.GetName(),
		DisplayName:          req.GetDisplayName(),
		ClientType:           req.GetClientType(),
		Domain:               req.GetDomain(),
		Config:               configJSON,
		Status:               req.GetStatus(),
		IdentityProviderUUID: req.GetIdentityProviderId(),
		ServiceUUID:          req.ServiceId,
	}).Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}

	result, err := h.clientService.Create(ctx, tenant.TenantID, req.GetName(), req.GetDisplayName(), req.GetClientType(), req.GetDomain(), configJSON, req.GetStatus(), req.GetIdentityProviderId(), nil, allowReg, nil, nil, nil, nil, actor, req.ServiceId)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateClientResponse{
		Client: toClientProto(result.Client),
		Credentials: &authv1.ClientCredentials{
			ClientId:      result.Client.ClientUUID.String(),
			OauthClientId: result.ClientIdentifier,
			ClientSecret:  result.PlaintextSecret,
		},
	}, nil
}

func (h *ClientGRPCHandler) UpdateClient(ctx context.Context, req *authv1.UpdateClientRequest) (*authv1.UpdateClientResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actor, err := h.clientMutationActor(ctx, tenant.TenantID, req.GetClientType(), req.GetServiceId())
	if err != nil {
		return nil, err
	}
	config := structProtoToMap(req.GetConfig())
	configJSON, err := mapToJSON(config)
	if err != nil {
		return nil, err
	}
	// An omitted config means "leave it unchanged". The DTO requires the field, so
	// validate against a placeholder and pass nil to the service — mapToJSON
	// returns nil for an empty map, and config is a NOT NULL jsonb column.
	configOmitted := len(config) == 0
	configForValidation := configJSON
	if configOmitted {
		configForValidation = []byte("{}")
	}

	// Same DTO as the REST path — see CreateClient above.
	if err := (ClientUpdateRequestDTO{
		Name:        req.GetName(),
		DisplayName: req.GetDisplayName(),
		ClientType:  req.GetClientType(),
		Domain:      req.GetDomain(),
		Config:      configForValidation,
		Status:      req.GetStatus(),
		ServiceUUID: req.ServiceId,
	}).Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	if configOmitted {
		configJSON = nil
	}

	result, err := h.clientService.Update(ctx, clientUUID, tenant.TenantID, req.GetName(), req.GetDisplayName(), req.GetClientType(), req.GetDomain(), configJSON, req.GetStatus(), nil, req.AllowRegistration, nil, nil, nil, nil, nil, actor, nil, req.ServiceId)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateClientResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) SetClientStatus(ctx context.Context, req *authv1.SetClientStatusRequest) (*authv1.SetClientStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.SetStatusByUUID(ctx, clientUUID, tenant.TenantID, req.GetStatus(), actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetClientStatusResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) DeleteClient(ctx context.Context, req *authv1.DeleteClientRequest) (*authv1.DeleteClientResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.DeleteByUUID(ctx, clientUUID, tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteClientResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) ListClientURIs(ctx context.Context, req *authv1.ListClientURIsRequest) (*authv1.ListClientURIsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.GetByUUID(ctx, clientUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	uris := make([]*authv1.ClientURI, 0)
	if result.ClientURIs != nil {
		for _, u := range *result.ClientURIs {
			uris = append(uris, toClientURIProto(&u))
		}
	}
	return &authv1.ListClientURIsResponse{Uris: uris}, nil
}

// CreateClientURI is replay-guarded for the same reason as CreateClient: the
// unique index on (client_id, type, uri) stops a duplicate row, but it answers a
// retry with a conflict that core cannot distinguish from "another operator
// registered this redirect URI".
func (h *ClientGRPCHandler) CreateClientURI(ctx context.Context, req *authv1.CreateClientURIRequest) (*authv1.CreateClientURIResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.CreateURI(ctx, clientUUID, tenant.TenantID, req.GetUri(), req.GetType(), actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	if result.ClientURIs != nil && len(*result.ClientURIs) > 0 {
		return &authv1.CreateClientURIResponse{Uri: toClientURIProto(&(*result.ClientURIs)[0])}, nil
	}
	return &authv1.CreateClientURIResponse{}, nil
}

func (h *ClientGRPCHandler) UpdateClientURI(ctx context.Context, req *authv1.UpdateClientURIRequest) (*authv1.UpdateClientURIResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	uriUUID, err := parseUUID(req.GetClientUriId(), "Client URI UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.UpdateURI(ctx, clientUUID, tenant.TenantID, uriUUID, req.GetUri(), req.GetType(), actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	if result.ClientURIs != nil {
		for _, u := range *result.ClientURIs {
			if u.ClientURIUUID == uriUUID {
				return &authv1.UpdateClientURIResponse{Uri: toClientURIProto(&u)}, nil
			}
		}
	}
	return &authv1.UpdateClientURIResponse{}, nil
}

func (h *ClientGRPCHandler) DeleteClientURI(ctx context.Context, req *authv1.DeleteClientURIRequest) (*authv1.DeleteClientURIResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	uriUUID, err := parseUUID(req.GetClientUriId(), "Client URI UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.DeleteURI(ctx, clientUUID, tenant.TenantID, uriUUID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteClientURIResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) ListClientAPIs(ctx context.Context, req *authv1.ListClientAPIsRequest) (*authv1.ListClientAPIsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.GetClientAPIs(ctx, tenant.TenantID, clientUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	apis := make([]*authv1.ClientAPI, len(result))
	for i, api := range result {
		permProtos := make([]*authv1.ClientAPIPermission, len(api.Permissions))
		for j, perm := range api.Permissions {
			permProtos[j] = toClientAPIPermissionProto(&perm)
		}
		apis[i] = &authv1.ClientAPI{
			ClientApiId: api.ClientAPIUUID.String(),
			Api: &authv1.ClientAPIDetail{
				ApiId:       api.Api.APIUUID.String(),
				Name:        api.Api.Name,
				DisplayName: api.Api.DisplayName,
				Description: api.Api.Description,
				Status:      api.Api.Status,
				IsSystem:    api.Api.IsSystem,
				CreatedAt:   timestamppb.New(api.Api.CreatedAt),
				UpdatedAt:   timestamppb.New(api.Api.UpdatedAt),
			},
			Permissions: permProtos,
			CreatedAt:   timestamppb.New(api.CreatedAt),
		}
	}
	return &authv1.ListClientAPIsResponse{Apis: apis}, nil
}

func (h *ClientGRPCHandler) AddClientAPIs(ctx context.Context, req *authv1.AddClientAPIsRequest) (*authv1.AddClientAPIsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUIDs := make([]uuid.UUID, len(req.GetApiIds()))
	for i, a := range req.GetApiIds() {
		parsed, err := parseUUID(a, "API UUID")
		if err != nil {
			return nil, err
		}
		apiUUIDs[i] = parsed
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.clientService.AddClientAPIs(ctx, tenant.TenantID, clientUUID, apiUUIDs, actorUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddClientAPIsResponse{Message: "APIs added to client successfully"}, nil
}

func (h *ClientGRPCHandler) RemoveClientAPI(ctx context.Context, req *authv1.RemoveClientAPIRequest) (*authv1.RemoveClientAPIResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiId(), "API UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.clientService.RemoveClientAPI(ctx, tenant.TenantID, clientUUID, apiUUID, actorUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveClientAPIResponse{Message: "API removed from client successfully"}, nil
}

func (h *ClientGRPCHandler) ListClientAPIPermissions(ctx context.Context, req *authv1.ListClientAPIPermissionsRequest) (*authv1.ListClientAPIPermissionsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiId(), "API UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.GetClientAPIPermissions(ctx, tenant.TenantID, clientUUID, apiUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	perms := make([]*authv1.ClientAPIPermission, len(result))
	for i, perm := range result {
		perms[i] = toClientAPIPermissionProto(&perm)
	}
	return &authv1.ListClientAPIPermissionsResponse{Permissions: perms}, nil
}

func (h *ClientGRPCHandler) AddClientAPIPermissions(ctx context.Context, req *authv1.AddClientAPIPermissionsRequest) (*authv1.AddClientAPIPermissionsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiId(), "API UUID")
	if err != nil {
		return nil, err
	}
	permUUIDs := make([]uuid.UUID, len(req.GetPermissionIds()))
	for i, p := range req.GetPermissionIds() {
		parsed, err := parseUUID(p, "Permission UUID")
		if err != nil {
			return nil, err
		}
		permUUIDs[i] = parsed
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.clientService.AddClientAPIPermissions(ctx, tenant.TenantID, clientUUID, apiUUID, permUUIDs, actorUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddClientAPIPermissionsResponse{Message: "Permissions added to client API successfully"}, nil
}

func (h *ClientGRPCHandler) RemoveClientAPIPermission(ctx context.Context, req *authv1.RemoveClientAPIPermissionRequest) (*authv1.RemoveClientAPIPermissionResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientId(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiId(), "API UUID")
	if err != nil {
		return nil, err
	}
	permUUID, err := parseUUID(req.GetPermissionId(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := clientActorUserUUID(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.clientService.RemoveClientAPIPermission(ctx, tenant.TenantID, clientUUID, apiUUID, permUUID, actorUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveClientAPIPermissionResponse{Message: "Permission removed from client API successfully"}, nil
}

// resolveTenant resolves the target tenant from the request AND checks the caller
// is allowed to act on it.
//
// Every RPC on this service takes the target tenant from the request body, and the
// interceptor authorizes an ACTION only — it never compares the requested tenant
// against the token. Existence was therefore the only check, so any principal
// holding `client:*` in its own tenant could pass another tenant's UUID and list,
// mutate, delete, or rotate the secret of that tenant's OAuth clients. Mirrors
// iam/grpc_helpers.go resolveIAMTenant.
func (h *ClientGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	if err := h.assertCallerMayActOnTenant(ctx, result.TenantID); err != nil {
		return nil, err
	}
	return result, nil
}

// assertCallerMayActOnTenant enforces the tenant boundary: a caller may act on its
// own tenant, and a caller whose token is bound to the SYSTEM tenant may act on any
// tenant. The latter is what lets the control plane provision clients for a tenant
// remotely; it is not a blanket grant, because a tenant principal is now pinned to
// its own tenant.
func (h *ClientGRPCHandler) assertCallerMayActOnTenant(ctx context.Context, targetTenantID int64) error {
	claims := middleware.JWTClaimsFromContext(ctx)
	if claims == nil || claims.TenantID == 0 {
		// A token with no tenant cannot prove it may act anywhere.
		return status.Error(codes.PermissionDenied, "this token is not bound to a tenant")
	}
	if claims.TenantID == targetTenantID {
		return nil
	}
	callerIsSystem, err := h.callerTenantIsSystem(ctx, claims.TenantID)
	if err != nil {
		return err
	}
	if !callerIsSystem {
		return status.Error(codes.PermissionDenied, "this token may only act on its own tenant")
	}
	return nil
}

// callerTenantIsSystem reports whether the caller's own tenant is the system tenant.
// GetSystem is reached through an optional interface rather than TenantResolver so
// the resolver wired in internal/server keeps satisfying the declared interface; a
// resolver that cannot answer means cross-tenant access cannot be justified, so it
// is refused.
func (h *ClientGRPCHandler) callerTenantIsSystem(ctx context.Context, callerTenantID int64) (bool, error) {
	resolver, ok := h.tenantResolver.(interface {
		GetSystem(ctx context.Context) (*TenantServiceDataResult, error)
	})
	if !ok {
		return false, status.Error(codes.PermissionDenied, "cross-tenant access cannot be verified")
	}
	systemTenant, err := resolver.GetSystem(ctx)
	if err != nil || systemTenant == nil {
		return false, status.Error(codes.PermissionDenied, "cross-tenant access cannot be verified")
	}
	return systemTenant.TenantID == callerTenantID, nil
}

// clientActorUserUUID resolves the acting user from the VERIFIED token.
//
// Every mutating RPC here took the actor from req.GetActorUserId(), a request-body
// field, and defaulted to uuid.Nil when it was absent. That value is both the audit
// attribution AND the subject of ValidateTenantAccess in the service layer, so a
// caller could pin a secret rotation or a client deletion on an innocent tenant admin
// and borrow that admin's membership to clear the boundary check — two holes fed by
// one unauthenticated string.
//
// There is deliberately NO fallback to the body. The gRPC interceptor also admits
// service principals, which carry no user identity; a token that cannot name a user
// simply may not mutate clients. Failing closed is the only option that does not
// reopen the hole.
// clientActorUserUUID resolves the acting user for a mutating client RPC.
//
// Delegates to the one shared definition: this used to be a private copy that
// also consulted raw JWT claims, making the client surface accept a token the
// tenant surface refused.
func clientActorUserUUID(ctx context.Context) (uuid.UUID, error) {
	return middleware.GRPCActorUUID(ctx, "client changes")
}

// clientMutationActor resolves who CreateClient/UpdateClient act as.
//
// A HUMAN actor (the signed on_behalf_of user) is resolved exactly as before
// and stays the default. The single sanctioned exception is a SERVICE principal
// provisioning a machine identity — the AWS service-linked pattern: the
// orchestrator must be able to mint credentials for the services it manages
// with its own client_credentials token, which can never carry a user. That
// exception is deliberately this narrow:
//
//   - the request must name a service binding (non-empty service_id) and an m2m
//     client type, so a lone service token can never mint a user-facing login
//     client or an unbound m2m credential — anything else keeps requiring a
//     human and fails with the same error it always has;
//   - the token's own tenant must BE the target tenant. A human actor's
//     membership is checked in the service layer (ValidateTenantAccess), but a
//     service has no membership rows, so the token's tenant binding is the only
//     boundary there is. This also deliberately withholds resolveTenant's
//     system-tenant override from bare machine tokens: provisioning INTO a
//     tenant remotely still requires acting as a user of that tenant.
//
// The service layer re-checks the m2m + bound shape (defense in depth); this
// handler owns the tenant comparison because only the transport knows the
// verified token.
func (h *ClientGRPCHandler) clientMutationActor(ctx context.Context, targetTenantID int64, clientType string, requestServiceID string) (ClientActor, error) {
	principal, err := middleware.GRPCActorOrService(ctx, "client changes")
	if err != nil {
		return ClientActor{}, err
	}
	if principal.User != nil {
		return UserActor(principal.User.UserUUID), nil
	}
	if requestServiceID == "" || clientType != shared.ClientTypeM2M {
		// Not the sanctioned shape. Refuse with exactly the error the user-actor
		// path has always produced (GRPCActor cannot succeed here — the context
		// carries no user), so this fallback changes nothing for every other call.
		_, actorErr := middleware.GRPCActor(ctx, "client changes")
		return ClientActor{}, actorErr
	}
	if principal.TenantID == 0 || principal.TenantID != targetTenantID {
		return ClientActor{}, status.Error(codes.PermissionDenied,
			"a service principal may only manage clients in its own tenant")
	}
	return ServiceActor(principal.ServiceName), nil
}

func parseUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func optionalStr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func grpcPagination(req *authv1.Pagination) PaginationRequestDTO {
	if req == nil {
		return PaginationRequestDTO{Page: 1, Limit: 10}
	}
	page := int(req.GetPage())
	limit := int(req.GetLimit())
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 10
	}
	return PaginationRequestDTO{Page: page, Limit: limit, SortBy: req.GetSortBy(), SortOrder: req.GetSortOrder()}
}

func grpcPageProto(total int64, page int, limit int, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func structProtoToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func mapToJSON(m map[string]any) (datatypes.JSON, error) {
	if len(m) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(m)
	if err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation("invalid config: " + err.Error()))
	}
	return datatypes.JSON(payload), nil
}

func toClientProto(result *ClientServiceDataResult) *authv1.Client {
	if result == nil {
		return nil
	}
	var idp *authv1.ClientIdentityProvider
	if result.IdentityProvider != nil {
		idp = &authv1.ClientIdentityProvider{
			IdentityProviderId: result.IdentityProvider.IdentityProviderUUID.String(),
			Name:               result.IdentityProvider.Name,
			DisplayName:        result.IdentityProvider.DisplayName,
			Provider:           result.IdentityProvider.Provider,
			ProviderType:       result.IdentityProvider.ProviderType,
			Identifier:         result.IdentityProvider.Identifier,
			Status:             result.IdentityProvider.Status,
			IsDefault:          result.IdentityProvider.IsDefault,
			IsSystem:           result.IdentityProvider.IsSystem,
			CreatedAt:          timestamppb.New(result.IdentityProvider.CreatedAt),
			UpdatedAt:          timestamppb.New(result.IdentityProvider.UpdatedAt),
		}
	}
	var uris []*authv1.ClientURI
	if result.ClientURIs != nil {
		uris = make([]*authv1.ClientURI, len(*result.ClientURIs))
		for i, u := range *result.ClientURIs {
			uris[i] = toClientURIProto(&u)
		}
	}
	return &authv1.Client{
		ClientId:          result.ClientUUID.String(),
		Name:              result.Name,
		DisplayName:       result.DisplayName,
		ClientType:        result.ClientType,
		Domain:            stringPtr(result.Domain),
		Status:            result.Status,
		IsDefault:         result.IsDefault,
		IsSystem:          result.IsSystem,
		IdentityProvider:  idp,
		Uris:              uris,
		CreatedAt:         timestamppb.New(result.CreatedAt),
		UpdatedAt:         timestamppb.New(result.UpdatedAt),
		BrandingId:        brandingUUIDToString(result.BrandingUUID),
		AllowRegistration: result.AllowRegistration,
		// The bound service's UUID, absent when unbound — what lets a
		// get-or-create caller converge on an existing credential instead of
		// re-binding or duplicating it.
		ServiceId: result.ServiceUUID,
	}
}

func toClientURIProto(u *ClientURIServiceDataResult) *authv1.ClientURI {
	if u == nil {
		return nil
	}
	return &authv1.ClientURI{
		ClientUriId: u.ClientURIUUID.String(),
		Uri:         u.URI,
		Type:        u.Type,
		CreatedAt:   timestamppb.New(u.CreatedAt),
		UpdatedAt:   timestamppb.New(u.UpdatedAt),
	}
}

func toClientAPIPermissionProto(p *PermissionServiceDataResult) *authv1.ClientAPIPermission {
	if p == nil {
		return nil
	}
	return &authv1.ClientAPIPermission{
		PermissionId: p.PermissionUUID.String(),
		Name:         p.Name,
		Description:  p.Description,
		Status:       p.Status,
		IsSystem:     p.IsSystem,
		CreatedAt:    timestamppb.New(p.CreatedAt),
		UpdatedAt:    timestamppb.New(p.UpdatedAt),
	}
}

func stringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func brandingUUIDToString(b *uuid.UUID) string {
	if b == nil {
		return ""
	}
	return b.String()
}
