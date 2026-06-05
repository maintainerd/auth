package idp

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

type IdentityProviderGRPCHandler struct {
	authv1.UnimplementedIdentityProviderServiceServer
	tenantResolver TenantResolver
	idpService     IdentityProviderService
}

func NewIdentityProviderGRPCHandler(tenantResolver TenantResolver, idpService IdentityProviderService) *IdentityProviderGRPCHandler {
	return &IdentityProviderGRPCHandler{tenantResolver: tenantResolver, idpService: idpService}
}

func (h *IdentityProviderGRPCHandler) ListIdentityProviders(ctx context.Context, req *authv1.ListIdentityProvidersRequest) (*authv1.ListIdentityProvidersResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	dto := IdentityProviderFilterDTO{
		Name:                 grpcStr(req.GetName()),
		DisplayName:          grpcStr(req.GetDisplayName()),
		Provider:             req.GetProvider(),
		ProviderType:         grpcStr(req.GetProviderType()),
		Identifier:           grpcStr(req.GetIdentifier()),
		Status:               req.GetStatus(),
		IsDefault:            req.IsDefault,
		IsSystem:             req.IsSystem,
		PaginationRequestDTO: grpcPagination(req.GetPagination()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.idpService.Get(ctx, IdentityProviderServiceGetFilter{
		Name:         dto.Name,
		DisplayName:  dto.DisplayName,
		Provider:     dto.Provider,
		ProviderType: dto.ProviderType,
		Identifier:   dto.Identifier,
		TenantID:     tenant.TenantID,
		Status:       dto.Status,
		IsDefault:    dto.IsDefault,
		IsSystem:     dto.IsSystem,
		Page:         dto.Page,
		Limit:        dto.Limit,
		SortBy:       dto.SortBy,
		SortOrder:    dto.SortOrder,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.IdentityProvider, len(result.Data))
	for i := range result.Data {
		rows[i] = toIdpProto(&result.Data[i])
	}
	return &authv1.ListIdentityProvidersResponse{IdentityProviders: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *IdentityProviderGRPCHandler) GetIdentityProvider(ctx context.Context, req *authv1.GetIdentityProviderRequest) (*authv1.GetIdentityProviderResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	idpUUID, err := grpcUUID(req.GetIdentityProviderUuid(), "IdentityProvider UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.idpService.GetByUUID(ctx, idpUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetIdentityProviderResponse{IdentityProvider: toIdpProto(result)}, nil
}

func (h *IdentityProviderGRPCHandler) CreateIdentityProvider(ctx context.Context, req *authv1.CreateIdentityProviderRequest) (*authv1.CreateIdentityProviderResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	actorUUID, err := grpcOptionalUUID(req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.idpService.Create(ctx, req.GetName(), req.GetDisplayName(), req.GetProvider(), req.GetProviderType(), structToJSON(req.GetConfig()), req.GetStatus(), tenant.TenantUUID.String(), tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateIdentityProviderResponse{IdentityProvider: toIdpProto(result)}, nil
}

func (h *IdentityProviderGRPCHandler) UpdateIdentityProvider(ctx context.Context, req *authv1.UpdateIdentityProviderRequest) (*authv1.UpdateIdentityProviderResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	idpUUID, err := grpcUUID(req.GetIdentityProviderUuid(), "IdentityProvider UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := grpcOptionalUUID(req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.idpService.Update(ctx, idpUUID, req.GetName(), req.GetDisplayName(), req.GetProvider(), req.GetProviderType(), structToJSON(req.GetConfig()), req.GetStatus(), tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateIdentityProviderResponse{IdentityProvider: toIdpProto(result)}, nil
}

func (h *IdentityProviderGRPCHandler) SetIdentityProviderStatus(ctx context.Context, req *authv1.SetIdentityProviderStatusRequest) (*authv1.SetIdentityProviderStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	idpUUID, err := grpcUUID(req.GetIdentityProviderUuid(), "IdentityProvider UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := grpcOptionalUUID(req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.idpService.SetStatusByUUID(ctx, idpUUID, req.GetStatus(), tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetIdentityProviderStatusResponse{IdentityProvider: toIdpProto(result)}, nil
}

func (h *IdentityProviderGRPCHandler) DeleteIdentityProvider(ctx context.Context, req *authv1.DeleteIdentityProviderRequest) (*authv1.DeleteIdentityProviderResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	idpUUID, err := grpcUUID(req.GetIdentityProviderUuid(), "IdentityProvider UUID")
	if err != nil {
		return nil, err
	}
	actorUUID, err := grpcOptionalUUID(req.GetActorUserUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.idpService.DeleteByUUID(ctx, idpUUID, tenant.TenantID, actorUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteIdentityProviderResponse{IdentityProvider: toIdpProto(result)}, nil
}

func (h *IdentityProviderGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := grpcUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
}

func toIdpProto(result *IdentityProviderServiceDataResult) *authv1.IdentityProvider {
	if result == nil {
		return nil
	}
	return &authv1.IdentityProvider{
		IdentityProviderUuid: result.IdentityProviderUUID.String(),
		Name:                 result.Name,
		DisplayName:          result.DisplayName,
		Provider:             result.Provider,
		ProviderType:         result.ProviderType,
		Identifier:           result.Identifier,
		Config:               jsonToStruct(result.Config),
		Status:               result.Status,
		IsDefault:            result.IsDefault,
		IsSystem:             result.IsSystem,
		CreatedAt:            timestamppb.New(result.CreatedAt),
		UpdatedAt:            timestamppb.New(result.UpdatedAt),
	}
}

func grpcUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func grpcOptionalUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	return grpcUUID(value, "UUID")
}

func grpcStr(value string) *string {
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

func structToJSON(s *structpb.Struct) datatypes.JSON {
	if s == nil {
		return nil
	}
	payload, _ := json.Marshal(s.AsMap())
	return datatypes.JSON(payload)
}

func jsonToStruct(raw *datatypes.JSON) *structpb.Struct {
	if raw == nil || len(*raw) == 0 {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(*raw, &values); err != nil {
		return nil
	}
	result, _ := structpb.NewStruct(values)
	return result
}
