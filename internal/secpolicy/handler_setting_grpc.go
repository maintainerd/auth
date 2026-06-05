package secpolicy

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/structpb"
)

type SecuritySettingGRPCHandler struct {
	authv1.UnimplementedSecuritySettingServiceServer
	svc SecuritySettingService
}

func NewSecuritySettingGRPCHandler(svc SecuritySettingService) *SecuritySettingGRPCHandler {
	return &SecuritySettingGRPCHandler{svc: svc}
}

func structMap(s *structpb.Struct) map[string]any {
	if s == nil { return nil }
	return s.AsMap()
}

func configProto(cfg map[string]any) *structpb.Struct {
	if cfg == nil { return &structpb.Struct{Fields: map[string]*structpb.Value{}} }
	s, _ := structpb.NewStruct(cfg)
	return s
}

func (h *SecuritySettingGRPCHandler) GetMFAConfig(ctx context.Context, req *authv1.GetMFAConfigRequest) (*authv1.GetMFAConfigResponse, error) {
	cfg, err := h.svc.GetMFAConfig(ctx, req.GetUserPoolId())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetMFAConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) UpdateMFAConfig(ctx context.Context, req *authv1.UpdateMFAConfigRequest) (*authv1.UpdateMFAConfigResponse, error) {
	_, err := h.svc.UpdateMFAConfig(ctx, req.GetUserPoolId(), structMap(req.GetConfig()), 0, req.GetIpAddress(), req.GetUserAgent())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	cfg, _ := h.svc.GetMFAConfig(ctx, req.GetUserPoolId())
	return &authv1.UpdateMFAConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) GetPasswordConfig(ctx context.Context, req *authv1.GetPasswordConfigRequest) (*authv1.GetPasswordConfigResponse, error) {
	cfg, err := h.svc.GetPasswordConfig(ctx, req.GetUserPoolId())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetPasswordConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) UpdatePasswordConfig(ctx context.Context, req *authv1.UpdatePasswordConfigRequest) (*authv1.UpdatePasswordConfigResponse, error) {
	_, err := h.svc.UpdatePasswordConfig(ctx, req.GetUserPoolId(), structMap(req.GetConfig()), 0, req.GetIpAddress(), req.GetUserAgent())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	cfg, _ := h.svc.GetPasswordConfig(ctx, req.GetUserPoolId())
	return &authv1.UpdatePasswordConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) GetSessionConfig(ctx context.Context, req *authv1.GetSessionConfigRequest) (*authv1.GetSessionConfigResponse, error) {
	cfg, err := h.svc.GetSessionConfig(ctx, req.GetUserPoolId())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetSessionConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) UpdateSessionConfig(ctx context.Context, req *authv1.UpdateSessionConfigRequest) (*authv1.UpdateSessionConfigResponse, error) {
	_, err := h.svc.UpdateSessionConfig(ctx, req.GetUserPoolId(), structMap(req.GetConfig()), 0, req.GetIpAddress(), req.GetUserAgent())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	cfg, _ := h.svc.GetSessionConfig(ctx, req.GetUserPoolId())
	return &authv1.UpdateSessionConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) GetThreatConfig(ctx context.Context, req *authv1.GetThreatConfigRequest) (*authv1.GetThreatConfigResponse, error) {
	cfg, err := h.svc.GetThreatConfig(ctx, req.GetUserPoolId())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetThreatConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) UpdateThreatConfig(ctx context.Context, req *authv1.UpdateThreatConfigRequest) (*authv1.UpdateThreatConfigResponse, error) {
	_, err := h.svc.UpdateThreatConfig(ctx, req.GetUserPoolId(), structMap(req.GetConfig()), 0, req.GetIpAddress(), req.GetUserAgent())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	cfg, _ := h.svc.GetThreatConfig(ctx, req.GetUserPoolId())
	return &authv1.UpdateThreatConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) GetLockoutConfig(ctx context.Context, req *authv1.GetLockoutConfigRequest) (*authv1.GetLockoutConfigResponse, error) {
	cfg, err := h.svc.GetLockoutConfig(ctx, req.GetUserPoolId())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetLockoutConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) UpdateLockoutConfig(ctx context.Context, req *authv1.UpdateLockoutConfigRequest) (*authv1.UpdateLockoutConfigResponse, error) {
	_, err := h.svc.UpdateLockoutConfig(ctx, req.GetUserPoolId(), structMap(req.GetConfig()), 0, req.GetIpAddress(), req.GetUserAgent())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	cfg, _ := h.svc.GetLockoutConfig(ctx, req.GetUserPoolId())
	return &authv1.UpdateLockoutConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) GetRegistrationConfig(ctx context.Context, req *authv1.GetRegistrationConfigRequest) (*authv1.GetRegistrationConfigResponse, error) {
	cfg, err := h.svc.GetRegistrationConfig(ctx, req.GetUserPoolId())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetRegistrationConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) UpdateRegistrationConfig(ctx context.Context, req *authv1.UpdateRegistrationConfigRequest) (*authv1.UpdateRegistrationConfigResponse, error) {
	_, err := h.svc.UpdateRegistrationConfig(ctx, req.GetUserPoolId(), structMap(req.GetConfig()), 0, req.GetIpAddress(), req.GetUserAgent())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	cfg, _ := h.svc.GetRegistrationConfig(ctx, req.GetUserPoolId())
	return &authv1.UpdateRegistrationConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) GetTokenConfig(ctx context.Context, req *authv1.GetTokenConfigRequest) (*authv1.GetTokenConfigResponse, error) {
	cfg, err := h.svc.GetTokenConfig(ctx, req.GetUserPoolId())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	return &authv1.GetTokenConfigResponse{Config: configProto(cfg)}, nil
}
func (h *SecuritySettingGRPCHandler) UpdateTokenConfig(ctx context.Context, req *authv1.UpdateTokenConfigRequest) (*authv1.UpdateTokenConfigResponse, error) {
	_, err := h.svc.UpdateTokenConfig(ctx, req.GetUserPoolId(), structMap(req.GetConfig()), 0, req.GetIpAddress(), req.GetUserAgent())
	if err != nil { return nil, apperror.ToGRPCError(err) }
	cfg, _ := h.svc.GetTokenConfig(ctx, req.GetUserPoolId())
	return &authv1.UpdateTokenConfigResponse{Config: configProto(cfg)}, nil
}
