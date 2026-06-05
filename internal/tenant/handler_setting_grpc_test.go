package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestTenantSettingGRPCHandler_ConfigRPCs(t *testing.T) {
	tenantUUID := uuid.New()
	baseTenant := &mockTenantService{getByUUIDFn: func(id uuid.UUID) (*TenantServiceDataResult, error) {
		assert.Equal(t, tenantUUID, id)
		return &TenantServiceDataResult{TenantID: 77, TenantUUID: id}, nil
	}}
	cfg := map[string]any{"enabled": true}
	cfgStruct, err := structpb.NewStruct(cfg)
	require.NoError(t, err)

	t.Run("get config rpc success", func(t *testing.T) {
		settingSvc := &mockTenantSettingService{
			getRateLimitConfigFn: func(tenantID int64) (map[string]any, error) {
				assert.Equal(t, int64(77), tenantID)
				return cfg, nil
			},
			getAuditConfigFn: func(tenantID int64) (map[string]any, error) { return cfg, nil },
			getMaintenanceConfigFn: func(tenantID int64) (map[string]any, error) {
				return map[string]any{}, nil
			},
			getFeatureFlagsFn: func(tenantID int64) (map[string]any, error) { return cfg, nil },
		}
		h := NewTenantSettingGRPCHandler(baseTenant, settingSvc)

		rateLimit, err := h.GetRateLimitConfig(context.Background(), &authv1.GetRateLimitConfigRequest{TenantUuid: tenantUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, true, rateLimit.Config.AsMap()["enabled"])
		audit, err := h.GetAuditConfig(context.Background(), &authv1.GetAuditConfigRequest{TenantUuid: tenantUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, true, audit.Config.AsMap()["enabled"])
		maintenance, err := h.GetMaintenanceConfig(context.Background(), &authv1.GetMaintenanceConfigRequest{TenantUuid: tenantUUID.String()})
		require.NoError(t, err)
		assert.Empty(t, maintenance.Config.AsMap())
		features, err := h.GetFeatureFlags(context.Background(), &authv1.GetFeatureFlagsRequest{TenantUuid: tenantUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, true, features.Config.AsMap()["enabled"])
	})

	t.Run("update config rpc success", func(t *testing.T) {
		settingSvc := &mockTenantSettingService{
			updateRateLimitConfigFn: func(tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
				assert.Equal(t, true, config["enabled"])
				return &TenantSettingServiceDataResult{RateLimitConfig: config}, nil
			},
			updateAuditConfigFn: func(tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
				return &TenantSettingServiceDataResult{AuditConfig: config}, nil
			},
			updateMaintenanceConfigFn: func(tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
				return &TenantSettingServiceDataResult{MaintenanceConfig: config}, nil
			},
			updateFeatureFlagsFn: func(tenantID int64, config map[string]any) (*TenantSettingServiceDataResult, error) {
				return &TenantSettingServiceDataResult{FeatureFlags: config}, nil
			},
		}
		h := NewTenantSettingGRPCHandler(baseTenant, settingSvc)

		rateLimit, err := h.UpdateRateLimitConfig(context.Background(), &authv1.UpdateRateLimitConfigRequest{TenantUuid: tenantUUID.String(), Config: cfgStruct})
		require.NoError(t, err)
		assert.Equal(t, true, rateLimit.Config.AsMap()["enabled"])
		audit, err := h.UpdateAuditConfig(context.Background(), &authv1.UpdateAuditConfigRequest{TenantUuid: tenantUUID.String(), Config: cfgStruct})
		require.NoError(t, err)
		assert.Equal(t, true, audit.Config.AsMap()["enabled"])
		maintenance, err := h.UpdateMaintenanceConfig(context.Background(), &authv1.UpdateMaintenanceConfigRequest{TenantUuid: tenantUUID.String(), Config: cfgStruct})
		require.NoError(t, err)
		assert.Equal(t, true, maintenance.Config.AsMap()["enabled"])
		features, err := h.UpdateFeatureFlags(context.Background(), &authv1.UpdateFeatureFlagsRequest{TenantUuid: tenantUUID.String(), Config: cfgStruct})
		require.NoError(t, err)
		assert.Equal(t, true, features.Config.AsMap()["enabled"])
	})

	t.Run("tenant and service errors map to grpc", func(t *testing.T) {
		h := NewTenantSettingGRPCHandler(&mockTenantService{}, &mockTenantSettingService{})
		_, err := h.GetRateLimitConfig(context.Background(), &authv1.GetRateLimitConfigRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetAuditConfig(context.Background(), &authv1.GetAuditConfigRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetMaintenanceConfig(context.Background(), &authv1.GetMaintenanceConfigRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetFeatureFlags(context.Background(), &authv1.GetFeatureFlagsRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateAuditConfig(context.Background(), &authv1.UpdateAuditConfigRequest{TenantUuid: "bad", Config: cfgStruct})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateMaintenanceConfig(context.Background(), &authv1.UpdateMaintenanceConfigRequest{TenantUuid: "bad", Config: cfgStruct})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateFeatureFlags(context.Background(), &authv1.UpdateFeatureFlagsRequest{TenantUuid: "bad", Config: cfgStruct})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		h = NewTenantSettingGRPCHandler(&mockTenantService{getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing")
		}}, &mockTenantSettingService{})
		_, err = h.GetRateLimitConfig(context.Background(), &authv1.GetRateLimitConfigRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))

		h = NewTenantSettingGRPCHandler(baseTenant, &mockTenantSettingService{
			getRateLimitConfigFn: func(int64) (map[string]any, error) { return nil, errors.New("db") },
			getAuditConfigFn:     func(int64) (map[string]any, error) { return nil, errors.New("db") },
		})
		_, err = h.GetRateLimitConfig(context.Background(), &authv1.GetRateLimitConfigRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetAuditConfig(context.Background(), &authv1.GetAuditConfigRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))

		h = NewTenantSettingGRPCHandler(baseTenant, &mockTenantSettingService{})
		_, err = h.UpdateRateLimitConfig(context.Background(), &authv1.UpdateRateLimitConfigRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		h = NewTenantSettingGRPCHandler(baseTenant, &mockTenantSettingService{
			updateRateLimitConfigFn: func(int64, map[string]any) (*TenantSettingServiceDataResult, error) {
				return nil, errors.New("db")
			},
		})
		_, err = h.UpdateRateLimitConfig(context.Background(), &authv1.UpdateRateLimitConfigRequest{TenantUuid: tenantUUID.String(), Config: cfgStruct})
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("helpers are nil safe", func(t *testing.T) {
		assert.Nil(t, structMap(nil))
		assert.Empty(t, configProto(nil).AsMap())
		assert.Equal(t, true, structMap(cfgStruct)["enabled"])
	})
}
