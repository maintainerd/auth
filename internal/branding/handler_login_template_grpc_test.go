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

func TestLoginTemplateGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	ltUUID := uuid.New()
	resolver := &testBrandingTenantResolver{}
	ltResult := LoginTemplateServiceDataResult{LoginTemplateUUID: ltUUID, Name: "default-login", Template: "<html></html>", Status: "active"}

	t.Run("list success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error) {
				return &LoginTemplateServiceListResult{Data: []LoginTemplateServiceDataResult{ltResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		res, err := h.ListLoginTemplates(ctx, &authv1.ListLoginTemplatesRequest{TenantUuid: tenantUUID.String()})
		if err != nil { t.Fatal(err) }
		if len(res.LoginTemplates) != 1 { t.Fatalf("expected 1, got %d", len(res.LoginTemplates)) }
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) { return &ltResult, nil },
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.GetLoginTemplate(ctx, &authv1.GetLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name string, desc *string, tmpl string, meta map[string]any, status string) (*LoginTemplateServiceDataResult, error) { return &ltResult, nil },
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateLoginTemplate(ctx, &authv1.CreateLoginTemplateRequest{TenantUuid: tenantUUID.String(), Name: "default-login", Template: "<html></html>", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) { return &ltResult, nil },
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteLoginTemplate(ctx, &authv1.DeleteLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testLoginTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.ListLoginTemplates(ctx, &authv1.ListLoginTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}
