package client

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type APIKeyGRPCHandler struct {
	authv1.UnimplementedAPIKeyServiceServer
	tenantResolver TenantResolver
	apiKeyService  APIKeyService
}

func NewAPIKeyGRPCHandler(tenantResolver TenantResolver, apiKeyService APIKeyService) *APIKeyGRPCHandler {
	return &APIKeyGRPCHandler{tenantResolver: tenantResolver, apiKeyService: apiKeyService}
}

func (h *APIKeyGRPCHandler) ListAPIKeys(ctx context.Context, req *authv1.ListAPIKeysRequest) (*authv1.ListAPIKeysResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.apiKeyService.Get(ctx, APIKeyServiceGetFilter{
		TenantID:    tenant.TenantID,
		Name:        optionalStr(req.GetName()),
		Description: optionalStr(req.GetDescription()),
		Status:      optionalStr(req.GetStatus()),
		Page:        dto.Page,
		Limit:       dto.Limit,
		SortBy:      dto.SortBy,
		SortOrder:   dto.SortOrder,
	}, uuid.Nil)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.APIKey, len(result.Data))
	for i := range result.Data {
		rows[i] = toAPIKeyProto(&result.Data[i])
	}
	return &authv1.ListAPIKeysResponse{ApiKeys: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *APIKeyGRPCHandler) GetAPIKey(ctx context.Context, req *authv1.GetAPIKeyRequest) (*authv1.GetAPIKeyResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.apiKeyService.GetByUUID(ctx, akUUID, tenant.TenantID, uuid.Nil)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetAPIKeyResponse{ApiKey: toAPIKeyProto(result)}, nil
}

