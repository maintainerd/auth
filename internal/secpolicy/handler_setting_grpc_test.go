package secpolicy

import (
	"context"
	"errors"
	"testing"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type testSecuritySettingService struct {
	getMFAConfigFn           func(ctx context.Context, userPoolID int64) (map[string]any, error)
	updateMFAConfigFn        func(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	getPasswordConfigFn      func(ctx context.Context, userPoolID int64) (map[string]any, error)
	updatePasswordConfigFn   func(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	getSessionConfigFn       func(ctx context.Context, userPoolID int64) (map[string]any, error)
	updateSessionConfigFn    func(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	getThreatConfigFn        func(ctx context.Context, userPoolID int64) (map[string]any, error)
	updateThreatConfigFn     func(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	getLockoutConfigFn       func(ctx context.Context, userPoolID int64) (map[string]any, error)
	updateLockoutConfigFn    func(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	getRegistrationConfigFn  func(ctx context.Context, userPoolID int64) (map[string]any, error)
	updateRegistrationConfigFn func(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	getTokenConfigFn         func(ctx context.Context, userPoolID int64) (map[string]any, error)
	updateTokenConfigFn      func(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error)
	getByUserPoolIDFn        func(ctx context.Context, userPoolID int64) (*SecuritySettingServiceDataResult, error)
}

func (m *testSecuritySettingService) GetByUserPoolID(ctx context.Context, userPoolID int64) (*SecuritySettingServiceDataResult, error) {
	return m.getByUserPoolIDFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) GetMFAConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	return m.getMFAConfigFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) GetPasswordConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	return m.getPasswordConfigFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) GetSessionConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	return m.getSessionConfigFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) GetThreatConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	return m.getThreatConfigFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) GetLockoutConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	return m.getLockoutConfigFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) GetRegistrationConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	return m.getRegistrationConfigFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) GetTokenConfig(ctx context.Context, userPoolID int64) (map[string]any, error) {
	return m.getTokenConfigFn(ctx, userPoolID)
}
func (m *testSecuritySettingService) UpdateMFAConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return m.updateMFAConfigFn(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)
}
func (m *testSecuritySettingService) UpdatePasswordConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return m.updatePasswordConfigFn(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)
}
func (m *testSecuritySettingService) UpdateSessionConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return m.updateSessionConfigFn(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)
}
func (m *testSecuritySettingService) UpdateThreatConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return m.updateThreatConfigFn(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)
}
func (m *testSecuritySettingService) UpdateLockoutConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return m.updateLockoutConfigFn(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)
}
func (m *testSecuritySettingService) UpdateRegistrationConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return m.updateRegistrationConfigFn(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)
}
func (m *testSecuritySettingService) UpdateTokenConfig(ctx context.Context, userPoolID int64, config map[string]any, updatedBy int64, ipAddress, userAgent string) (*SecuritySettingServiceDataResult, error) {
	return m.updateTokenConfigFn(ctx, userPoolID, config, updatedBy, ipAddress, userAgent)
}

func TestSecuritySettingGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	emptyConfig := map[string]any{}

	svc := &testSecuritySettingService{
		getMFAConfigFn:           func(ctx context.Context, id int64) (map[string]any, error) { return map[string]any{"mfa": "enabled"}, nil },
		updateMFAConfigFn:        func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
		getPasswordConfigFn:      func(ctx context.Context, id int64) (map[string]any, error) { return emptyConfig, nil },
		updatePasswordConfigFn:   func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
		getSessionConfigFn:       func(ctx context.Context, id int64) (map[string]any, error) { return emptyConfig, nil },
		updateSessionConfigFn:    func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
		getThreatConfigFn:        func(ctx context.Context, id int64) (map[string]any, error) { return emptyConfig, nil },
		updateThreatConfigFn:     func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
		getLockoutConfigFn:       func(ctx context.Context, id int64) (map[string]any, error) { return emptyConfig, nil },
		updateLockoutConfigFn:    func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
		getRegistrationConfigFn:  func(ctx context.Context, id int64) (map[string]any, error) { return emptyConfig, nil },
		updateRegistrationConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
		getTokenConfigFn:         func(ctx context.Context, id int64) (map[string]any, error) { return emptyConfig, nil },
		updateTokenConfigFn:      func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
	}

	h := NewSecuritySettingGRPCHandler(svc)

	t.Run("get mfa config", func(t *testing.T) {
		res, err := h.GetMFAConfig(ctx, &authv1.GetMFAConfigRequest{UserPoolId: 1})
		if err != nil { t.Fatal(err) }
		if res.Config == nil { t.Error("expected config") }
	})

	t.Run("update mfa config", func(t *testing.T) {
		res, err := h.UpdateMFAConfig(ctx, &authv1.UpdateMFAConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if err != nil { t.Fatal(err) }
		if res.Config == nil { t.Error("expected config") }
	})

	t.Run("get password config", func(t *testing.T) {
		_, err := h.GetPasswordConfig(ctx, &authv1.GetPasswordConfigRequest{UserPoolId: 1})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update password config", func(t *testing.T) {
		_, err := h.UpdatePasswordConfig(ctx, &authv1.UpdatePasswordConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("get session config", func(t *testing.T) {
		_, err := h.GetSessionConfig(ctx, &authv1.GetSessionConfigRequest{UserPoolId: 1})
		if err != nil { t.Fatal(err) }
	})

	t.Run("get threat config", func(t *testing.T) {
		_, err := h.GetThreatConfig(ctx, &authv1.GetThreatConfigRequest{UserPoolId: 1})
		if err != nil { t.Fatal(err) }
	})

	t.Run("get lockout config", func(t *testing.T) {
		_, err := h.GetLockoutConfig(ctx, &authv1.GetLockoutConfigRequest{UserPoolId: 1})
		if err != nil { t.Fatal(err) }
	})

	t.Run("get registration config", func(t *testing.T) {
		_, err := h.GetRegistrationConfig(ctx, &authv1.GetRegistrationConfigRequest{UserPoolId: 1})
		if err != nil { t.Fatal(err) }
	})

	t.Run("get token config", func(t *testing.T) {
		_, err := h.GetTokenConfig(ctx, &authv1.GetTokenConfigRequest{UserPoolId: 1})
		if err != nil { t.Fatal(err) }
	})

	t.Run("service error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			getMFAConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return nil, errors.New("db error") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.GetMFAConfig(ctx, &authv1.GetMFAConfigRequest{UserPoolId: 1})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			updateMFAConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return nil, errors.New("db error") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.UpdateMFAConfig(ctx, &authv1.UpdateMFAConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update password config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			updatePasswordConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return nil, errors.New("db error") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.UpdatePasswordConfig(ctx, &authv1.UpdatePasswordConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get password config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			getPasswordConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.GetPasswordConfig(ctx, &authv1.GetPasswordConfigRequest{UserPoolId: 1})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get session config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			getSessionConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.GetSessionConfig(ctx, &authv1.GetSessionConfigRequest{UserPoolId: 1})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get threat config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			getThreatConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.GetThreatConfig(ctx, &authv1.GetThreatConfigRequest{UserPoolId: 1})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get lockout config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			getLockoutConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.GetLockoutConfig(ctx, &authv1.GetLockoutConfigRequest{UserPoolId: 1})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get registration config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			getRegistrationConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.GetRegistrationConfig(ctx, &authv1.GetRegistrationConfigRequest{UserPoolId: 1})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("get token config error", func(t *testing.T) {
		errSvc := &testSecuritySettingService{
			getTokenConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(errSvc)
		_, err := h.GetTokenConfig(ctx, &authv1.GetTokenConfigRequest{UserPoolId: 1})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update session config success", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateSessionConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
			getSessionConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return map[string]any{}, nil },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateSessionConfig(ctx, &authv1.UpdateSessionConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update session config error", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateSessionConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateSessionConfig(ctx, &authv1.UpdateSessionConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update threat config success", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateThreatConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
			getThreatConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return map[string]any{}, nil },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateThreatConfig(ctx, &authv1.UpdateThreatConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update threat config error", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateThreatConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateThreatConfig(ctx, &authv1.UpdateThreatConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update lockout config success", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateLockoutConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
			getLockoutConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return map[string]any{}, nil },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateLockoutConfig(ctx, &authv1.UpdateLockoutConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update lockout config error", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateLockoutConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateLockoutConfig(ctx, &authv1.UpdateLockoutConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update registration config success", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateRegistrationConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
			getRegistrationConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return map[string]any{}, nil },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateRegistrationConfig(ctx, &authv1.UpdateRegistrationConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update registration config error", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateRegistrationConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateRegistrationConfig(ctx, &authv1.UpdateRegistrationConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("update token config success", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateTokenConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return &SecuritySettingServiceDataResult{}, nil },
			getTokenConfigFn: func(ctx context.Context, id int64) (map[string]any, error) { return map[string]any{}, nil },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateTokenConfig(ctx, &authv1.UpdateTokenConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if err != nil { t.Fatal(err) }
	})

	t.Run("update token config error", func(t *testing.T) {
		uSvc := &testSecuritySettingService{
			updateTokenConfigFn: func(ctx context.Context, id int64, c map[string]any, ub int64, ip, ua string) (*SecuritySettingServiceDataResult, error) { return nil, errors.New("db") },
		}
		h := NewSecuritySettingGRPCHandler(uSvc)
		_, err := h.UpdateTokenConfig(ctx, &authv1.UpdateTokenConfigRequest{UserPoolId: 1, IpAddress: "1.2.3.4", UserAgent: "test"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})

	t.Run("configProto nil", func(t *testing.T) {
		cfg := configProto(nil)
		if cfg == nil { t.Error("expected non-nil empty struct") }
	})

	t.Run("structMap nil", func(t *testing.T) {
		m := structMap(nil)
		if m != nil { t.Error("expected nil") }
	})

	t.Run("structMap non-nil", func(t *testing.T) {
		s, _ := structpb.NewStruct(map[string]any{"key": "val"})
		m := structMap(s)
		if m == nil { t.Error("expected non-nil") }
	})
}
