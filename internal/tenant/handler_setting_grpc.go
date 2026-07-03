package tenant

import (
	"context"
	"encoding/json"

	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

type TenantSettingGRPCHandler struct {
	authv1.UnimplementedTenantSettingServiceServer
	tenantService        TenantService
	tenantSettingService TenantSettingService
	authEventService     authevent.AuthEventService
}

func NewTenantSettingGRPCHandler(tenantService TenantService, tenantSettingService TenantSettingService, authEventService ...authevent.AuthEventService) *TenantSettingGRPCHandler {
	var eventService authevent.AuthEventService
	if len(authEventService) > 0 {
		eventService = authEventService[0]
	}
	return &TenantSettingGRPCHandler{tenantService: tenantService, tenantSettingService: tenantSettingService, authEventService: eventService}
}

func (h *TenantSettingGRPCHandler) GetRateLimitConfig(ctx context.Context, req *authv1.GetRateLimitConfigRequest) (*authv1.GetRateLimitConfigResponse, error) {
	config, err := h.getConfig(ctx, req.GetTenantUuid(), h.tenantSettingService.GetRateLimitConfig)
	if err != nil {
		return nil, err
	}
	return &authv1.GetRateLimitConfigResponse{Config: config}, nil
}

func (h *TenantSettingGRPCHandler) UpdateRateLimitConfig(ctx context.Context, req *authv1.UpdateRateLimitConfigRequest) (*authv1.UpdateRateLimitConfigResponse, error) {
	config, err := h.updateConfig(ctx, req.GetTenantUuid(), req.GetConfig(), h.tenantSettingService.UpdateRateLimitConfig, func(dto TenantSettingUpdateConfigRequestDTO) error {
		return dto.ValidateRateLimitConfig()
	}, func(result *TenantSettingServiceDataResult) map[string]any {
		return result.RateLimitConfig
	})
	if err != nil {
		return nil, err
	}
	return &authv1.UpdateRateLimitConfigResponse{Config: config}, nil
}

func (h *TenantSettingGRPCHandler) GetAuditConfig(ctx context.Context, req *authv1.GetAuditConfigRequest) (*authv1.GetAuditConfigResponse, error) {
	config, err := h.getConfig(ctx, req.GetTenantUuid(), h.tenantSettingService.GetAuditConfig)
	if err != nil {
		return nil, err
	}
	return &authv1.GetAuditConfigResponse{Config: config}, nil
}

func (h *TenantSettingGRPCHandler) UpdateAuditConfig(ctx context.Context, req *authv1.UpdateAuditConfigRequest) (*authv1.UpdateAuditConfigResponse, error) {
	config, err := h.updateConfig(ctx, req.GetTenantUuid(), req.GetConfig(), h.tenantSettingService.UpdateAuditConfig, func(dto TenantSettingUpdateConfigRequestDTO) error {
		return dto.ValidateAuditConfig()
	}, func(result *TenantSettingServiceDataResult) map[string]any {
		return result.AuditConfig
	})
	if err != nil {
		return nil, err
	}
	return &authv1.UpdateAuditConfigResponse{Config: config}, nil
}

func (h *TenantSettingGRPCHandler) GetMaintenanceConfig(ctx context.Context, req *authv1.GetMaintenanceConfigRequest) (*authv1.GetMaintenanceConfigResponse, error) {
	config, err := h.getConfig(ctx, req.GetTenantUuid(), h.tenantSettingService.GetMaintenanceConfig)
	if err != nil {
		return nil, err
	}
	return &authv1.GetMaintenanceConfigResponse{Config: config}, nil
}

func (h *TenantSettingGRPCHandler) UpdateMaintenanceConfig(ctx context.Context, req *authv1.UpdateMaintenanceConfigRequest) (*authv1.UpdateMaintenanceConfigResponse, error) {
	config, err := h.updateConfig(ctx, req.GetTenantUuid(), req.GetConfig(), h.tenantSettingService.UpdateMaintenanceConfig, func(dto TenantSettingUpdateConfigRequestDTO) error {
		return dto.ValidateMaintenanceConfig()
	}, func(result *TenantSettingServiceDataResult) map[string]any {
		return result.MaintenanceConfig
	})
	if err != nil {
		return nil, err
	}
	h.logMaintenanceConfigUpdatedGRPC(ctx, req.GetTenantUuid(), config)
	return &authv1.UpdateMaintenanceConfigResponse{Config: config}, nil
}

func (h *TenantSettingGRPCHandler) getConfig(ctx context.Context, tenantUUID string, getter func(context.Context, int64) (map[string]any, error)) (*structpb.Struct, error) {
	tenant, err := h.resolveTenant(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	config, err := getter(ctx, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return configProto(config), nil
}

func (h *TenantSettingGRPCHandler) updateConfig(
	ctx context.Context,
	tenantUUID string,
	config *structpb.Struct,
	updater func(context.Context, int64, map[string]any) (*TenantSettingServiceDataResult, error),
	validator func(TenantSettingUpdateConfigRequestDTO) error,
	selector func(*TenantSettingServiceDataResult) map[string]any,
) (*structpb.Struct, error) {
	tenant, err := h.resolveTenant(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	payload := structMap(config)
	dto := TenantSettingUpdateConfigRequestDTO(payload)
	if validator == nil {
		validator = TenantSettingUpdateConfigRequestDTO.Validate
	}
	if err := validator(dto); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := updater(ctx, tenant.TenantID, payload)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return configProto(selector(result)), nil
}

func (h *TenantSettingGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseGRPCUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	tenant, err := h.tenantService.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return tenant, nil
}

func structMap(metadata interface{ AsMap() map[string]any }) map[string]any {
	if metadata == nil {
		return nil
	}
	return metadata.AsMap()
}

func configProto(config map[string]any) *structpb.Struct {
	if len(config) == 0 {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	result, _ := structpb.NewStruct(config)
	return result
}

func (h *TenantSettingGRPCHandler) logMaintenanceConfigUpdatedGRPC(ctx context.Context, tenantUUID string, config *structpb.Struct) {
	if h.authEventService == nil {
		return
	}
	tenant, err := h.resolveTenant(ctx, tenantUUID)
	if err != nil {
		return
	}
	var ip string
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		ip = p.Addr.String()
	}
	description := "Maintenance config updated"
	metadata, _ := json.Marshal(map[string]any{"config": config.AsMap()})
	h.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenant.TenantID,
		IPAddress:   ip,
		Category:    authevent.AuthEventCategorySystem,
		EventType:   authevent.AuthEventTypeMaintenanceConfigUpdated,
		Severity:    authevent.AuthEventSeverityWarn,
		Result:      authevent.AuthEventResultSuccess,
		Description: &description,
		Metadata:    datatypes.JSON(metadata),
	})
}
