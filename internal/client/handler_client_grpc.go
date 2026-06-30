package client

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
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
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := ClientFilterDTO{
		Name:                 optionalStr(req.GetName()),
		DisplayName:          optionalStr(req.GetDisplayName()),
		ClientType:           req.GetClientType(),
		IdentityProviderUUID: optionalStr(req.GetIdentityProviderUuid()),
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
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.GetByUUID(ctx, clientUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetClientResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) GetClientSecret(ctx context.Context, req *authv1.GetClientSecretRequest) (*authv1.GetClientSecretResponse, error) {
	return &authv1.GetClientSecretResponse{
		Message: "Client secrets cannot be retrieved after creation. Use RotateSecret to obtain a new secret.",
	}, nil
}

func (h *ClientGRPCHandler) RotateClientSecret(ctx context.Context, req *authv1.RotateClientSecretRequest) (*authv1.RotateClientSecretResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	newSecret, err := h.clientService.RotateSecret(ctx, clientUUID, tenant.TenantID, actorUUID, int(req.GetGracePeriodHours()))
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RotateClientSecretResponse{ClientSecret: newSecret}, nil
}

func (h *ClientGRPCHandler) GetClientConfig(ctx context.Context, req *authv1.GetClientConfigRequest) (*authv1.GetClientConfigResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
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

func (h *ClientGRPCHandler) CreateClient(ctx context.Context, req *authv1.CreateClientRequest) (*authv1.CreateClientResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
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
	result, err := h.clientService.Create(ctx, tenant.TenantID, req.GetName(), req.GetDisplayName(), req.GetClientType(), req.GetDomain(), configJSON, req.GetStatus(), false, req.GetIdentityProviderUuid(), nil, allowReg, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateClientResponse{
		Client: toClientProto(result.Client),
		Credentials: &authv1.ClientCredentials{
			ClientUuid:   result.Client.ClientUUID.String(),
			ClientId:     result.ClientIdentifier,
			ClientSecret: result.PlaintextSecret,
		},
	}, nil
}

func (h *ClientGRPCHandler) UpdateClient(ctx context.Context, req *authv1.UpdateClientRequest) (*authv1.UpdateClientResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	config := structProtoToMap(req.GetConfig())
	configJSON, err := mapToJSON(config)
	if err != nil {
		return nil, err
	}
	result, err := h.clientService.Update(ctx, clientUUID, tenant.TenantID, req.GetName(), req.GetDisplayName(), req.GetClientType(), req.GetDomain(), configJSON, req.GetStatus(), false, nil, req.AllowRegistration, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateClientResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) SetClientStatus(ctx context.Context, req *authv1.SetClientStatusRequest) (*authv1.SetClientStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	result, err := h.clientService.SetStatusByUUID(ctx, clientUUID, tenant.TenantID, req.GetStatus(), actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetClientStatusResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) DeleteClient(ctx context.Context, req *authv1.DeleteClientRequest) (*authv1.DeleteClientResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	result, err := h.clientService.DeleteByUUID(ctx, clientUUID, tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteClientResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) ListClientURIs(ctx context.Context, req *authv1.ListClientURIsRequest) (*authv1.ListClientURIsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
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

func (h *ClientGRPCHandler) CreateClientURI(ctx context.Context, req *authv1.CreateClientURIRequest) (*authv1.CreateClientURIResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
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
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	uriUUID, err := parseUUID(req.GetClientUriUuid(), "Client URI UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
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
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	uriUUID, err := parseUUID(req.GetClientUriUuid(), "Client URI UUID")
	if err != nil {
		return nil, err
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	result, err := h.clientService.DeleteURI(ctx, clientUUID, tenant.TenantID, uriUUID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteClientURIResponse{Client: toClientProto(result)}, nil
}

func (h *ClientGRPCHandler) ListClientAPIs(ctx context.Context, req *authv1.ListClientAPIsRequest) (*authv1.ListClientAPIsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
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
			ClientApiUuid: api.ClientAPIUUID.String(),
			Api: &authv1.ClientAPIDetail{
				ApiUuid:     api.Api.APIUUID.String(),
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
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUIDs := make([]uuid.UUID, len(req.GetApiUuids()))
	for i, a := range req.GetApiUuids() {
		parsed, err := parseUUID(a, "API UUID")
		if err != nil {
			return nil, err
		}
		apiUUIDs[i] = parsed
	}
	if err := h.clientService.AddClientAPIs(ctx, tenant.TenantID, clientUUID, apiUUIDs); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddClientAPIsResponse{Message: "APIs added to client successfully"}, nil
}

func (h *ClientGRPCHandler) RemoveClientAPI(ctx context.Context, req *authv1.RemoveClientAPIRequest) (*authv1.RemoveClientAPIResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	if err := h.clientService.RemoveClientAPI(ctx, tenant.TenantID, clientUUID, apiUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveClientAPIResponse{Message: "API removed from client successfully"}, nil
}

func (h *ClientGRPCHandler) ListClientAPIPermissions(ctx context.Context, req *authv1.ListClientAPIPermissionsRequest) (*authv1.ListClientAPIPermissionsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiUuid(), "API UUID")
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
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	permUUIDs := make([]uuid.UUID, len(req.GetPermissionUuids()))
	for i, p := range req.GetPermissionUuids() {
		parsed, err := parseUUID(p, "Permission UUID")
		if err != nil {
			return nil, err
		}
		permUUIDs[i] = parsed
	}
	if err := h.clientService.AddClientAPIPermissions(ctx, tenant.TenantID, clientUUID, apiUUID, permUUIDs); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddClientAPIPermissionsResponse{Message: "Permissions added to client API successfully"}, nil
}

func (h *ClientGRPCHandler) RemoveClientAPIPermission(ctx context.Context, req *authv1.RemoveClientAPIPermissionRequest) (*authv1.RemoveClientAPIPermissionResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	clientUUID, err := parseUUID(req.GetClientUuid(), "Client UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	permUUID, err := parseUUID(req.GetPermissionUuid(), "Permission UUID")
	if err != nil {
		return nil, err
	}
	if err := h.clientService.RemoveClientAPIPermission(ctx, tenant.TenantID, clientUUID, apiUUID, permUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveClientAPIPermissionResponse{Message: "Permission removed from client API successfully"}, nil
}

func (h *ClientGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
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
			IdentityProviderUuid: result.IdentityProvider.IdentityProviderUUID.String(),
			Name:                 result.IdentityProvider.Name,
			DisplayName:          result.IdentityProvider.DisplayName,
			Provider:             result.IdentityProvider.Provider,
			ProviderType:         result.IdentityProvider.ProviderType,
			Identifier:           result.IdentityProvider.Identifier,
			Status:               result.IdentityProvider.Status,
			IsDefault:            result.IdentityProvider.IsDefault,
			IsSystem:             result.IdentityProvider.IsSystem,
			CreatedAt:            timestamppb.New(result.IdentityProvider.CreatedAt),
			UpdatedAt:            timestamppb.New(result.IdentityProvider.UpdatedAt),
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
		ClientUuid:        result.ClientUUID.String(),
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
	}
}

func toClientURIProto(u *ClientURIServiceDataResult) *authv1.ClientURI {
	if u == nil {
		return nil
	}
	return &authv1.ClientURI{
		ClientUriUuid: u.ClientURIUUID.String(),
		Uri:           u.URI,
		Type:          u.Type,
		CreatedAt:     timestamppb.New(u.CreatedAt),
		UpdatedAt:     timestamppb.New(u.UpdatedAt),
	}
}

func toClientAPIPermissionProto(p *PermissionServiceDataResult) *authv1.ClientAPIPermission {
	if p == nil {
		return nil
	}
	return &authv1.ClientAPIPermission{
		PermissionUuid: p.PermissionUUID.String(),
		Name:           p.Name,
		Description:    p.Description,
		Status:         p.Status,
		IsSystem:       p.IsSystem,
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
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
