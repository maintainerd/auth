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

	t.Run("get success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) { return &stResult, nil },
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.GetSMSTemplate(ctx, &authv1.GetSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name string, desc *string, msg string, sid *string, status string) (*SMSTemplateServiceDataResult, error) { return &stResult, nil },
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateSMSTemplate(ctx, &authv1.CreateSMSTemplateRequest{TenantUuid: tenantUUID.String(), Name: "otp", Message: "Your code is {{code}}", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testSMSTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SMSTemplateServiceDataResult, error) { return &stResult, nil },
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteSMSTemplate(ctx, &authv1.DeleteSMSTemplateRequest{TenantUuid: tenantUUID.String(), SmsTemplateUuid: stUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testSMSTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*SMSTemplateServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSMSTemplateGRPCHandler(resolver, svc)
		_, err := h.ListSMSTemplates(ctx, &authv1.ListSMSTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}
