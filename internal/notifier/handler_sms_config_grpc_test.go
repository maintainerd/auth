package notifier

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testSMSConfigService struct {
	getFn    func(ctx context.Context, tenantID int64) (*SMSConfigServiceDataResult, error)
	updateFn func(ctx context.Context, tenantID int64, provider, accountSID, authToken, fromNumber, senderID string, testMode *bool) (*SMSConfigServiceDataResult, error)
}

func (m *testSMSConfigService) Get(ctx context.Context, tenantID int64) (*SMSConfigServiceDataResult, error) {
	return m.getFn(ctx, tenantID)
}
func (m *testSMSConfigService) Update(ctx context.Context, tenantID int64, provider, accountSID, authToken, fromNumber, senderID string, testMode *bool) (*SMSConfigServiceDataResult, error) {
	return m.updateFn(ctx, tenantID, provider, accountSID, authToken, fromNumber, senderID, testMode)
}

func TestSMSConfigGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	resolver := &testNotifierTenant{}

	t.Run("get success", func(t *testing.T) {
		svc := &testSMSConfigService{
			getFn: func(ctx context.Context, tenantID int64) (*SMSConfigServiceDataResult, error) {
				return &SMSConfigServiceDataResult{SMSConfigUUID: uuid.New(), Provider: "twilio"}, nil
			},
		}
		h := NewSMSConfigGRPCHandler(resolver, svc)
		res, err := h.GetSMSConfig(ctx, &authv1.GetSMSConfigRequest{TenantUuid: tenantUUID.String()})
		if err != nil { t.Fatal(err) }
		if res.Config.Provider != "twilio" { t.Errorf("expected twilio") }
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testSMSConfigService{
			updateFn: func(ctx context.Context, tenantID int64, provider, accountSID, authToken, fromNumber, senderID string, testMode *bool) (*SMSConfigServiceDataResult, error) {
				return &SMSConfigServiceDataResult{SMSConfigUUID: uuid.New(), Provider: "twilio"}, nil
			},
		}
		h := NewSMSConfigGRPCHandler(resolver, svc)
		_, err := h.UpdateSMSConfig(ctx, &authv1.UpdateSMSConfigRequest{TenantUuid: tenantUUID.String(), Provider: "twilio", AccountSid: "sid"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testSMSConfigService{getFn: func(ctx context.Context, tenantID int64) (*SMSConfigServiceDataResult, error) { return nil, errors.New("db") }}
		h := NewSMSConfigGRPCHandler(resolver, svc)
		_, err := h.GetSMSConfig(ctx, &authv1.GetSMSConfigRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("tenant error invalid UUID", func(t *testing.T) {
		svc := &testSMSConfigService{}
		h := NewSMSConfigGRPCHandler(resolver, svc)
		_, err := h.GetSMSConfig(ctx, &authv1.GetSMSConfigRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument { t.Errorf("expected InvalidArgument, got %v", code) }
	})

	t.Run("tenant error resolver", func(t *testing.T) {
		errResolver := &testNotifierTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		svc := &testSMSConfigService{}
		h := NewSMSConfigGRPCHandler(errResolver, svc)
		_, err := h.GetSMSConfig(ctx, &authv1.GetSMSConfigRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testSMSConfigService{
			updateFn: func(ctx context.Context, tenantID int64, provider, accountSID, authToken, fromNumber, senderID string, testMode *bool) (*SMSConfigServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewSMSConfigGRPCHandler(resolver, svc)
		_, err := h.UpdateSMSConfig(ctx, &authv1.UpdateSMSConfigRequest{TenantUuid: tenantUUID.String(), Provider: "twilio"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update tenant resolver error", func(t *testing.T) {
		errResolver := &testNotifierTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("tenant") }}
		svc := &testSMSConfigService{}
		h := NewSMSConfigGRPCHandler(errResolver, svc)
		_, err := h.UpdateSMSConfig(ctx, &authv1.UpdateSMSConfigRequest{TenantUuid: tenantUUID.String(), Provider: "twilio"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}