func (h *APIKeyGRPCHandler) GetAPIKeyConfig(ctx context.Context, req *authv1.GetAPIKeyConfigRequest) (*authv1.GetAPIKeyConfigResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
	if err != nil {
		return nil, err
	}
	config, err := h.apiKeyService.GetConfigByUUID(ctx, akUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	configProto := jsonToConfigProto(config)
	return &authv1.GetAPIKeyConfigResponse{Config: configProto}, nil
}

func (h *APIKeyGRPCHandler) CreateAPIKey(ctx context.Context, req *authv1.CreateAPIKeyRequest) (*authv1.CreateAPIKeyResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	config := structProtoToMap(req.GetConfig())
	configJSON, _ := mapToJSON(config)
	var expiresAt *time.Time
	if req.GetExpiresAt() != nil {
		t := req.GetExpiresAt().AsTime()
		expiresAt = &t
	}
	var rateLimit *int
	if req.RateLimit != nil {
		v := int(req.GetRateLimit())
		rateLimit = &v
	}
	result, rawKey, err := h.apiKeyService.Create(ctx, tenant.TenantID, req.GetName(), req.GetDescription(), configJSON, expiresAt, rateLimit, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateAPIKeyResponse{ApiKey: toAPIKeyProto(result), Key: rawKey}, nil
}

func (h *APIKeyGRPCHandler) UpdateAPIKey(ctx context.Context, req *authv1.UpdateAPIKeyRequest) (*authv1.UpdateAPIKeyResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
	if err != nil {
		return nil, err
	}
	config := structProtoToMap(req.GetConfig())
	configJSON, _ := mapToJSON(config)
	var expiresAt *time.Time
	if req.GetExpiresAt() != nil {
		t := req.GetExpiresAt().AsTime()
		expiresAt = &t
	}
	var rateLimit *int
	if req.RateLimit != nil {
		v := int(req.GetRateLimit())
		rateLimit = &v
	}
	actorUUID := uuid.Nil
	if req.GetActorUserUuid() != "" {
		actorUUID, err = parseUUID(req.GetActorUserUuid(), "Actor user UUID")
		if err != nil {
			return nil, err
		}
	}
	result, err := h.apiKeyService.Update(ctx, akUUID, tenant.TenantID, optionalStr(req.GetName()), optionalStr(req.GetDescription()), configJSON, expiresAt, rateLimit, optionalStr(req.GetStatus()), actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateAPIKeyResponse{ApiKey: toAPIKeyProto(result)}, nil
}

func (h *APIKeyGRPCHandler) SetAPIKeyStatus(ctx context.Context, req *authv1.SetAPIKeyStatusRequest) (*authv1.SetAPIKeyStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.apiKeyService.SetStatusByUUID(ctx, akUUID, tenant.TenantID, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetAPIKeyStatusResponse{ApiKey: toAPIKeyProto(result)}, nil
}

func (h *APIKeyGRPCHandler) DeleteAPIKey(ctx context.Context, req *authv1.DeleteAPIKeyRequest) (*authv1.DeleteAPIKeyResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
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
	result, err := h.apiKeyService.Delete(ctx, akUUID, tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteAPIKeyResponse{ApiKey: toAPIKeyProto(result)}, nil
}

func (h *APIKeyGRPCHandler) ListAPIKeyAPIs(ctx context.Context, req *authv1.ListAPIKeyAPIsRequest) (*authv1.ListAPIKeyAPIsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	result, err := h.apiKeyService.GetAPIKeyAPIs(ctx, tenant.TenantID, akUUID, dto.Page, dto.Limit, dto.SortBy, dto.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	apis := make([]*authv1.APIKeyAPI, len(result.Data))
	for i, api := range result.Data {
		permProtos := make([]*authv1.ClientAPIPermission, len(api.Permissions))
		for j, perm := range api.Permissions {
			permProtos[j] = toClientAPIPermissionProto(&perm)
		}
		apis[i] = &authv1.APIKeyAPI{
			ApiKeyApiUuid: api.APIKeyAPIUUID.String(),
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
	return &authv1.ListAPIKeyAPIsResponse{Apis: apis, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *APIKeyGRPCHandler) AddAPIKeyAPIs(ctx context.Context, req *authv1.AddAPIKeyAPIsRequest) (*authv1.AddAPIKeyAPIsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
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
	if err := h.apiKeyService.AddAPIKeyAPIs(ctx, tenant.TenantID, akUUID, apiUUIDs); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddAPIKeyAPIsResponse{Message: "APIs added to API key successfully"}, nil
}

func (h *APIKeyGRPCHandler) RemoveAPIKeyAPI(ctx context.Context, req *authv1.RemoveAPIKeyAPIRequest) (*authv1.RemoveAPIKeyAPIResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	if err := h.apiKeyService.RemoveAPIKeyAPI(ctx, tenant.TenantID, akUUID, apiUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveAPIKeyAPIResponse{Message: "API removed from API key successfully"}, nil
}

func (h *APIKeyGRPCHandler) ListAPIKeyAPIPermissions(ctx context.Context, req *authv1.ListAPIKeyAPIPermissionsRequest) (*authv1.ListAPIKeyAPIPermissionsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
	if err != nil {
		return nil, err
	}
	apiUUID, err := parseUUID(req.GetApiUuid(), "API UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.apiKeyService.GetAPIKeyAPIPermissions(ctx, tenant.TenantID, akUUID, apiUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	perms := make([]*authv1.ClientAPIPermission, len(result))
	for i, perm := range result {
		perms[i] = toClientAPIPermissionProto(&perm)
	}
	return &authv1.ListAPIKeyAPIPermissionsResponse{Permissions: perms}, nil
}

func (h *APIKeyGRPCHandler) AddAPIKeyAPIPermissions(ctx context.Context, req *authv1.AddAPIKeyAPIPermissionsRequest) (*authv1.AddAPIKeyAPIPermissionsResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
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
	if err := h.apiKeyService.AddAPIKeyAPIPermissions(ctx, tenant.TenantID, akUUID, apiUUID, permUUIDs); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.AddAPIKeyAPIPermissionsResponse{Message: "Permissions added to API key API successfully"}, nil
}

func (h *APIKeyGRPCHandler) RemoveAPIKeyAPIPermission(ctx context.Context, req *authv1.RemoveAPIKeyAPIPermissionRequest) (*authv1.RemoveAPIKeyAPIPermissionResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	akUUID, err := parseUUID(req.GetApiKeyUuid(), "APIKey UUID")
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
	if err := h.apiKeyService.RemoveAPIKeyAPIPermission(ctx, tenant.TenantID, akUUID, apiUUID, permUUID); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RemoveAPIKeyAPIPermissionResponse{Message: "Permission removed from API key API successfully"}, nil
}

func (h *APIKeyGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
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

func toAPIKeyProto(result *APIKeyServiceDataResult) *authv1.APIKey {
	if result == nil {
		return nil
	}
	var expiresAt *timestamppb.Timestamp
	if result.ExpiresAt != nil {
		expiresAt = timestamppb.New(*result.ExpiresAt)
	}
	var rateLimit *int32
	if result.RateLimit != nil {
		v := int32(*result.RateLimit)
		rateLimit = &v
	}
	return &authv1.APIKey{
		ApiKeyUuid:  result.APIKeyUUID.String(),
		Name:        result.Name,
		Description: result.Description,
		KeyPrefix:   result.KeyPrefix,
		ExpiresAt:   expiresAt,
		RateLimit:   rateLimit,
		Status:      result.Status,
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
}

func jsonToConfigProto(config datatypes.JSON) *structpb.Struct {
	if len(config) == 0 {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	result, _ := structpb.NewStruct(m)
	return result
}
