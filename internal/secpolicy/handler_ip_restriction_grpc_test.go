package secpolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testIPRestrictionRuleService struct {
	getAllFn       func(ctx context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error)
	getByUUIDFn    func(ctx context.Context, tenantID int64, ruleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
	createFn       func(ctx context.Context, tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IPRestrictionRuleServiceDataResult, error)
	updateFn       func(ctx context.Context, tenantID int64, ruleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error)
	updateStatusFn func(ctx context.Context, tenantID int64, ruleUUID uuid.UUID, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error)
	deleteFn       func(ctx context.Context, tenantID int64, ruleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error)
}

func (m *testIPRestrictionRuleService) GetAll(ctx context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error) {
	return m.getAllFn(ctx, tenantID, ruleType, status, ipAddress, description, page, limit, sortBy, sortOrder)
}
func (m *testIPRestrictionRuleService) GetByUUID(ctx context.Context, tenantID int64, ruleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	return m.getByUUIDFn(ctx, tenantID, ruleUUID)
}
func (m *testIPRestrictionRuleService) Create(ctx context.Context, tenantID int64, description, ruleType, ipAddress, status string, createdBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	return m.createFn(ctx, tenantID, description, ruleType, ipAddress, status, createdBy)
}
func (m *testIPRestrictionRuleService) Update(ctx context.Context, tenantID int64, ruleUUID uuid.UUID, description, ruleType, ipAddress, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	return m.updateFn(ctx, tenantID, ruleUUID, description, ruleType, ipAddress, status, updatedBy)
}
func (m *testIPRestrictionRuleService) UpdateStatus(ctx context.Context, tenantID int64, ruleUUID uuid.UUID, status string, updatedBy int64) (*IPRestrictionRuleServiceDataResult, error) {
	return m.updateStatusFn(ctx, tenantID, ruleUUID, status, updatedBy)
}
func (m *testIPRestrictionRuleService) Delete(ctx context.Context, tenantID int64, ruleUUID uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
	return m.deleteFn(ctx, tenantID, ruleUUID)
}

type testSecpolicyTenantResolver struct {
	getByUUIDFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *testSecpolicyTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(ctx, tenantUUID)
	}
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tenantUUID}, nil
}

func TestIPRestrictionRuleGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	ruleUUID := uuid.New()
	resolver := &testSecpolicyTenantResolver{}
	ruleResult := IPRestrictionRuleServiceDataResult{
		IPRestrictionRuleUUID: ruleUUID, Description: "Block bad IP", Type: "block", IPAddress: "10.0.0.1", Status: "active",
	}

	t.Run("list success", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			getAllFn: func(ctx context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error) {
				return &IPRestrictionRuleServiceListResult{Data: []IPRestrictionRuleServiceDataResult{ruleResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		res, err := h.ListIPRestrictionRules(ctx, &authv1.ListIPRestrictionRulesRequest{TenantUuid: tenantUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Rules) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(res.Rules))
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			getByUUIDFn: func(ctx context.Context, tid int64, id uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
				return &ruleResult, nil
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.GetIPRestrictionRule(ctx, &authv1.GetIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			createFn: func(ctx context.Context, tenantID int64, desc, ruleType, ip, status string, cb int64) (*IPRestrictionRuleServiceDataResult, error) {
				return &ruleResult, nil
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.CreateIPRestrictionRule(ctx, &authv1.CreateIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), Description: "Block bad IP", Type: "block", IpAddress: "10.0.0.1", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			updateFn: func(ctx context.Context, tenantID int64, id uuid.UUID, desc, ruleType, ip, status string, ub int64) (*IPRestrictionRuleServiceDataResult, error) {
				return &ruleResult, nil
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.UpdateIPRestrictionRule(ctx, &authv1.UpdateIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String(), Description: "Block bad IP", Type: "block", IpAddress: "10.0.0.1", Status: "inactive"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("setStatus success", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			updateStatusFn: func(ctx context.Context, tenantID int64, id uuid.UUID, status string, ub int64) (*IPRestrictionRuleServiceDataResult, error) {
				return &ruleResult, nil
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.SetIPRestrictionRuleStatus(ctx, &authv1.SetIPRestrictionRuleStatusRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String(), Status: "inactive"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			deleteFn: func(ctx context.Context, tenantID int64, id uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
				return &ruleResult, nil
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.DeleteIPRestrictionRule(ctx, &authv1.DeleteIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			getAllFn: func(ctx context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error) {
				return &IPRestrictionRuleServiceListResult{}, nil
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.ListIPRestrictionRules(ctx, &authv1.ListIPRestrictionRulesRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
		_, err = h.GetIPRestrictionRule(ctx, &authv1.GetIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service errors", func(t *testing.T) {
		svcErr := errors.New("db error")
		svc := &testIPRestrictionRuleService{
			getAllFn: func(ctx context.Context, tenantID int64, ruleType *string, status []string, ipAddress, description *string, page, limit int, sortBy, sortOrder string) (*IPRestrictionRuleServiceListResult, error) {
				return nil, svcErr
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.ListIPRestrictionRules(ctx, &authv1.ListIPRestrictionRulesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			getByUUIDFn: func(ctx context.Context, tid int64, id uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.GetIPRestrictionRule(ctx, &authv1.GetIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("create service error", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			createFn: func(ctx context.Context, tenantID int64, desc, ruleType, ip, status string, cb int64) (*IPRestrictionRuleServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.CreateIPRestrictionRule(ctx, &authv1.CreateIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), Type: "block", IpAddress: "10.0.0.1"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			updateFn: func(ctx context.Context, tenantID int64, id uuid.UUID, desc, ruleType, ip, rs string, ub int64) (*IPRestrictionRuleServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.UpdateIPRestrictionRule(ctx, &authv1.UpdateIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String(), Type: "block", IpAddress: "10.0.0.1"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("setStatus service error", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			updateStatusFn: func(ctx context.Context, tenantID int64, id uuid.UUID, rs string, ub int64) (*IPRestrictionRuleServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.SetIPRestrictionRuleStatus(ctx, &authv1.SetIPRestrictionRuleStatusRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String(), Status: "active"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("delete service error", func(t *testing.T) {
		svc := &testIPRestrictionRuleService{
			deleteFn: func(ctx context.Context, tenantID int64, id uuid.UUID) (*IPRestrictionRuleServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewIPRestrictionRuleGRPCHandler(resolver, svc)
		_, err := h.DeleteIPRestrictionRule(ctx, &authv1.DeleteIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("resolveTenant resolver error", func(t *testing.T) {
		errResolver := &testSecpolicyTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewIPRestrictionRuleGRPCHandler(errResolver, &testIPRestrictionRuleService{})
		_, err := h.ListIPRestrictionRules(ctx, &authv1.ListIPRestrictionRulesRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("optionalStr non-empty", func(t *testing.T) {
		v := optionalStr("hello")
		if v == nil || *v != "hello" {
			t.Error("expected non-nil")
		}
	})

	t.Run("grpcPagination nil", func(t *testing.T) {
		dto := grpcPagination(nil)
		if dto.Page != 1 || dto.Limit != pagination.DefaultPageSize {
			t.Errorf("expected defaults, got %+v", dto)
		}
	})

	t.Run("ipRuleProto nil", func(t *testing.T) {
		v := ipRuleProto(nil)
		if v != nil {
			t.Error("expected nil")
		}
	})

	t.Run("resolveTenant error get", func(t *testing.T) {
		errResolver := &testSecpolicyTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewIPRestrictionRuleGRPCHandler(errResolver, &testIPRestrictionRuleService{})
		_, err := h.GetIPRestrictionRule(ctx, &authv1.GetIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("resolveTenant error create", func(t *testing.T) {
		errResolver := &testSecpolicyTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewIPRestrictionRuleGRPCHandler(errResolver, &testIPRestrictionRuleService{})
		_, err := h.CreateIPRestrictionRule(ctx, &authv1.CreateIPRestrictionRuleRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("resolveTenant error update", func(t *testing.T) {
		errResolver := &testSecpolicyTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewIPRestrictionRuleGRPCHandler(errResolver, &testIPRestrictionRuleService{})
		_, err := h.UpdateIPRestrictionRule(ctx, &authv1.UpdateIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("resolveTenant error setStatus", func(t *testing.T) {
		errResolver := &testSecpolicyTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewIPRestrictionRuleGRPCHandler(errResolver, &testIPRestrictionRuleService{})
		_, err := h.SetIPRestrictionRuleStatus(ctx, &authv1.SetIPRestrictionRuleStatusRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("resolveTenant error delete", func(t *testing.T) {
		errResolver := &testSecpolicyTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewIPRestrictionRuleGRPCHandler(errResolver, &testIPRestrictionRuleService{})
		_, err := h.DeleteIPRestrictionRule(ctx, &authv1.DeleteIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: ruleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("grpcPagination zero", func(t *testing.T) {
		dto := grpcPagination(&authv1.Pagination{Page: 0, Limit: 0})
		if dto.Page != 1 || dto.Limit != pagination.DefaultPageSize {
			t.Errorf("expected defaults, got %+v", dto)
		}
	})

	t.Run("parseUUID empty", func(t *testing.T) {
		_, err := parseUUID("", "test")
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("invalid rule UUID update", func(t *testing.T) {
		h := NewIPRestrictionRuleGRPCHandler(resolver, &testIPRestrictionRuleService{})
		_, err := h.UpdateIPRestrictionRule(ctx, &authv1.UpdateIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("invalid rule UUID setStatus", func(t *testing.T) {
		h := NewIPRestrictionRuleGRPCHandler(resolver, &testIPRestrictionRuleService{})
		_, err := h.SetIPRestrictionRuleStatus(ctx, &authv1.SetIPRestrictionRuleStatusRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("invalid rule UUID delete", func(t *testing.T) {
		h := NewIPRestrictionRuleGRPCHandler(resolver, &testIPRestrictionRuleService{})
		_, err := h.DeleteIPRestrictionRule(ctx, &authv1.DeleteIPRestrictionRuleRequest{TenantUuid: tenantUUID.String(), IpRestrictionRuleUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})
}
