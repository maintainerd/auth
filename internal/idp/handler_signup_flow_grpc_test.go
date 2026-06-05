package idp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type testSignupFlowService struct {
	getAllFn       func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error)
	getByUUIDFn    func(ctx context.Context, sfUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error)
	createFn       func(ctx context.Context, tenantID int64, name, description string, config map[string]any, status string, clientUUID uuid.UUID) (*SignupFlowServiceDataResult, error)
	updateFn       func(ctx context.Context, sfUUID uuid.UUID, tenantID int64, name, description string, config map[string]any, status string) (*SignupFlowServiceDataResult, error)
	updateStatusFn func(ctx context.Context, sfUUID uuid.UUID, tenantID int64, status string) (*SignupFlowServiceDataResult, error)
	deleteFn       func(ctx context.Context, sfUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error)
	assignRolesFn  func(ctx context.Context, sfUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error)
	getRolesFn     func(ctx context.Context, sfUUID uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error)
	removeRoleFn   func(ctx context.Context, sfUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error
}

func (m *testSignupFlowService) GetAll(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
	return m.getAllFn(ctx, tenantID, name, identifier, status, clientUUID, page, limit, sortBy, sortOrder)
}
func (m *testSignupFlowService) GetByUUID(ctx context.Context, sfUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
	return m.getByUUIDFn(ctx, sfUUID, tenantID)
}
func (m *testSignupFlowService) Create(ctx context.Context, tenantID int64, name, description string, config map[string]any, status string, clientUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
	return m.createFn(ctx, tenantID, name, description, config, status, clientUUID)
}
func (m *testSignupFlowService) Update(ctx context.Context, sfUUID uuid.UUID, tenantID int64, name, description string, config map[string]any, status string) (*SignupFlowServiceDataResult, error) {
	return m.updateFn(ctx, sfUUID, tenantID, name, description, config, status)
}
func (m *testSignupFlowService) UpdateStatus(ctx context.Context, sfUUID uuid.UUID, tenantID int64, status string) (*SignupFlowServiceDataResult, error) {
	return m.updateStatusFn(ctx, sfUUID, tenantID, status)
}
func (m *testSignupFlowService) Delete(ctx context.Context, sfUUID uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
	return m.deleteFn(ctx, sfUUID, tenantID)
}
func (m *testSignupFlowService) AssignRoles(ctx context.Context, sfUUID uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error) {
	return m.assignRolesFn(ctx, sfUUID, tenantID, roleUUIDs)
}
func (m *testSignupFlowService) GetRoles(ctx context.Context, sfUUID uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error) {
	return m.getRolesFn(ctx, sfUUID, tenantID, page, limit)
}
func (m *testSignupFlowService) RemoveRole(ctx context.Context, sfUUID uuid.UUID, tenantID int64, roleUUID uuid.UUID) error {
	return m.removeRoleFn(ctx, sfUUID, tenantID, roleUUID)
}

func TestSignupFlowGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	sfUUID := uuid.New()
	clientUUID := uuid.New()
	roleUUID := uuid.New()
	now := time.Now()
	resolver := &mockIDPTenantResolver{getByUUIDFn: func(ctx context.Context, id uuid.UUID) (*TenantServiceDataResult, error) {
		return &TenantServiceDataResult{TenantID: 1, TenantUUID: id}, nil
	}}
	sfResult := SignupFlowServiceDataResult{
		SignupFlowUUID: sfUUID,
		Name:           "default-signup",
		Description:    "Default signup flow",
		Identifier:     "default-signup-123",
		Status:         "active",
		ClientUUID:     clientUUID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	sfRole := SignupFlowRoleServiceDataResult{
		SignupFlowRoleUUID: uuid.New(),
		RoleUUID:           roleUUID,
		RoleName:           "admin",
		RoleDescription:    "Admin role",
		RoleStatus:         "active",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	t.Run("list success", func(t *testing.T) {
		svc := &testSignupFlowService{
			getAllFn: func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
				return &SignupFlowServiceListResult{Data: []SignupFlowServiceDataResult{sfResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: tenantUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.SignupFlows) != 1 {
			t.Fatalf("expected 1 flow, got %d", len(res.SignupFlows))
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testSignupFlowService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
				return &sfResult, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.GetSignupFlow(ctx, &authv1.GetSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.SignupFlow.Name != "default-signup" {
			t.Errorf("expected name, got %s", res.SignupFlow.Name)
		}
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testSignupFlowService{
			createFn: func(ctx context.Context, tenantID int64, name, description string, config map[string]any, status string, id uuid.UUID) (*SignupFlowServiceDataResult, error) {
				return &sfResult, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.CreateSignupFlow(ctx, &authv1.CreateSignupFlowRequest{TenantUuid: tenantUUID.String(), Name: "default-signup", Description: "Default signup flow", Status: "active", ClientUuid: clientUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.SignupFlow.Status != "active" {
			t.Errorf("expected status active, got %s", res.SignupFlow.Status)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testSignupFlowService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, description string, config map[string]any, status string) (*SignupFlowServiceDataResult, error) {
				return &sfResult, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.UpdateSignupFlow(ctx, &authv1.UpdateSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Name: "default-signup", Description: "Default", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
		if res.SignupFlow.Name != "default-signup" {
			t.Errorf("expected name, got %s", res.SignupFlow.Name)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testSignupFlowService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
				return &sfResult, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.DeleteSignupFlow(ctx, &authv1.DeleteSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.SignupFlow.Name != "default-signup" {
			t.Errorf("expected name, got %s", res.SignupFlow.Name)
		}
	})

	t.Run("setStatus success", func(t *testing.T) {
		svc := &testSignupFlowService{
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*SignupFlowServiceDataResult, error) {
				return &sfResult, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.SetSignupFlowStatus(ctx, &authv1.SetSignupFlowStatusRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Status: "inactive"})
		if err != nil {
			t.Fatal(err)
		}
		if res.SignupFlow.Name != "default-signup" {
			t.Errorf("expected name, got %s", res.SignupFlow.Name)
		}
	})

	t.Run("assignRoles success", func(t *testing.T) {
		svc := &testSignupFlowService{
			assignRolesFn: func(ctx context.Context, id uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error) {
				return []SignupFlowRoleServiceDataResult{sfRole}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.AssignSignupFlowRoles(ctx, &authv1.AssignSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuids: []string{roleUUID.String()}})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Roles) != 1 {
			t.Fatalf("expected 1 role, got %d", len(res.Roles))
		}
	})

	t.Run("listRoles success", func(t *testing.T) {
		svc := &testSignupFlowService{
			getRolesFn: func(ctx context.Context, id uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error) {
				return &SignupFlowRoleServiceListResult{Data: []SignupFlowRoleServiceDataResult{sfRole}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.ListSignupFlowRoles(ctx, &authv1.ListSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Roles) != 1 {
			t.Fatalf("expected 1 role, got %d", len(res.Roles))
		}
	})

	t.Run("removeRole success", func(t *testing.T) {
		svc := &testSignupFlowService{
			removeRoleFn: func(ctx context.Context, id uuid.UUID, tenantID int64, roleID uuid.UUID) error {
				return nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.RemoveSignupFlowRole(ctx, &authv1.RemoveSignupFlowRoleRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuid: roleUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if !res.Removed {
			t.Error("expected removed true")
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		svc := &testSignupFlowService{
			getAllFn: func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
				return &SignupFlowServiceListResult{}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
		_, err = h.GetSignupFlow(ctx, &authv1.GetSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service errors", func(t *testing.T) {
		svcErr := errors.New("db error")
		svc := &testSignupFlowService{
			getAllFn: func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
				return nil, svcErr
			},
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
				return nil, svcErr
			},
			createFn: func(ctx context.Context, tenantID int64, name, description string, config map[string]any, status string, clientUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
				return nil, svcErr
			},
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, description string, config map[string]any, status string) (*SignupFlowServiceDataResult, error) {
				return nil, svcErr
			},
			updateStatusFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*SignupFlowServiceDataResult, error) {
				return nil, svcErr
			},
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*SignupFlowServiceDataResult, error) {
				return nil, svcErr
			},
			assignRolesFn: func(ctx context.Context, id uuid.UUID, tenantID int64, roleUUIDs []uuid.UUID) ([]SignupFlowRoleServiceDataResult, error) {
				return nil, svcErr
			},
			getRolesFn: func(ctx context.Context, id uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error) {
				return nil, svcErr
			},
			removeRoleFn: func(ctx context.Context, id uuid.UUID, tenantID int64, roleID uuid.UUID) error {
				return svcErr
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.GetSignupFlow(ctx, &authv1.GetSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.CreateSignupFlow(ctx, &authv1.CreateSignupFlowRequest{TenantUuid: tenantUUID.String(), Name: "test", Description: "desc", Status: "active", ClientUuid: clientUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.UpdateSignupFlow(ctx, &authv1.UpdateSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Name: "test", Description: "desc", Status: "active"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.SetSignupFlowStatus(ctx, &authv1.SetSignupFlowStatusRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.DeleteSignupFlow(ctx, &authv1.DeleteSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.AssignSignupFlowRoles(ctx, &authv1.AssignSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuids: []string{roleUUID.String()}})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.ListSignupFlowRoles(ctx, &authv1.ListSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.RemoveSignupFlowRole(ctx, &authv1.RemoveSignupFlowRoleRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuid: roleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("list with client uuid parse error", func(t *testing.T) {
		svc := &testSignupFlowService{
			getAllFn: func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
				return &SignupFlowServiceListResult{}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: tenantUUID.String(), ClientUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("list with valid client uuid filter", func(t *testing.T) {
		svc := &testSignupFlowService{
			getAllFn: func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
				if clientUUID == nil {
					t.Error("expected clientUUID passed through")
				}
				return &SignupFlowServiceListResult{Data: []SignupFlowServiceDataResult{sfResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.SignupFlows) != 1 {
			t.Fatalf("expected 1 flow, got %d", len(res.SignupFlows))
		}
	})

	t.Run("list with validation error", func(t *testing.T) {
		svc := &testSignupFlowService{
			getAllFn: func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
				return &SignupFlowServiceListResult{}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: tenantUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10, SortOrder: "invalid"}})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("list with nil pagination", func(t *testing.T) {
		svc := &testSignupFlowService{
			getAllFn: func(ctx context.Context, tenantID int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*SignupFlowServiceListResult, error) {
				return &SignupFlowServiceListResult{Data: []SignupFlowServiceDataResult{sfResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: tenantUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.SignupFlows) != 1 {
			t.Fatalf("expected 1 flow, got %d", len(res.SignupFlows))
		}
	})

	t.Run("listRoles with validation error", func(t *testing.T) {
		svc := &testSignupFlowService{
			getRolesFn: func(ctx context.Context, sfUUID uuid.UUID, tenantID int64, page, limit int) (*SignupFlowRoleServiceListResult, error) {
				return &SignupFlowRoleServiceListResult{}, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.ListSignupFlowRoles(ctx, &authv1.ListSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10, SortOrder: "invalid"}})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("get with bad sf uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.GetSignupFlow(ctx, &authv1.GetSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("create with bad client uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.CreateSignupFlow(ctx, &authv1.CreateSignupFlowRequest{TenantUuid: tenantUUID.String(), Name: "test", Description: "desc", Status: "active", ClientUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("create with config", func(t *testing.T) {
		cfg, _ := structpb.NewStruct(map[string]any{"key": "value"})
		svc := &testSignupFlowService{
			createFn: func(ctx context.Context, tenantID int64, name, description string, config map[string]any, status string, clientUUID uuid.UUID) (*SignupFlowServiceDataResult, error) {
				if config == nil || config["key"] != "value" {
					t.Error("expected config with key=value")
				}
				return &sfResult, nil
			},
		}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		res, err := h.CreateSignupFlow(ctx, &authv1.CreateSignupFlowRequest{TenantUuid: tenantUUID.String(), Name: "default-signup", Description: "Default signup flow", Status: "active", ClientUuid: clientUUID.String(), Config: cfg})
		if err != nil {
			t.Fatal(err)
		}
		if res.SignupFlow.Status != "active" {
			t.Errorf("expected status active, got %s", res.SignupFlow.Status)
		}
	})

	t.Run("update with bad sf uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.UpdateSignupFlow(ctx, &authv1.UpdateSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad", Name: "test", Description: "desc", Status: "active"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("setStatus with bad sf uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.SetSignupFlowStatus(ctx, &authv1.SetSignupFlowStatusRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad", Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("delete with bad sf uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.DeleteSignupFlow(ctx, &authv1.DeleteSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("assignRoles with bad sf uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.AssignSignupFlowRoles(ctx, &authv1.AssignSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad", RoleUuids: []string{roleUUID.String()}})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("assignRoles with bad role uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.AssignSignupFlowRoles(ctx, &authv1.AssignSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuids: []string{"bad"}})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("listRoles with bad sf uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.ListSignupFlowRoles(ctx, &authv1.ListSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad", Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("removeRole with bad sf uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.RemoveSignupFlowRole(ctx, &authv1.RemoveSignupFlowRoleRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: "bad", RoleUuid: roleUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("removeRole with bad role uuid", func(t *testing.T) {
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(resolver, svc)
		_, err := h.RemoveSignupFlowRole(ctx, &authv1.RemoveSignupFlowRoleRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("tenant not found", func(t *testing.T) {
		badResolver := &mockIDPTenantResolver{getByUUIDFn: func(ctx context.Context, id uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("tenant not found")
		}}
		svc := &testSignupFlowService{}
		h := NewSignupFlowGRPCHandler(badResolver, svc)
		_, err := h.ListSignupFlows(ctx, &authv1.ListSignupFlowsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.GetSignupFlow(ctx, &authv1.GetSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String()})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.CreateSignupFlow(ctx, &authv1.CreateSignupFlowRequest{TenantUuid: tenantUUID.String(), Name: "test", Description: "desc", Status: "active", ClientUuid: clientUUID.String()})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.UpdateSignupFlow(ctx, &authv1.UpdateSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Name: "test", Description: "desc", Status: "active"})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.SetSignupFlowStatus(ctx, &authv1.SetSignupFlowStatusRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.DeleteSignupFlow(ctx, &authv1.DeleteSignupFlowRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String()})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.AssignSignupFlowRoles(ctx, &authv1.AssignSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuids: []string{roleUUID.String()}})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.ListSignupFlowRoles(ctx, &authv1.ListSignupFlowRolesRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
		_, err = h.RemoveSignupFlowRole(ctx, &authv1.RemoveSignupFlowRoleRequest{TenantUuid: tenantUUID.String(), SignupFlowUuid: sfUUID.String(), RoleUuid: roleUUID.String()})
		if code := status.Code(err); code != codes.NotFound {
			t.Errorf("expected NotFound, got %v", code)
		}
	})
}

func TestSignupFlowProtoConverters(t *testing.T) {
	assert := require.New(t)
	assert.Nil(toSignupFlowProto(nil))
	assert.Nil(toSignupFlowRoleProto(nil))
	sf := &SignupFlowServiceDataResult{
		SignupFlowUUID: uuid.New(),
		Name:           "test",
		Status:         "active",
		ClientUUID:     uuid.New(),
	}
	proto := toSignupFlowProto(sf)
	assert.Equal("test", proto.Name)
	assert.Equal("active", proto.Status)
}

func TestSignupFlowHelperFunctions(t *testing.T) {
	t.Run("toSignupFlowProto nil", func(t *testing.T) {
		if toSignupFlowProto(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("toSignupFlowProto with config", func(t *testing.T) {
		now := time.Now()
		sfUUID := uuid.New()
		clientUUID := uuid.New()
		sf := &SignupFlowServiceDataResult{
			SignupFlowUUID: sfUUID,
			Name:           "test",
			Description:    "desc",
			Config:         map[string]any{"key": "value"},
			Status:         "active",
			ClientUUID:     clientUUID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		proto := toSignupFlowProto(sf)
		if proto == nil {
			t.Fatal("expected non-nil proto")
		}
		if proto.Config == nil {
			t.Error("expected non-nil config")
		}
	})

	t.Run("toSignupFlowProto with nil config", func(t *testing.T) {
		now := time.Now()
		sfUUID := uuid.New()
		clientUUID := uuid.New()
		sf := &SignupFlowServiceDataResult{
			SignupFlowUUID: sfUUID,
			Name:           "test",
			Config:         nil,
			Status:         "active",
			ClientUUID:     clientUUID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		proto := toSignupFlowProto(sf)
		if proto == nil {
			t.Fatal("expected non-nil proto")
		}
		if proto.Config != nil {
			t.Error("expected nil config")
		}
	})

	t.Run("toSignupFlowRoleProto nil", func(t *testing.T) {
		if toSignupFlowRoleProto(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("toSignupFlowRoleProto with data", func(t *testing.T) {
		now := time.Now()
		r := &SignupFlowRoleServiceDataResult{
			SignupFlowRoleUUID: uuid.New(),
			RoleUUID:           uuid.New(),
			RoleName:           "admin",
			RoleDescription:    "Admin role",
			RoleStatus:         "active",
			RoleIsDefault:      true,
			RoleIsSystem:       false,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		proto := toSignupFlowRoleProto(r)
		if proto == nil {
			t.Fatal("expected non-nil proto")
		}
		if proto.RoleName != "admin" {
			t.Errorf("expected RoleName admin, got %s", proto.RoleName)
		}
	})

	t.Run("structpbToMap nil", func(t *testing.T) {
		if structpbToMap(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("structpbToMap with data", func(t *testing.T) {
		s, _ := structpb.NewStruct(map[string]any{"key": "value"})
		m := structpbToMap(s)
		if m == nil {
			t.Fatal("expected non-nil map")
		}
		if m["key"] != "value" {
			t.Error("expected key=value")
		}
	})

	t.Run("mapToStructpb nil", func(t *testing.T) {
		if mapToStructpb(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("mapToStructpb with data", func(t *testing.T) {
		s := mapToStructpb(map[string]any{"key": "value"})
		if s == nil {
			t.Fatal("expected non-nil struct")
		}
		if s.Fields["key"].GetStringValue() != "value" {
			t.Error("expected key=value")
		}
	})
}
