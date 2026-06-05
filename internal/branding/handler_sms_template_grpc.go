package branding

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SMSTemplateGRPCHandler struct {
	authv1.UnimplementedSMSTemplateServiceServer
	tenantResolver TenantResolver
	svc            SMSTemplateService
}

func NewSMSTemplateGRPCHandler(tenantResolver TenantResolver, svc SMSTemplateService) *SMSTemplateGRPCHandler {
	return &SMSTemplateGRPCHandler{tenantResolver: tenantResolver, svc: svc}
}

func (h *SMSTemplateGRPCHandler) ListSMSTemplates(ctx context.Context, req *authv1.ListSMSTemplatesRequest) (*authv1.ListSMSTemplatesResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	pg := grpcPagination(req.GetPagination())
	result, err := h.svc.GetAll(ctx, tenant.TenantID, optionalStr(req.GetName()), req.GetStatus(), req.IsDefault, req.IsSystem, pg.Page, pg.Limit, pg.SortBy, pg.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.SMSTemplate, len(result.Data))
	for i := range result.Data {
		rows[i] = smsTemplateProto(&result.Data[i])
	}
	return &authv1.ListSMSTemplatesResponse{SmsTemplates: rows, Page: pageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *SMSTemplateGRPCHandler) GetSMSTemplate(ctx context.Context, req *authv1.GetSMSTemplateRequest) (*authv1.GetSMSTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	stUUID, err := parseUUID(req.GetSmsTemplateUuid(), "SMS template UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.GetByUUID(ctx, stUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetSMSTemplateResponse{SmsTemplate: smsTemplateProto(result)}, nil
}

func (h *SMSTemplateGRPCHandler) CreateSMSTemplate(ctx context.Context, req *authv1.CreateSMSTemplateRequest) (*authv1.CreateSMSTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.Create(ctx, tenant.TenantID, req.GetName(), req.Description, req.GetMessage(), req.SenderId, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateSMSTemplateResponse{SmsTemplate: smsTemplateProto(result)}, nil
}

func (h *SMSTemplateGRPCHandler) UpdateSMSTemplate(ctx context.Context, req *authv1.UpdateSMSTemplateRequest) (*authv1.UpdateSMSTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	stUUID, err := parseUUID(req.GetSmsTemplateUuid(), "SMS template UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.Update(ctx, stUUID, tenant.TenantID, req.GetName(), req.Description, req.GetMessage(), req.SenderId, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateSMSTemplateResponse{SmsTemplate: smsTemplateProto(result)}, nil
}

func (h *SMSTemplateGRPCHandler) SetSMSTemplateStatus(ctx context.Context, req *authv1.SetSMSTemplateStatusRequest) (*authv1.SetSMSTemplateStatusResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	stUUID, err := parseUUID(req.GetSmsTemplateUuid(), "SMS template UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.UpdateStatus(ctx, stUUID, tenant.TenantID, req.GetStatus())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetSMSTemplateStatusResponse{SmsTemplate: smsTemplateProto(result)}, nil
}

func (h *SMSTemplateGRPCHandler) DeleteSMSTemplate(ctx context.Context, req *authv1.DeleteSMSTemplateRequest) (*authv1.DeleteSMSTemplateResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	stUUID, err := parseUUID(req.GetSmsTemplateUuid(), "SMS template UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.svc.Delete(ctx, stUUID, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteSMSTemplateResponse{SmsTemplate: smsTemplateProto(result)}, nil
}

func (h *SMSTemplateGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
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

func smsTemplateProto(r *SMSTemplateServiceDataResult) *authv1.SMSTemplate {
	if r == nil {
		return nil
	}
	return &authv1.SMSTemplate{
		SmsTemplateUuid: r.SMSTemplateUUID.String(), Name: r.Name, Description: r.Description,
		Message: r.Message, SenderId: r.SenderID, Status: r.Status,
		IsDefault: r.IsDefault, IsSystem: r.IsSystem,
		CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}
