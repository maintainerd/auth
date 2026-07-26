package tenant

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/structpb"
)

// TenantSettingGRPCHandler exposes tenant OPERATIONAL settings (rate limit,
// audit, maintenance) to the control plane. These are tenant-management concerns
// core owns — NOT security settings (password/MFA/lockout/session/token/threat/
// IP rules), which stay REST/console-only and have no gRPC surface. Every RPC is
// gated by grpcAuthorizeTenantManagement: a system-tenant principal (the control
// plane) or a member of the target tenant.
type TenantSettingGRPCHandler struct {
	authv1.UnimplementedTenantSettingServiceServer
	tenantService       TenantService
	tenantMemberService TenantMemberService
	settingService      TenantSettingService
}

func NewTenantSettingGRPCHandler(tenantService TenantService, tenantMemberService TenantMemberService, settingService TenantSettingService) *TenantSettingGRPCHandler {
	return &TenantSettingGRPCHandler{
		tenantService:       tenantService,
		tenantMemberService: tenantMemberService,
		settingService:      settingService,
	}
}

// resolveAndAuthorize parses the target tenant UUID, enforces the tenant-
// management boundary, and returns the tenant's internal ID.
func (h *TenantSettingGRPCHandler) resolveAndAuthorize(ctx context.Context, tenantUUIDStr string) (int64, error) {
	tenantUUID, err := parseGRPCUUID(tenantUUIDStr, "Tenant UUID")
	if err != nil {
		return 0, err
	}
	if err := grpcAuthorizeTenantManagement(ctx, h.tenantService, h.tenantMemberService, tenantUUID); err != nil {
		return 0, err
	}
	t, err := h.tenantService.GetByUUID(ctx, tenantUUID)
	if err != nil {
		return 0, apperror.ToGRPCError(err)
	}
	return t.TenantID, nil
}

func configStruct(cfg map[string]any) (*structpb.Struct, error) {
	if cfg == nil {
		cfg = map[string]any{}
	}
	s, err := structpb.NewStruct(cfg)
	if err != nil {
		return nil, apperror.ToGRPCError(apperror.NewInternal("failed to encode config", err))
	}
	return s, nil
}

func (h *TenantSettingGRPCHandler) GetRateLimitConfig(ctx context.Context, req *authv1.GetRateLimitConfigRequest) (*authv1.GetRateLimitConfigResponse, error) {
	tenantID, err := h.resolveAndAuthorize(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	cfg, err := h.settingService.GetRateLimitConfig(ctx, tenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	s, err := configStruct(cfg)
	if err != nil {
		return nil, err
	}
	return &authv1.GetRateLimitConfigResponse{Config: s}, nil
}

func (h *TenantSettingGRPCHandler) UpdateRateLimitConfig(ctx context.Context, req *authv1.UpdateRateLimitConfigRequest) (*authv1.UpdateRateLimitConfigResponse, error) {
	tenantID, err := h.resolveAndAuthorize(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	if _, err := h.settingService.UpdateRateLimitConfig(ctx, tenantID, req.GetConfig().AsMap()); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	cfg, err := h.settingService.GetRateLimitConfig(ctx, tenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	s, err := configStruct(cfg)
	if err != nil {
		return nil, err
	}
	return &authv1.UpdateRateLimitConfigResponse{Config: s}, nil
}

func (h *TenantSettingGRPCHandler) GetAuditConfig(ctx context.Context, req *authv1.GetAuditConfigRequest) (*authv1.GetAuditConfigResponse, error) {
	tenantID, err := h.resolveAndAuthorize(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	cfg, err := h.settingService.GetAuditConfig(ctx, tenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	s, err := configStruct(cfg)
	if err != nil {
		return nil, err
	}
	return &authv1.GetAuditConfigResponse{Config: s}, nil
}

func (h *TenantSettingGRPCHandler) UpdateAuditConfig(ctx context.Context, req *authv1.UpdateAuditConfigRequest) (*authv1.UpdateAuditConfigResponse, error) {
	tenantID, err := h.resolveAndAuthorize(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	if _, err := h.settingService.UpdateAuditConfig(ctx, tenantID, req.GetConfig().AsMap()); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	cfg, err := h.settingService.GetAuditConfig(ctx, tenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	s, err := configStruct(cfg)
	if err != nil {
		return nil, err
	}
	return &authv1.UpdateAuditConfigResponse{Config: s}, nil
}

func (h *TenantSettingGRPCHandler) GetMaintenanceConfig(ctx context.Context, req *authv1.GetMaintenanceConfigRequest) (*authv1.GetMaintenanceConfigResponse, error) {
	tenantID, err := h.resolveAndAuthorize(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	cfg, err := h.settingService.GetMaintenanceConfig(ctx, tenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	s, err := configStruct(cfg)
	if err != nil {
		return nil, err
	}
	return &authv1.GetMaintenanceConfigResponse{Config: s}, nil
}

func (h *TenantSettingGRPCHandler) UpdateMaintenanceConfig(ctx context.Context, req *authv1.UpdateMaintenanceConfigRequest) (*authv1.UpdateMaintenanceConfigResponse, error) {
	tenantID, err := h.resolveAndAuthorize(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	if _, err := h.settingService.UpdateMaintenanceConfig(ctx, tenantID, req.GetConfig().AsMap()); err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	cfg, err := h.settingService.GetMaintenanceConfig(ctx, tenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	s, err := configStruct(cfg)
	if err != nil {
		return nil, err
	}
	return &authv1.UpdateMaintenanceConfigResponse{Config: s}, nil
}
