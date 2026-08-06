package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestTenantSettingGRPCHandler(t *testing.T) {
	const sysTID = int64(1)
	tenantUUID := uuid.New()
	sysCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{TenantID: sysTID, UserUUID: uuid.New()})
	tenantSvc := func() *mockTenantService {
		return &mockTenantService{
			getSystemFn: func() (*TenantServiceDataResult, error) {
				return &TenantServiceDataResult{TenantID: sysTID, IsSystem: true}, nil
			},
			getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
				return &TenantServiceDataResult{TenantID: 42, TenantUUID: tenantUUID}, nil
			},
		}
	}

	t.Run("system principal reads maintenance config", func(t *testing.T) {
		h := NewTenantSettingGRPCHandler(tenantSvc(), &mockTenantMemberService{}, &mockTenantSettingService{
			getMaintenanceConfigFn: func(tid int64) (map[string]any, error) {
				assert.Equal(t, int64(42), tid)
				return map[string]any{"enabled": true, "message": "brb"}, nil
			},
		})
		resp, err := h.GetMaintenanceConfig(sysCtx, &authv1.GetMaintenanceConfigRequest{TenantUuid: tenantUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, true, resp.Config.AsMap()["enabled"])
		assert.Equal(t, "brb", resp.Config.AsMap()["message"])
	})

	t.Run("system principal updates rate-limit config and gets the fresh value", func(t *testing.T) {
		var updatedWith map[string]any
		h := NewTenantSettingGRPCHandler(tenantSvc(), &mockTenantMemberService{}, &mockTenantSettingService{
			updateRateLimitConfigFn: func(_ int64, cfg map[string]any) (*TenantSettingServiceDataResult, error) {
				updatedWith = cfg
				return &TenantSettingServiceDataResult{}, nil
			},
			getRateLimitConfigFn: func(int64) (map[string]any, error) {
				return map[string]any{"enabled": true, "requests_per_window": float64(100)}, nil
			},
		})
		in, _ := structpb.NewStruct(map[string]any{"enabled": true, "requests_per_window": float64(100)})
		resp, err := h.UpdateRateLimitConfig(sysCtx, &authv1.UpdateRateLimitConfigRequest{TenantUuid: tenantUUID.String(), Config: in})
		require.NoError(t, err)
		assert.Equal(t, true, updatedWith["enabled"])
		assert.Equal(t, float64(100), resp.Config.AsMap()["requests_per_window"])
	})

	t.Run("principal of another tenant is denied", func(t *testing.T) {
		h := NewTenantSettingGRPCHandler(tenantSvc(), &mockTenantMemberService{
			canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil },
		}, &mockTenantSettingService{})
		otherCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{TenantID: 777, UserUUID: uuid.New()})
		_, err := h.GetMaintenanceConfig(otherCtx, &authv1.GetMaintenanceConfigRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	// Regression: the Update* RPCs wrote GetConfig().AsMap() straight to the
	// JSONB column, skipping the Validate*Config the REST handlers run — so the
	// control plane could store values the console could never produce and the
	// readers then had to survive.
	t.Run("update RPCs run the same validation as REST", func(t *testing.T) {
		var reached bool
		h := NewTenantSettingGRPCHandler(tenantSvc(), &mockTenantMemberService{}, &mockTenantSettingService{
			updateRateLimitConfigFn: func(int64, map[string]any) (*TenantSettingServiceDataResult, error) {
				reached = true
				return &TenantSettingServiceDataResult{}, nil
			},
			updateAuditConfigFn: func(int64, map[string]any) (*TenantSettingServiceDataResult, error) {
				reached = true
				return &TenantSettingServiceDataResult{}, nil
			},
			updateMaintenanceConfigFn: func(int64, map[string]any) (*TenantSettingServiceDataResult, error) {
				reached = true
				return &TenantSettingServiceDataResult{}, nil
			},
		})

		// Out of range: requests_per_window caps at 100000.
		badRate, _ := structpb.NewStruct(map[string]any{"requests_per_window": float64(999999)})
		_, err := h.UpdateRateLimitConfig(sysCtx, &authv1.UpdateRateLimitConfigRequest{TenantUuid: tenantUUID.String(), Config: badRate})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		// Unknown field.
		badAudit, _ := structpb.NewStruct(map[string]any{"retention_dayz": float64(30)})
		_, err = h.UpdateAuditConfig(sysCtx, &authv1.UpdateAuditConfigRequest{TenantUuid: tenantUUID.String(), Config: badAudit})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		// Wrong type for a known field.
		badMaint, _ := structpb.NewStruct(map[string]any{"enabled": "yes"})
		_, err = h.UpdateMaintenanceConfig(sysCtx, &authv1.UpdateMaintenanceConfigRequest{TenantUuid: tenantUUID.String(), Config: badMaint})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		assert.False(t, reached, "an invalid config must never reach the service")
	})

	t.Run("unauthenticated is rejected", func(t *testing.T) {
		h := NewTenantSettingGRPCHandler(tenantSvc(), &mockTenantMemberService{}, &mockTenantSettingService{})
		_, err := h.GetAuditConfig(context.Background(), &authv1.GetAuditConfigRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}
