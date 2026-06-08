package notifier

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
)

type SMSConfigGRPCHandler struct {
	authv1.UnimplementedSMSConfigServiceServer
	tenantResolver TenantResolver
	svc            SMSConfigService
}

func NewSMSConfigGRPCHandler(r TenantResolver, svc SMSConfigService) *SMSConfigGRPCHandler {
	return &SMSConfigGRPCHandler{tenantResolver: r, svc: svc}
}

func (h *SMSConfigGRPCHandler) GetSMSConfig(ctx context.Context, req *authv1.GetSMSConfigRequest) (*authv1.GetSMSConfigResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Get(ctx, t.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetSMSConfigResponse{Config: &authv1.SMSConfig{
		SmsConfigUuid: r.SMSConfigUUID.String(), Provider: r.Provider,
		AccountSid: r.AccountSID, FromNumber: r.FromNumber, SenderId: r.SenderID,
		TestMode: r.TestMode, Status: r.Status,
	}}, nil
}

func (h *SMSConfigGRPCHandler) UpdateSMSConfig(ctx context.Context, req *authv1.UpdateSMSConfigRequest) (*authv1.UpdateSMSConfigResponse, error) {
	t, err := h.tenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Update(ctx, t.TenantID, req.GetProvider(), req.GetAccountSid(), req.GetAuthToken(),
		req.GetFromNumber(), req.GetSenderId(), nil, req.TestMode)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateSMSConfigResponse{Config: &authv1.SMSConfig{
		SmsConfigUuid: r.SMSConfigUUID.String(), Provider: r.Provider,
		AccountSid: r.AccountSID, FromNumber: r.FromNumber, SenderId: r.SenderID,
		TestMode: r.TestMode, Status: r.Status,
	}}, nil
}

func (h *SMSConfigGRPCHandler) tenant(ctx context.Context, tuuid string) (*TenantServiceDataResult, error) {
	p, err := uuid.Parse(tuuid)
	if err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation("Invalid tenant UUID"))
	}
	r, err := h.tenantResolver.GetByUUID(ctx, p)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return r, nil
}
