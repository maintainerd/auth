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

type testNotifierTenant struct {
	getFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *testNotifierTenant) GetByUUID(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getFn != nil {
		return m.getFn(ctx, tuuid)
	}
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tuuid}, nil
}

type testEmailConfigService struct {
	getFn    func(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error)
	updateFn func(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption, logoURL string, testMode *bool) (*EmailConfigServiceDataResult, error)
}

func (m *testEmailConfigService) Get(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error) {
	return m.getFn(ctx, tenantID)
}
func (m *testEmailConfigService) GetStatus(ctx context.Context, tenantID int64) (*ConfigStatusResult, error) {
	return &ConfigStatusResult{}, nil
}
func (m *testEmailConfigService) Update(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption, logoURL string, testMode *bool) (*EmailConfigServiceDataResult, error) {
	return m.updateFn(ctx, tenantID, provider, host, port, username, password, fromAddress, fromName, replyTo, encryption, logoURL, testMode)
}

func TestEmailConfigGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	resolver := &testNotifierTenant{}

	t.Run("get success", func(t *testing.T) {
		svc := &testEmailConfigService{
			getFn: func(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error) {
				return &EmailConfigServiceDataResult{EmailConfigUUID: uuid.New(), Provider: "smtp"}, nil
			},
		}
		h := NewEmailConfigGRPCHandler(resolver, svc)
		res, err := h.GetEmailConfig(ctx, &authv1.GetEmailConfigRequest{TenantUuid: tenantUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Config.Provider != "smtp" {
			t.Errorf("expected smtp, got %s", res.Config.Provider)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testEmailConfigService{
			updateFn: func(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption, logoURL string, testMode *bool) (*EmailConfigServiceDataResult, error) {
				return &EmailConfigServiceDataResult{EmailConfigUUID: uuid.New(), Provider: "smtp"}, nil
			},
		}
		h := NewEmailConfigGRPCHandler(resolver, svc)
		_, err := h.UpdateEmailConfig(ctx, &authv1.UpdateEmailConfigRequest{TenantUuid: tenantUUID.String(), Provider: "smtp", Host: "localhost", Port: 587})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("service error", func(t *testing.T) {
		svc := &testEmailConfigService{
			getFn: func(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewEmailConfigGRPCHandler(resolver, svc)
		_, err := h.GetEmailConfig(ctx, &authv1.GetEmailConfigRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("tenant error invalid UUID", func(t *testing.T) {
		svc := &testEmailConfigService{}
		h := NewEmailConfigGRPCHandler(resolver, svc)
		_, err := h.GetEmailConfig(ctx, &authv1.GetEmailConfigRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("tenant error resolver", func(t *testing.T) {
		errResolver := &testNotifierTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		svc := &testEmailConfigService{}
		h := NewEmailConfigGRPCHandler(errResolver, svc)
		_, err := h.GetEmailConfig(ctx, &authv1.GetEmailConfigRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testEmailConfigService{
			updateFn: func(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption, logoURL string, testMode *bool) (*EmailConfigServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewEmailConfigGRPCHandler(resolver, svc)
		_, err := h.UpdateEmailConfig(ctx, &authv1.UpdateEmailConfigRequest{TenantUuid: tenantUUID.String(), Provider: "smtp"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update tenant resolver error", func(t *testing.T) {
		errResolver := &testNotifierTenant{getFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		svc := &testEmailConfigService{}
		h := NewEmailConfigGRPCHandler(errResolver, svc)
		_, err := h.UpdateEmailConfig(ctx, &authv1.UpdateEmailConfigRequest{TenantUuid: tenantUUID.String(), Provider: "smtp"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})
}
