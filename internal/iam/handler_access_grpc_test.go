package iam

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/shared"
	tenantpkg "github.com/maintainerd/auth/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

func TestPermissionGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	tenantUUID := uuid.New()
	permissionUUID := uuid.New()
	apiUUID := uuid.New()
	tenantResolver := accessTenantResolver(t, tenantUUID)
	permission := PermissionServiceDataResult{
		PermissionUUID: permissionUUID,
		Name:           "read:users",
		Description:    "Read users permission",
		Status:         shared.StatusActive,
		API:            &APIServiceDataResult{APIUUID: apiUUID, Name: "users", APIType: shared.APITypeRest},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	t.Run("success", func(t *testing.T) {
		svc := &mockPermissionService{
			getFn: func(f PermissionServiceGetFilter) (*PermissionServiceGetResult, error) {
				assert.Equal(t, int64(77), f.TenantID)
				assert.Equal(t, "read:users", *f.Name)
				assert.Equal(t, shared.StatusActive, *f.Status)
				return &PermissionServiceGetResult{Data: []PermissionServiceDataResult{permission}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			getByUUIDFn: func(id uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
				assert.Equal(t, permissionUUID, id)
				assert.Equal(t, int64(77), tenantID)
				return &permission, nil
			},
			createFn: func(tenantID int64, name string, description string, permStatus string, isSystem bool, apiID string) (*PermissionServiceDataResult, error) {
				assert.Equal(t, int64(77), tenantID)
				assert.Equal(t, apiUUID.String(), apiID)
				assert.False(t, isSystem)
				return &permission, nil
			},
			updateFn: func(id uuid.UUID, tenantID int64, name string, description string, permStatus string) (*PermissionServiceDataResult, error) {
				assert.Equal(t, permissionUUID, id)
				return &permission, nil
			},
			setStatusFn: func(id uuid.UUID, tenantID int64, permStatus string) (*PermissionServiceDataResult, error) {
				assert.Equal(t, shared.StatusInactive, permStatus)
				updated := permission
				updated.Status = permStatus
				return &updated, nil
			},
			deleteByUUIDFn: func(id uuid.UUID, tenantID int64) (*PermissionServiceDataResult, error) {
				assert.Equal(t, permissionUUID, id)
				return &permission, nil
			},
		}
		h := NewPermissionGRPCHandler(tenantResolver, svc)

		list, err := h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantUuid: tenantUUID.String(), Name: "read:users", Status: shared.StatusActive, Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "users", list.Permissions[0].Api.Name)
		got, err := h.GetPermission(ctx, &authv1.GetPermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "read:users", got.Permission.Name)
		created, err := h.CreatePermission(ctx, validCreatePermissionRequest(tenantUUID, apiUUID))
		require.NoError(t, err)
		assert.Equal(t, permissionUUID.String(), created.Permission.PermissionUuid)
		updated, err := h.UpdatePermission(ctx, validUpdatePermissionRequest(tenantUUID, permissionUUID))
		require.NoError(t, err)
		assert.Equal(t, "read:users", updated.Permission.Name)
		statusRes, err := h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String(), Status: shared.StatusInactive})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, statusRes.Permission.Status)
		deleted, err := h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, permissionUUID.String(), deleted.Permission.PermissionUuid)
	})

	t.Run("validation and service errors", func(t *testing.T) {
		h := NewPermissionGRPCHandler(mockTenantResolver{}, &mockPermissionService{})
		_, err := h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantUuid: tenantUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetPermission(ctx, &authv1.GetPermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreatePermission(ctx, &authv1.CreatePermissionRequest{TenantUuid: tenantUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreate := validCreatePermissionRequest(tenantUUID, apiUUID)
		badTenantCreate.TenantUuid = "bad"
		_, err = h.CreatePermission(ctx, badTenantCreate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdatePermission(ctx, &authv1.UpdatePermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantUpdate := validUpdatePermissionRequest(tenantUUID, permissionUUID)
		badTenantUpdate.TenantUuid = "bad"
		_, err = h.UpdatePermission(ctx, badTenantUpdate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantUuid: "bad", PermissionUuid: permissionUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantUuid: "bad", PermissionUuid: permissionUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		h = NewPermissionGRPCHandler(mockTenantResolver{getByUUIDFn: func(uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing tenant")
		}}, &mockPermissionService{})
		_, err = h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))

		serviceErr := errors.New("db")
		h = NewPermissionGRPCHandler(tenantResolver, &mockPermissionService{
			getFn:       func(PermissionServiceGetFilter) (*PermissionServiceGetResult, error) { return nil, serviceErr },
			getByUUIDFn: func(uuid.UUID, int64) (*PermissionServiceDataResult, error) { return nil, serviceErr },
			createFn: func(int64, string, string, string, bool, string) (*PermissionServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, string, string) (*PermissionServiceDataResult, error) {
				return nil, serviceErr
			},
			setStatusFn:    func(uuid.UUID, int64, string) (*PermissionServiceDataResult, error) { return nil, serviceErr },
			deleteByUUIDFn: func(uuid.UUID, int64) (*PermissionServiceDataResult, error) { return nil, serviceErr },
		})
		_, err = h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetPermission(ctx, &authv1.GetPermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreatePermission(ctx, validCreatePermissionRequest(tenantUUID, apiUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdatePermission(ctx, validUpdatePermissionRequest(tenantUUID, permissionUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestPolicyGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	tenantUUID := uuid.New()
	policyUUID := uuid.New()
	serviceUUID := uuid.New()
	document := validPolicyDocumentStruct(t)
	documentJSON := datatypes.JSON(`{"statement":[{"action":["user:read"],"effect":"allow","resource":["user:*"]}],"version":"v1"}`)
	description := "Read users"
	policy := PolicyServiceDataResult{PolicyUUID: policyUUID, Name: "user:read", Description: &description, Document: documentJSON, Version: "v1", Status: shared.StatusActive, CreatedAt: now, UpdatedAt: now}
	service := PolicyServiceServiceDataResult{ServiceUUID: serviceUUID, Name: "auth", DisplayName: "Auth Service", Description: "Authentication service", Version: "v1", Status: shared.StatusActive}
	tenantResolver := accessTenantResolver(t, tenantUUID)

	t.Run("success", func(t *testing.T) {
		svc := &mockPolicyService{
			getFn: func(f PolicyServiceGetFilter) (*PolicyServiceGetResult, error) {
				assert.Equal(t, int64(77), f.TenantID)
				assert.Equal(t, serviceUUID, *f.ServiceID)
				return &PolicyServiceGetResult{Data: []PolicyServiceDataResult{policy}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			getByUUIDFn: func(id uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
				assert.Equal(t, policyUUID, id)
				return &policy, nil
			},
			getServicesByPolicyUUIDFn: func(id uuid.UUID, tenantID int64, f PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error) {
				assert.Equal(t, policyUUID, id)
				return &PolicyServiceServicesResult{Data: []PolicyServiceServiceDataResult{service}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			createFn: func(tenantID int64, name string, desc *string, doc datatypes.JSON, version string, policyStatus string, isSystem bool) (*PolicyServiceDataResult, error) {
				assert.JSONEq(t, string(documentJSON), string(doc))
				assert.False(t, isSystem)
				return &policy, nil
			},
			updateFn: func(id uuid.UUID, tenantID int64, name string, desc *string, doc datatypes.JSON, version string, policyStatus string) (*PolicyServiceDataResult, error) {
				assert.Equal(t, policyUUID, id)
				return &policy, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, tenantID int64, policyStatus string) (*PolicyServiceDataResult, error) {
				updated := policy
				updated.Status = policyStatus
				return &updated, nil
			},
			deleteByUUIDFn: func(id uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
				return &policy, nil
			},
		}
		h := NewPolicyGRPCHandler(tenantResolver, svc)

		list, err := h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), Status: []string{shared.StatusActive}, Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "user:read", list.Policies[0].Name)
		got, err := h.GetPolicy(ctx, &authv1.GetPolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "v1", got.Policy.Document.AsMap()["version"])
		services, err := h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "auth", services.Services[0].Name)
		created, err := h.CreatePolicy(ctx, validCreatePolicyRequest(tenantUUID, document))
		require.NoError(t, err)
		assert.Equal(t, "user:read", created.Policy.Name)
		updated, err := h.UpdatePolicy(ctx, validUpdatePolicyRequest(tenantUUID, policyUUID, document))
		require.NoError(t, err)
		assert.Equal(t, policyUUID.String(), updated.Policy.PolicyUuid)
		statusRes, err := h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Status: shared.StatusInactive})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, statusRes.Policy.Status)
		deleted, err := h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, policyUUID.String(), deleted.Policy.PolicyUuid)
	})

	t.Run("validation and service errors", func(t *testing.T) {
		h := NewPolicyGRPCHandler(mockTenantResolver{}, &mockPolicyService{})
		_, err := h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String(), ServiceUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String(), Status: []string{"bad"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetPolicy(ctx, &authv1.GetPolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Name: string(make([]byte, 151))})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreatePolicy(ctx, &authv1.CreatePolicyRequest{TenantUuid: tenantUUID.String(), Name: "BAD", Document: document})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreatePolicy(ctx, &authv1.CreatePolicyRequest{TenantUuid: tenantUUID.String(), Name: "user:read"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreate := validCreatePolicyRequest(tenantUUID, document)
		badTenantCreate.TenantUuid = "bad"
		_, err = h.CreatePolicy(ctx, badTenantCreate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdatePolicy(ctx, &authv1.UpdatePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Name: "BAD", Document: document})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdatePolicy(ctx, &authv1.UpdatePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad", Name: "user:read", Document: document})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantUpdate := validUpdatePolicyRequest(tenantUUID, policyUUID, document)
		badTenantUpdate.TenantUuid = "bad"
		_, err = h.UpdatePolicy(ctx, badTenantUpdate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		nilDocumentUpdate := validUpdatePolicyRequest(tenantUUID, policyUUID, document)
		nilDocumentUpdate.Document = nil
		_, err = h.UpdatePolicy(ctx, nilDocumentUpdate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: "bad", PolicyUuid: policyUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: "bad", PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		serviceErr := errors.New("db")
		h = NewPolicyGRPCHandler(tenantResolver, &mockPolicyService{
			getFn:       func(PolicyServiceGetFilter) (*PolicyServiceGetResult, error) { return nil, serviceErr },
			getByUUIDFn: func(uuid.UUID, int64) (*PolicyServiceDataResult, error) { return nil, serviceErr },
			getServicesByPolicyUUIDFn: func(uuid.UUID, int64, PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error) {
				return nil, serviceErr
			},
			createFn: func(int64, string, *string, datatypes.JSON, string, string, bool) (*PolicyServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, *string, datatypes.JSON, string, string) (*PolicyServiceDataResult, error) {
				return nil, serviceErr
			},
			setStatusByUUIDFn: func(uuid.UUID, int64, string) (*PolicyServiceDataResult, error) { return nil, serviceErr },
			deleteByUUIDFn:    func(uuid.UUID, int64) (*PolicyServiceDataResult, error) { return nil, serviceErr },
		})
		_, err = h.ListPolicies(ctx, &authv1.ListPoliciesRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetPolicy(ctx, &authv1.GetPolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.ListPolicyServices(ctx, &authv1.ListPolicyServicesRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreatePolicy(ctx, validCreatePolicyRequest(tenantUUID, document))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdatePolicy(ctx, validUpdatePolicyRequest(tenantUUID, policyUUID, document))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetPolicyStatus(ctx, &authv1.SetPolicyStatusRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeletePolicy(ctx, &authv1.DeletePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestRoleGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	tenantUUID := uuid.New()
	roleUUID := uuid.New()
	actorUUID := uuid.New()
	permissionUUID := uuid.New()
	permission := PermissionServiceDataResult{PermissionUUID: permissionUUID, Name: "read:users", Description: "Read users permission", Status: shared.StatusActive}
	role := RoleServiceDataResult{RoleUUID: roleUUID, Name: "admin", Description: "Admin role", Status: shared.StatusActive, Permissions: &[]PermissionServiceDataResult{permission}, CreatedAt: now, UpdatedAt: now}
	tenantResolver := accessTenantResolver(t, tenantUUID)

	t.Run("success", func(t *testing.T) {
		svc := &mockRoleService{
			getFn: func(f RoleServiceGetFilter) (*RoleServiceGetResult, error) {
				assert.Equal(t, int64(77), f.TenantID)
				return &RoleServiceGetResult{Data: []RoleServiceDataResult{role}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			getByUUIDFn: func(id uuid.UUID, tenantID int64) (*RoleServiceDataResult, error) {
				assert.Equal(t, roleUUID, id)
				return &role, nil
			},
			getRolePermissionsFn: func(f RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error) {
				assert.Equal(t, roleUUID, f.RoleUUID)
				return &RoleServiceGetPermissionsResult{Data: []PermissionServiceDataResult{permission}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			createFn: func(name string, description string, isDefault bool, isSystem bool, roleStatus string, tenantID string, actor uuid.UUID) (*RoleServiceDataResult, error) {
				assert.Equal(t, tenantUUID.String(), tenantID)
				assert.Equal(t, actorUUID, actor)
				return &role, nil
			},
			updateFn: func(id uuid.UUID, tenantID int64, name string, description string, isDefault bool, isSystem bool, roleStatus string, actor uuid.UUID) (*RoleServiceDataResult, error) {
				assert.Equal(t, roleUUID, id)
				return &role, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, tenantID int64, roleStatus string, actor uuid.UUID) (*RoleServiceDataResult, error) {
				updated := role
				updated.Status = roleStatus
				return &updated, nil
			},
			deleteByUUIDFn: func(id uuid.UUID, tenantID int64, actor uuid.UUID) (*RoleServiceDataResult, error) {
				return &role, nil
			},
			addRolePermsFn: func(id uuid.UUID, tenantID int64, permissionIDs []uuid.UUID, actor uuid.UUID) (*RoleServiceDataResult, error) {
				assert.Equal(t, []uuid.UUID{permissionUUID}, permissionIDs)
				return &role, nil
			},
			removeRolePermsFn: func(id uuid.UUID, tenantID int64, permissionID uuid.UUID, actor uuid.UUID) (*RoleServiceDataResult, error) {
				assert.Equal(t, permissionUUID, permissionID)
				return &role, nil
			},
		}
		h := NewRoleGRPCHandler(tenantResolver, svc)

		list, err := h.ListRoles(ctx, &authv1.ListRolesRequest{TenantUuid: tenantUUID.String(), Status: shared.StatusActive, Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "admin", list.Roles[0].Name)
		got, err := h.GetRole(ctx, &authv1.GetRoleRequest{TenantUuid: tenantUUID.String(), RoleUuid: roleUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "read:users", got.Role.Permissions[0].Name)
		created, err := h.CreateRole(ctx, validCreateRoleRequest(tenantUUID, actorUUID))
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), created.Role.RoleUuid)
		updated, err := h.UpdateRole(ctx, validUpdateRoleRequest(tenantUUID, actorUUID, roleUUID))
		require.NoError(t, err)
		assert.Equal(t, "admin", updated.Role.Name)
		statusRes, err := h.SetRoleStatus(ctx, &authv1.SetRoleStatusRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), Status: shared.StatusInactive})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, statusRes.Role.Status)
		deleted, err := h.DeleteRole(ctx, &authv1.DeleteRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), deleted.Role.RoleUuid)
		perms, err := h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantUuid: tenantUUID.String(), RoleUuid: roleUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, permissionUUID.String(), perms.Permissions[0].PermissionUuid)
		added, err := h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuids: []string{permissionUUID.String()}})
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), added.Role.RoleUuid)
		removed, err := h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuid: permissionUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), removed.Role.RoleUuid)
	})

	t.Run("validation and service errors", func(t *testing.T) {
		h := NewRoleGRPCHandler(mockTenantResolver{}, &mockRoleService{})
		_, err := h.ListRoles(ctx, &authv1.ListRolesRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRoles(ctx, &authv1.ListRolesRequest{TenantUuid: tenantUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetRole(ctx, &authv1.GetRoleRequest{TenantUuid: tenantUUID.String(), RoleUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreateRole(ctx, &authv1.CreateRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: "bad", Name: "admin", Description: "Admin role", Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreate := validCreateRoleRequest(tenantUUID, actorUUID)
		badTenantCreate.TenantUuid = "bad"
		_, err = h.CreateRole(ctx, badTenantCreate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreateRole(ctx, &authv1.CreateRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateRole(ctx, &authv1.UpdateRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateRole(ctx, &authv1.UpdateRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: "bad", RoleUuid: roleUUID.String(), Name: "admin", Description: "Admin role", Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetRoleStatus(ctx, &authv1.SetRoleStatusRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetRoleStatus(ctx, &authv1.SetRoleStatusRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: "bad", RoleUuid: roleUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteRole(ctx, &authv1.DeleteRoleRequest{TenantUuid: "bad", ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteRole(ctx, &authv1.DeleteRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: "bad", RoleUuid: roleUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantUuid: "bad", RoleUuid: roleUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantUuid: tenantUUID.String(), RoleUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantUuid: tenantUUID.String(), RoleUuid: roleUUID.String(), Pagination: &authv1.Pagination{SortOrder: "sideways"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantUuid: "bad", ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuids: []string{permissionUUID.String()}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: "bad", RoleUuid: roleUUID.String(), PermissionUuids: []string{permissionUUID.String()}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuids: []string{"bad"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantUuid: "bad", ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuid: permissionUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: "bad", RoleUuid: roleUUID.String(), PermissionUuid: permissionUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		serviceErr := errors.New("db")
		h = NewRoleGRPCHandler(tenantResolver, &mockRoleService{
			getFn:       func(RoleServiceGetFilter) (*RoleServiceGetResult, error) { return nil, serviceErr },
			getByUUIDFn: func(uuid.UUID, int64) (*RoleServiceDataResult, error) { return nil, serviceErr },
			getRolePermissionsFn: func(RoleServiceGetPermissionsFilter) (*RoleServiceGetPermissionsResult, error) {
				return nil, serviceErr
			},
			createFn: func(string, string, bool, bool, string, string, uuid.UUID) (*RoleServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, string, bool, bool, string, uuid.UUID) (*RoleServiceDataResult, error) {
				return nil, serviceErr
			},
			setStatusByUUIDFn: func(uuid.UUID, int64, string, uuid.UUID) (*RoleServiceDataResult, error) { return nil, serviceErr },
			deleteByUUIDFn:    func(uuid.UUID, int64, uuid.UUID) (*RoleServiceDataResult, error) { return nil, serviceErr },
			addRolePermsFn:    func(uuid.UUID, int64, []uuid.UUID, uuid.UUID) (*RoleServiceDataResult, error) { return nil, serviceErr },
			removeRolePermsFn: func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*RoleServiceDataResult, error) { return nil, serviceErr },
		})
		_, err = h.ListRoles(ctx, &authv1.ListRolesRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetRole(ctx, &authv1.GetRoleRequest{TenantUuid: tenantUUID.String(), RoleUuid: roleUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantUuid: tenantUUID.String(), RoleUuid: roleUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreateRole(ctx, validCreateRoleRequest(tenantUUID, actorUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdateRole(ctx, validUpdateRoleRequest(tenantUUID, actorUUID, roleUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetRoleStatus(ctx, &authv1.SetRoleStatusRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeleteRole(ctx, &authv1.DeleteRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuids: []string{permissionUUID.String()}})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), PermissionUuid: permissionUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestAuthorizationGRPCHandler_Authorize(t *testing.T) {
	h := NewAuthorizationGRPCHandler(&mockAuthorizationService{
		authorizeFn: func(req AuthzRequest) Decision {
			assert.Equal(t, "auth", req.Principal)
			return Decision{Allowed: true, Reason: "matched allow"}
		},
	})
	res, err := h.Authorize(context.Background(), &authv1.AuthorizeRequest{Principal: "auth", Action: "user:read", Resource: "user:*"})
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, "matched allow", res.Reason)
}

func TestAccessGRPCHelpers(t *testing.T) {
	assert.Nil(t, permissionProto(nil))
	assert.Nil(t, policyProto(nil))
	assert.Nil(t, policyServiceProto(nil))
	assert.Nil(t, roleProto(nil))
	assert.Empty(t, policyDocumentProto(nil).AsMap())
	assert.Empty(t, policyDocumentProto(datatypes.JSON(`bad`)).AsMap())
	_, err := policyDocumentJSON(nil)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	ids, err := parseIAMUUIDs([]string{testResourceUUID.String()}, "Permission UUID")
	require.NoError(t, err)
	assert.Equal(t, testResourceUUID, ids[0])
}

func accessTenantResolver(t *testing.T, tenantUUID uuid.UUID) mockTenantResolver {
	t.Helper()
	return mockTenantResolver{getByUUIDFn: func(id uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
		assert.Equal(t, tenantUUID, id)
		return &tenantpkg.TenantServiceDataResult{TenantID: 77, TenantUUID: id}, nil
	}}
}

func validCreatePermissionRequest(tenantUUID uuid.UUID, apiUUID uuid.UUID) *authv1.CreatePermissionRequest {
	return &authv1.CreatePermissionRequest{TenantUuid: tenantUUID.String(), Name: "read:users", Description: "Read users permission", Status: shared.StatusActive, ApiUuid: apiUUID.String()}
}

func validUpdatePermissionRequest(tenantUUID uuid.UUID, permissionUUID uuid.UUID) *authv1.UpdatePermissionRequest {
	return &authv1.UpdatePermissionRequest{TenantUuid: tenantUUID.String(), PermissionUuid: permissionUUID.String(), Name: "read:users", Description: "Read users permission", Status: shared.StatusActive}
}

func validPolicyDocumentStruct(t *testing.T) *structpb.Struct {
	t.Helper()
	doc, err := structpb.NewStruct(map[string]any{
		"version": "v1",
		"statement": []any{
			map[string]any{"effect": "allow", "action": []any{"user:read"}, "resource": []any{"user:*"}},
		},
	})
	require.NoError(t, err)
	return doc
}

func validCreatePolicyRequest(tenantUUID uuid.UUID, document *structpb.Struct) *authv1.CreatePolicyRequest {
	description := "Read users"
	return &authv1.CreatePolicyRequest{TenantUuid: tenantUUID.String(), Name: "user:read", Description: &description, Document: document, Version: "v1", Status: shared.StatusActive}
}

func validUpdatePolicyRequest(tenantUUID uuid.UUID, policyUUID uuid.UUID, document *structpb.Struct) *authv1.UpdatePolicyRequest {
	description := "Read users"
	return &authv1.UpdatePolicyRequest{TenantUuid: tenantUUID.String(), PolicyUuid: policyUUID.String(), Name: "user:read", Description: &description, Document: document, Version: "v1", Status: shared.StatusActive}
}

func validCreateRoleRequest(tenantUUID uuid.UUID, actorUUID uuid.UUID) *authv1.CreateRoleRequest {
	return &authv1.CreateRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), Name: "admin", Description: "Admin role", Status: shared.StatusActive}
}

func validUpdateRoleRequest(tenantUUID uuid.UUID, actorUUID uuid.UUID, roleUUID uuid.UUID) *authv1.UpdateRoleRequest {
	return &authv1.UpdateRoleRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: actorUUID.String(), RoleUuid: roleUUID.String(), Name: "admin", Description: "Admin role", Status: shared.StatusActive}
}
