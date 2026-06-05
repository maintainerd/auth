package branding

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSMSTemplateGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	stUUID := uuid.New()
	resolver := &testBrandingTenantResolver{}
	stResult := SMSTemplateServiceDataResult{SMSTemplateUUID: stUUID, Name: "otp", Message: "Your code is {{code}}", Status: "active"}

	t.Run("list success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error) {
				return &SMSTemplateServiceListResult{Data: []SMSTemplateServiceDataResult{stResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		res, err := h.ListSMSTemplates(ctx, &authv1.ListSMSTemplatesRequest{TenantUuid: tenantUUID.String()})
		if err != nil { t.Fatal(err) }
		if len(res.SmsTemplates) != 1 { t.Fatalf("expected 1, got %d", len(res.SmsTemplates)) }
	})

	t.Run("list validation error", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.ListSMSTemplates(ctx, &authv1.ListSMSTemplatesRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("list empty tenant uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.ListSMSTemplates(ctx, &authv1.ListSMSTemplatesRequest{})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("list tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewSMSTemplateGRPCHandler(errResolver, &testSMSTemplateService{})
		_, err := h.ListSMSTemplates(ctx, &authv1.ListSMSTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("list service error", func(t *testing.T) {
		svc := &testSMSTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.ListSMSTemplates(ctx, &authv1.ListSMSTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) { return &stResult, nil },
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.GetSMSTemplate(ctx, &authv1.GetSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("get bad tenant uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.GetSMSTemplate(ctx, &authv1.GetSMSTemplateRequest{TenantUuid: "bad", SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("get bad template uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.GetSMSTemplate(ctx, &authv1.GetSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("get tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewSMSTemplateGRPCHandler(errResolver, &testSMSTemplateService{})
		_, err := h.GetSMSTemplate(ctx, &authv1.GetSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testSMSTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.GetSMSTemplate(ctx, &authv1.GetSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name string, desc *string, msg string, sid *string, status string) (*SMSTemplateServiceDataResult, error) { return &stResult, nil },
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateSMSTemplate(ctx, &authv1.CreateSMSTemplateRequest{TenantUuid: tenantUUID.String(), Name: "otp", Message: "Your code is {{code}}", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("create validation error", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.CreateSMSTemplate(ctx, &authv1.CreateSMSTemplateRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("create tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewSMSTemplateGRPCHandler(errResolver, &testSMSTemplateService{})
		_, err := h.CreateSMSTemplate(ctx, &authv1.CreateSMSTemplateRequest{TenantUuid: tenantUUID.String(), Name: "otp", Message: "Your code is {{code}}", Status: "active"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("create service error", func(t *testing.T) {
		svc := &testSMSTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name string, desc *string, msg string, sid *string, status string) (*SMSTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateSMSTemplate(ctx, &authv1.CreateSMSTemplateRequest{TenantUuid: tenantUUID.String(), Name: "otp", Message: "Your code is {{code}}", Status: "active"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name string, desc *string, msg string, sid *string, status string) (*SMSTemplateServiceDataResult, error) {
				return &stResult, nil
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.UpdateSMSTemplate(ctx, &authv1.UpdateSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String(), Name: "updated", Message: "Updated message", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update bad tenant uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.UpdateSMSTemplate(ctx, &authv1.UpdateSMSTemplateRequest{TenantUuid: "bad", SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("update bad template uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.UpdateSMSTemplate(ctx, &authv1.UpdateSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("update tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewSMSTemplateGRPCHandler(errResolver, &testSMSTemplateService{})
		_, err := h.UpdateSMSTemplate(ctx, &authv1.UpdateSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testSMSTemplateService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name string, desc *string, msg string, sid *string, status string) (*SMSTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.UpdateSMSTemplate(ctx, &authv1.UpdateSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("set status success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error) {
				return &stResult, nil
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.SetSMSTemplateStatus(ctx, &authv1.SetSMSTemplateStatusRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String(), Status: "inactive"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("set status bad tenant uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.SetSMSTemplateStatus(ctx, &authv1.SetSMSTemplateStatusRequest{TenantUuid: "bad", SmsTemplateUuid: stUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("set status bad template uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.SetSMSTemplateStatus(ctx, &authv1.SetSMSTemplateStatusRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: "bad", Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("set status tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewSMSTemplateGRPCHandler(errResolver, &testSMSTemplateService{})
		_, err := h.SetSMSTemplateStatus(ctx, &authv1.SetSMSTemplateStatusRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("set status service error", func(t *testing.T) {
		svc := &testSMSTemplateService{
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*SMSTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.SetSMSTemplateStatus(ctx, &authv1.SetSMSTemplateStatusRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) { return &stResult, nil },
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteSMSTemplate(ctx, &authv1.DeleteSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("delete bad tenant uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.DeleteSMSTemplate(ctx, &authv1.DeleteSMSTemplateRequest{TenantUuid: "bad", SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("delete bad template uuid", func(t *testing.T) {
		h := NewSMSTemplateGRPCHandler(resolver, &testSMSTemplateService{})
		_, err := h.DeleteSMSTemplate(ctx, &authv1.DeleteSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("delete tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewSMSTemplateGRPCHandler(errResolver, &testSMSTemplateService{})
		_, err := h.DeleteSMSTemplate(ctx, &authv1.DeleteSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("delete service error", func(t *testing.T) {
		svc := &testSMSTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteSMSTemplate(ctx, &authv1.DeleteSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}

func TestSMSTemplateProto(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if smsTemplateProto(nil) != nil { t.Error("expected nil for nil input") }
	})

	t.Run("valid input", func(t *testing.T) {
		id := uuid.New()
		r := &SMSTemplateServiceDataResult{SMSTemplateUUID: id, Name: "otp", Message: "Your code is {{code}}", Status: "active"}
		p := smsTemplateProto(r)
		if p == nil { t.Fatal("expected non-nil proto") }
		if p.SmsTemplateUuid != id.String() { t.Errorf("expected %s, got %s", id.String(), p.SmsTemplateUuid) }
		if p.Name != "otp" { t.Errorf("expected otp, got %s", p.Name) }
	})
}
