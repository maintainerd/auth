package branding

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
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
		if err != nil {
			t.Fatal(err)
		}
		if len(res.LoginTemplates) != 1 {
			t.Fatalf("expected 1, got %d", len(res.LoginTemplates))
		}
	})

	t.Run("list validation error", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.ListLoginTemplates(ctx, &authv1.ListLoginTemplatesRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("list empty tenant uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.ListLoginTemplates(ctx, &authv1.ListLoginTemplatesRequest{})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("list tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewLoginTemplateGRPCHandler(errResolver, &testLoginTemplateService{})
		_, err := h.ListLoginTemplates(ctx, &authv1.ListLoginTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("list service error", func(t *testing.T) {
		svc := &testLoginTemplateService{
			getAllFn: func(ctx context.Context, tenantID int64, name *string, status []string, template *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*LoginTemplateServiceListResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.ListLoginTemplates(ctx, &authv1.ListLoginTemplatesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
				return &ltResult, nil
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.GetLoginTemplate(ctx, &authv1.GetLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("get bad tenant uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.GetLoginTemplate(ctx, &authv1.GetLoginTemplateRequest{TenantUuid: "bad", LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("get bad template uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.GetLoginTemplate(ctx, &authv1.GetLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("get tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewLoginTemplateGRPCHandler(errResolver, &testLoginTemplateService{})
		_, err := h.GetLoginTemplate(ctx, &authv1.GetLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testLoginTemplateService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.GetLoginTemplate(ctx, &authv1.GetLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name string, desc *string, tmpl string, meta map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
				return &ltResult, nil
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateLoginTemplate(ctx, &authv1.CreateLoginTemplateRequest{TenantUuid: tenantUUID.String(), Name: "default-login", Template: "<html></html>", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create validation error", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.CreateLoginTemplate(ctx, &authv1.CreateLoginTemplateRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("create tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewLoginTemplateGRPCHandler(errResolver, &testLoginTemplateService{})
		_, err := h.CreateLoginTemplate(ctx, &authv1.CreateLoginTemplateRequest{TenantUuid: tenantUUID.String(), Name: "default-login", Template: "<html></html>", Status: "active"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("create service error", func(t *testing.T) {
		svc := &testLoginTemplateService{
			createFn: func(ctx context.Context, tenantID int64, name string, desc *string, tmpl string, meta map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.CreateLoginTemplate(ctx, &authv1.CreateLoginTemplateRequest{TenantUuid: tenantUUID.String(), Name: "default-login", Template: "<html></html>", Status: "active"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name string, desc *string, tmpl string, meta map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
				return &ltResult, nil
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.UpdateLoginTemplate(ctx, &authv1.UpdateLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String(), Name: "updated", Template: "<html></html>", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update bad tenant uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.UpdateLoginTemplate(ctx, &authv1.UpdateLoginTemplateRequest{TenantUuid: "bad", LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("update bad template uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.UpdateLoginTemplate(ctx, &authv1.UpdateLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("update tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewLoginTemplateGRPCHandler(errResolver, &testLoginTemplateService{})
		_, err := h.UpdateLoginTemplate(ctx, &authv1.UpdateLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testLoginTemplateService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name string, desc *string, tmpl string, meta map[string]any, status string) (*LoginTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.UpdateLoginTemplate(ctx, &authv1.UpdateLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("set status success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*LoginTemplateServiceDataResult, error) {
				return &ltResult, nil
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.SetLoginTemplateStatus(ctx, &authv1.SetLoginTemplateStatusRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String(), Status: "inactive"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("set status bad tenant uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.SetLoginTemplateStatus(ctx, &authv1.SetLoginTemplateStatusRequest{TenantUuid: "bad", LoginTemplateUuid: ltUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("set status bad template uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.SetLoginTemplateStatus(ctx, &authv1.SetLoginTemplateStatusRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: "bad", Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("set status tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewLoginTemplateGRPCHandler(errResolver, &testLoginTemplateService{})
		_, err := h.SetLoginTemplateStatus(ctx, &authv1.SetLoginTemplateStatusRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("set status service error", func(t *testing.T) {
		svc := &testLoginTemplateService{
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*LoginTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.SetLoginTemplateStatus(ctx, &authv1.SetLoginTemplateStatusRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testLoginTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
				return &ltResult, nil
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteLoginTemplate(ctx, &authv1.DeleteLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete bad tenant uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.DeleteLoginTemplate(ctx, &authv1.DeleteLoginTemplateRequest{TenantUuid: "bad", LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("delete bad template uuid", func(t *testing.T) {
		h := NewLoginTemplateGRPCHandler(resolver, &testLoginTemplateService{})
		_, err := h.DeleteLoginTemplate(ctx, &authv1.DeleteLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("delete tenant resolver error", func(t *testing.T) {
		errResolver := &testBrandingTenantResolver{
			getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
				return nil, errors.New("tenant not found")
			},
		}
		h := NewLoginTemplateGRPCHandler(errResolver, &testLoginTemplateService{})
		_, err := h.DeleteLoginTemplate(ctx, &authv1.DeleteLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("delete service error", func(t *testing.T) {
		svc := &testLoginTemplateService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*LoginTemplateServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewLoginTemplateGRPCHandler(resolver, svc)
		_, err := h.DeleteLoginTemplate(ctx, &authv1.DeleteLoginTemplateRequest{TenantUuid: tenantUUID.String(), LoginTemplateUuid: ltUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})
}

func TestStructMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if structMap(nil) != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("valid struct returns map", func(t *testing.T) {
		s, err := structpb.NewStruct(map[string]any{"key": "value", "num": float64(42)})
		if err != nil {
			t.Fatal(err)
		}
		m := structMap(s)
		if m == nil {
			t.Fatal("expected non-nil map")
		}
		if m["key"] != "value" {
			t.Errorf("expected value, got %v", m["key"])
		}
	})
}

func TestLoginTemplateProto(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if loginTemplateProto(nil) != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("valid input without metadata", func(t *testing.T) {
		id := uuid.New()
		r := &LoginTemplateServiceDataResult{LoginTemplateUUID: id, Name: "default-login", Template: "<html></html>", Status: "active"}
		p := loginTemplateProto(r)
		if p == nil {
			t.Fatal("expected non-nil proto")
		}
		if p.LoginTemplateUuid != id.String() {
			t.Errorf("expected %s, got %s", id.String(), p.LoginTemplateUuid)
		}
		if p.Name != "default-login" {
			t.Errorf("expected default-login, got %s", p.Name)
		}
		if p.Metadata != nil {
			t.Errorf("expected nil metadata, got %v", p.Metadata)
		}
	})

	t.Run("valid input with metadata", func(t *testing.T) {
		id := uuid.New()
		r := &LoginTemplateServiceDataResult{
			LoginTemplateUUID: id,
			Name:              "default-login",
			Template:          "<html></html>",
			Status:            "active",
			Metadata:          map[string]any{"key": "value", "num": float64(42)},
		}
		p := loginTemplateProto(r)
		if p == nil {
			t.Fatal("expected non-nil proto")
		}
		if p.Metadata == nil {
			t.Fatal("expected non-nil metadata")
		}
		if p.Metadata.Fields["key"].GetStringValue() != "value" {
			t.Errorf("expected value, got %v", p.Metadata.Fields["key"])
		}
	})
}
