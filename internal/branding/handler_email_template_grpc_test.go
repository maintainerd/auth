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

func TestEmailTemplateGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	etUUID := uuid.New()
	resolver := &testBrandingTenantResolver{}
	etResult := EmailTemplateServiceDataResult{EmailTemplateUUID: etUUID, Name: "welcome", Subject: "Hello", Status: "active"}

	t.Run("list success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
				return &EmailTemplateServiceListResult{Data: []EmailTemplateServiceDataResult{etResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		res, err := h.ListEmailTemplates(ctx, &authv1.ListEmailTemplatesRequest{TenantUuid: tenantUUID.String()})
		if err != nil { t.Fatal(err) }
		if len(res.EmailTemplates) != 1 { t.Fatalf("expected 1, got %d", len(res.EmailTemplates)) }
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) { return &etResult, nil },
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.GetEmailTemplate(ctx, &authv1.GetEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error) { return &etResult, nil },
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateEmailTemplate(ctx, &authv1.CreateEmailTemplateRequest{TenantUuid: tenantUUID.String(), Name: "welcome", Subject: "Hello", BodyHtml: "<p>Hi</p>", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) { return &etResult, nil },
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteEmailTemplate(ctx, &authv1.DeleteEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("validation error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
				return &EmailTemplateServiceListResult{}, nil
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.ListEmailTemplates(ctx, &authv1.ListEmailTemplatesRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.ListEmailTemplates(ctx, &authv1.ListEmailTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}
