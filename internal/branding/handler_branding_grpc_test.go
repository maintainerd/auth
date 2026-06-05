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
		if err != nil { t.Fatal(err) }
		if res.Branding.CompanyName != "Acme" { t.Errorf("expected Acme, got %s", res.Branding.CompanyName) }
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testBrandingService{
			updateFn: func(ctx context.Context, tenantID int64, cn, lu, fu, pc, sc, ac, ff, cs, su, pp, ts string) (*BrandingServiceDataResult, error) { return &brResult, nil },
		}
		h := NewBrandingGRPCHandler(resolver, svc)
		_, err := h.UpdateBranding(ctx, &authv1.UpdateBrandingRequest{TenantUuid: tenantUUID.String(), CompanyName: "Acme", LogoUrl: "https://logo.png"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("validation error", func(t *testing.T) {
		h := NewBrandingGRPCHandler(resolver, &testBrandingService{})
		_, err := h.GetBranding(ctx, &authv1.GetBrandingRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testBrandingService{
			getFn: func(ctx context.Context, tenantID int64) (*BrandingServiceDataResult, error) { return nil, errors.New("db error") },
		}
		h := NewBrandingGRPCHandler(resolver, svc)
		_, err := h.GetBranding(ctx, &authv1.GetBrandingRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}
