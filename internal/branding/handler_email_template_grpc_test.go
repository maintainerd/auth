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

	t.Run("list validation error", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.ListEmailTemplates(ctx, &authv1.ListEmailTemplatesRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("list empty tenant uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.ListEmailTemplates(ctx, &authv1.ListEmailTemplatesRequest{})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("list tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewEmailTemplateGRPCHandler(errResolver, &testEmailTemplateService{})
		_, err := h.ListEmailTemplates(ctx, &authv1.ListEmailTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("list service error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*EmailTemplateServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.ListEmailTemplates(ctx, &authv1.ListEmailTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) { return &etResult, nil },
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.GetEmailTemplate(ctx, &authv1.GetEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("get bad tenant uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.GetEmailTemplate(ctx, &authv1.GetEmailTemplateRequest{TenantUuid: "bad", EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("get bad template uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.GetEmailTemplate(ctx, &authv1.GetEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("get tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewEmailTemplateGRPCHandler(errResolver, &testEmailTemplateService{})
		_, err := h.GetEmailTemplate(ctx, &authv1.GetEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.GetEmailTemplate(ctx, &authv1.GetEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error) { return &etResult, nil },
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateEmailTemplate(ctx, &authv1.CreateEmailTemplateRequest{TenantUuid: tenantUUID.String(), Name: "welcome", Subject: "Hello", BodyHtml: "<p>Hi</p>", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("create validation error", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.CreateEmailTemplate(ctx, &authv1.CreateEmailTemplateRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("create tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewEmailTemplateGRPCHandler(errResolver, &testEmailTemplateService{})
		_, err := h.CreateEmailTemplate(ctx, &authv1.CreateEmailTemplateRequest{TenantUuid: tenantUUID.String(), Name: "welcome", Subject: "Hello", BodyHtml: "<p>Hi</p>", Status: "active"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("create service error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*EmailTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateEmailTemplate(ctx, &authv1.CreateEmailTemplateRequest{TenantUuid: tenantUUID.String(), Name: "welcome", Subject: "Hello", BodyHtml: "<p>Hi</p>", Status: "active"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error) {
				return &etResult, nil
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.UpdateEmailTemplate(ctx, &authv1.UpdateEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String(), Name: "updated", Subject: "Updated", BodyHtml: "<p>Updated</p>", Status: "active"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update bad tenant uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.UpdateEmailTemplate(ctx, &authv1.UpdateEmailTemplateRequest{TenantUuid: "bad", EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("update bad template uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.UpdateEmailTemplate(ctx, &authv1.UpdateEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("update tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewEmailTemplateGRPCHandler(errResolver, &testEmailTemplateService{})
		_, err := h.UpdateEmailTemplate(ctx, &authv1.UpdateEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*EmailTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.UpdateEmailTemplate(ctx, &authv1.UpdateEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("set status success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*EmailTemplateServiceDataResult, error) {
				return &etResult, nil
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.SetEmailTemplateStatus(ctx, &authv1.SetEmailTemplateStatusRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String(), Status: "inactive"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("set status bad tenant uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.SetEmailTemplateStatus(ctx, &authv1.SetEmailTemplateStatusRequest{TenantUuid: "bad", EmailTemplateUuid: etUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("set status bad template uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.SetEmailTemplateStatus(ctx, &authv1.SetEmailTemplateStatusRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: "bad", Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("set status tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewEmailTemplateGRPCHandler(errResolver, &testEmailTemplateService{})
		_, err := h.SetEmailTemplateStatus(ctx, &authv1.SetEmailTemplateStatusRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("set status service error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*EmailTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.SetEmailTemplateStatus(ctx, &authv1.SetEmailTemplateStatusRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testEmailTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) { return &etResult, nil },
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteEmailTemplate(ctx, &authv1.DeleteEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if err != nil { t.Fatal(err) }
	})

	t.Run("delete bad tenant uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.DeleteEmailTemplate(ctx, &authv1.DeleteEmailTemplateRequest{TenantUuid: "bad", EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("delete bad template uuid", func(t *testing.T) {
		h := NewEmailTemplateGRPCHandler(resolver, &testEmailTemplateService{})
		_, err := h.DeleteEmailTemplate(ctx, &authv1.DeleteEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("delete tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewEmailTemplateGRPCHandler(errResolver, &testEmailTemplateService{})
		_, err := h.DeleteEmailTemplate(ctx, &authv1.DeleteEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("delete service error", func(t *testing.T) {
		svc := &testEmailTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*EmailTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewEmailTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteEmailTemplate(ctx, &authv1.DeleteEmailTemplateRequest{TenantUuid: tenantUUID.String(), EmailTemplateUuid: etUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}

func TestEmailTemplateProto(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if emailTemplateProto(nil) != nil { t.Error("expected nil for nil input") }
	})

	t.Run("valid input", func(t *testing.T) {
		id := uuid.New()
		r := &EmailTemplateServiceDataResult{EmailTemplateUUID: id, Name: "welcome", Subject: "Hello", BodyHTML: "<p>Hi</p>", Status: "active"}
		p := emailTemplateProto(r)
		if p == nil { t.Fatal("expected non-nil proto") }
		if p.EmailTemplateUuid != id.String() { t.Errorf("expected %s, got %s", id.String(), p.EmailTemplateUuid) }
		if p.Name != "welcome" { t.Errorf("expected welcome, got %s", p.Name) }
	})
}
