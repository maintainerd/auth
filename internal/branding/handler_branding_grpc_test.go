package branding

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func TestBrandingGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	resolver := &testBrandingTenantResolver{}
	brResult := BrandingServiceDataResult{BrandingUUID: uuid.New(), CompanyName: "Acme", LogoURL: "https://logo.png"}

	t.Run("get success", func(t *testing.T) {
		svc := &testBrandingService{
			getFn: func(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) { return &brResult, nil },
		}
		h := NewBrandingGRPCHandler(resolver, svc)
		res, err := h.GetBranding(ctx, &authv1.GetBrandingRequest{TenantUuid: tenantUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Branding.CompanyName != "Acme" {
			t.Errorf("expected Acme, got %s", res.Branding.CompanyName)
		}
	})

	t.Run("get validation error", func(t *testing.T) {
		h := NewBrandingGRPCHandler(resolver, &testBrandingService{})
		_, err := h.GetBranding(ctx, &authv1.GetBrandingRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("get empty tenant uuid", func(t *testing.T) {
		svc := &testBrandingService{
			getFn: func(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) { return &brResult, nil },
		}
		h := NewBrandingGRPCHandler(resolver, svc)
		_, err := h.GetBranding(ctx, &authv1.GetBrandingRequest{})
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
		h := NewBrandingGRPCHandler(errResolver, &testBrandingService{})
		_, err := h.GetBranding(ctx, &authv1.GetBrandingRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testBrandingService{
			getFn: func(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewBrandingGRPCHandler(resolver, svc)
		_, err := h.GetBranding(ctx, &authv1.GetBrandingRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testBrandingService{
			updateFn: func(ctx context.Context, tenantID int64, nm, cn, lu, fu string, md datatypes.JSON, su, pp, ts string) (*BrandingServiceDataResult, error) {
				return &brResult, nil
			},
		}
		h := NewBrandingGRPCHandler(resolver, svc)
		_, err := h.UpdateBranding(ctx, &authv1.UpdateBrandingRequest{TenantUuid: tenantUUID.String(), CompanyName: "Acme", LogoUrl: "https://logo.png"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update validation error", func(t *testing.T) {
		h := NewBrandingGRPCHandler(resolver, &testBrandingService{})
		_, err := h.UpdateBranding(ctx, &authv1.UpdateBrandingRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("update empty tenant uuid", func(t *testing.T) {
		h := NewBrandingGRPCHandler(resolver, &testBrandingService{})
		_, err := h.UpdateBranding(ctx, &authv1.UpdateBrandingRequest{})
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
		h := NewBrandingGRPCHandler(errResolver, &testBrandingService{})
		_, err := h.UpdateBranding(ctx, &authv1.UpdateBrandingRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testBrandingService{
			updateFn: func(ctx context.Context, tenantID int64, nm, cn, lu, fu string, md datatypes.JSON, su, pp, ts string) (*BrandingServiceDataResult, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewBrandingGRPCHandler(resolver, svc)
		_, err := h.UpdateBranding(ctx, &authv1.UpdateBrandingRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})
}

func TestParseUUID(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		id, err := parseUUID("", "TestLabel")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
		if id != uuid.Nil {
			t.Errorf("expected Nil UUID, got %s", id.String())
		}
	})

	t.Run("bad format", func(t *testing.T) {
		id, err := parseUUID("not-a-uuid", "BadUUID")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
		if id != uuid.Nil {
			t.Errorf("expected Nil UUID, got %s", id.String())
		}
	})

	t.Run("valid uuid", func(t *testing.T) {
		valid := uuid.New()
		id, err := parseUUID(valid.String(), "ValidUUID")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != valid {
			t.Errorf("expected %s, got %s", valid.String(), id.String())
		}
	})
}

func TestOptionalStr(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		if optionalStr("") != nil {
			t.Error("expected nil for empty string")
		}
	})

	t.Run("non-empty returns pointer", func(t *testing.T) {
		v := "hello"
		p := optionalStr(v)
		if p == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *p != v {
			t.Errorf("expected '%s', got '%s'", v, *p)
		}
	})
}

func TestGRPCPagination(t *testing.T) {
	t.Run("nil pagination returns defaults", func(t *testing.T) {
		pg := grpcPagination(nil)
		if pg.Page != 1 {
			t.Errorf("expected Page=1, got %d", pg.Page)
		}
		if pg.Limit != 20 {
			t.Errorf("expected Limit=20, got %d", pg.Limit)
		}
	})

	t.Run("zero page falls back to 1", func(t *testing.T) {
		pg := grpcPagination(&authv1.Pagination{Page: 0, Limit: 20})
		if pg.Page != 1 {
			t.Errorf("expected Page=1, got %d", pg.Page)
		}
		if pg.Limit != 20 {
			t.Errorf("expected Limit=20, got %d", pg.Limit)
		}
	})

	t.Run("zero limit falls back to default", func(t *testing.T) {
		pg := grpcPagination(&authv1.Pagination{Page: 3, Limit: 0})
		if pg.Page != 3 {
			t.Errorf("expected Page=3, got %d", pg.Page)
		}
		if pg.Limit != 20 {
			t.Errorf("expected Limit=20, got %d", pg.Limit)
		}
	})

	t.Run("full pagination", func(t *testing.T) {
		pg := grpcPagination(&authv1.Pagination{Page: 2, Limit: 50, SortBy: "name", SortOrder: "asc"})
		if pg.Page != 2 {
			t.Errorf("expected Page=2, got %d", pg.Page)
		}
		if pg.Limit != 50 {
			t.Errorf("expected Limit=50, got %d", pg.Limit)
		}
		if pg.SortBy != "name" {
			t.Errorf("expected SortBy=name, got %s", pg.SortBy)
		}
		if pg.SortOrder != "asc" {
			t.Errorf("expected SortOrder=asc, got %s", pg.SortOrder)
		}
	})
}

func TestPageProto(t *testing.T) {
	t.Run("converts correctly", func(t *testing.T) {
		p := pageProto(42, 2, 10, 5)
		if p.Total != 42 {
			t.Errorf("expected Total=42, got %d", p.Total)
		}
		if p.Page != 2 {
			t.Errorf("expected Page=2, got %d", p.Page)
		}
		if p.Limit != 10 {
			t.Errorf("expected Limit=10, got %d", p.Limit)
		}
		if p.TotalPages != 5 {
			t.Errorf("expected TotalPages=5, got %d", p.TotalPages)
		}
	})
}

func TestBrandingProto(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if brandingProto(nil) != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("valid input", func(t *testing.T) {
		id := uuid.New()
		r := &BrandingServiceDataResult{BrandingUUID: id, CompanyName: "TestCo", LogoURL: "https://logo.png"}
		p := brandingProto(r)
		if p == nil {
			t.Fatal("expected non-nil proto")
		}
		if p.BrandingUuid != id.String() {
			t.Errorf("expected %s, got %s", id.String(), p.BrandingUuid)
		}
		if p.CompanyName != "TestCo" {
			t.Errorf("expected TestCo, got %s", p.CompanyName)
		}
	})
}
