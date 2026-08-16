package iam

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRoleGRPCHandler_RPCS(t *testing.T) {
	now := time.Now()
	tenantUUID := uuid.New()
	roleUUID := uuid.New()
	actorUUID := uuid.New()
	// The actor comes from the VERIFIED token now, not req.ActorUserId, so the
	// caller context has to carry a user principal.
	ctx := grpcUserCallerCtx(77, actorUUID)
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

		list, err := h.ListRoles(ctx, &authv1.ListRolesRequest{TenantId: tenantUUID.String(), Status: shared.StatusActive, Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "admin", list.Roles[0].Name)
		got, err := h.GetRole(ctx, &authv1.GetRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "read:users", got.Role.Permissions[0].Name)
		created, err := h.CreateRole(ctx, validCreateRoleRequest(tenantUUID))
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), created.Role.RoleId)
		updated, err := h.UpdateRole(ctx, validUpdateRoleRequest(tenantUUID, roleUUID))
		require.NoError(t, err)
		assert.Equal(t, "admin", updated.Role.Name)
		statusRes, err := h.SetRoleStatus(ctx, &authv1.SetRoleStatusRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Status: shared.StatusInactive})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, statusRes.Role.Status)
		deleted, err := h.DeleteRole(ctx, &authv1.DeleteRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), deleted.Role.RoleId)
		perms, err := h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, permissionUUID.String(), perms.Permissions[0].PermissionId)
		added, err := h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionIds: []string{permissionUUID.String()}})
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), added.Role.RoleId)
		removed, err := h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionId: permissionUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), removed.Role.RoleId)
	})

	t.Run("validation and service errors", func(t *testing.T) {
		h := NewRoleGRPCHandler(mockTenantResolver{}, &mockRoleService{})
		_, err := h.ListRoles(ctx, &authv1.ListRolesRequest{TenantId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRoles(ctx, &authv1.ListRolesRequest{TenantId: tenantUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetRole(ctx, &authv1.GetRoleRequest{TenantId: tenantUUID.String(), RoleId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreate := validCreateRoleRequest(tenantUUID)
		badTenantCreate.TenantId = "bad"
		_, err = h.CreateRole(ctx, badTenantCreate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreateRole(ctx, &authv1.CreateRoleRequest{TenantId: tenantUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateRole(ctx, &authv1.UpdateRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetRoleStatus(ctx, &authv1.SetRoleStatusRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteRole(ctx, &authv1.DeleteRoleRequest{TenantId: "bad", RoleId: roleUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantId: "bad", RoleId: roleUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Pagination: &authv1.Pagination{SortOrder: "sideways"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantId: "bad", RoleId: roleUUID.String(), PermissionIds: []string{permissionUUID.String()}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionIds: []string{"bad"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantId: "bad", RoleId: roleUUID.String(), PermissionId: permissionUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionId: "bad"})
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
		_, err = h.ListRoles(ctx, &authv1.ListRolesRequest{TenantId: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetRole(ctx, &authv1.GetRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.ListRolePermissions(ctx, &authv1.ListRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreateRole(ctx, validCreateRoleRequest(tenantUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdateRole(ctx, validUpdateRoleRequest(tenantUUID, roleUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetRoleStatus(ctx, &authv1.SetRoleStatusRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeleteRole(ctx, &authv1.DeleteRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.AddRolePermissions(ctx, &authv1.AddRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionIds: []string{permissionUUID.String()}})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.RemoveRolePermission(ctx, &authv1.RemoveRolePermissionRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionId: permissionUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	// The "invalid actor UUID" cases above are gone: ActorUserId is no longer
	// read. What replaces them is the inverse — a body-supplied actor must NOT be
	// able to stand in for a missing user principal. The service is left nil so any
	// RPC that got as far as calling it would panic rather than pass.
	t.Run("mutating RPCs fail closed for a service principal", func(t *testing.T) {
		h := NewRoleGRPCHandler(tenantResolver, nil)
		serviceCtx := grpcCallerCtx(77)
		innocent := uuid.New().String()

		for name, call := range map[string]func() error{
			"CreateRole": func() error {
				req := validCreateRoleRequest(tenantUUID)
				req.ActorUserId = innocent
				_, err := h.CreateRole(serviceCtx, req)
				return err
			},
			"UpdateRole": func() error {
				req := validUpdateRoleRequest(tenantUUID, roleUUID)
				req.ActorUserId = innocent
				_, err := h.UpdateRole(serviceCtx, req)
				return err
			},
			"SetRoleStatus": func() error {
				_, err := h.SetRoleStatus(serviceCtx, &authv1.SetRoleStatusRequest{TenantId: tenantUUID.String(), ActorUserId: innocent, RoleId: roleUUID.String(), Status: shared.StatusActive})
				return err
			},
			"DeleteRole": func() error {
				_, err := h.DeleteRole(serviceCtx, &authv1.DeleteRoleRequest{TenantId: tenantUUID.String(), ActorUserId: innocent, RoleId: roleUUID.String()})
				return err
			},
			"AddRolePermissions": func() error {
				_, err := h.AddRolePermissions(serviceCtx, &authv1.AddRolePermissionsRequest{TenantId: tenantUUID.String(), ActorUserId: innocent, RoleId: roleUUID.String(), PermissionIds: []string{permissionUUID.String()}})
				return err
			},
			"RemoveRolePermission": func() error {
				_, err := h.RemoveRolePermission(serviceCtx, &authv1.RemoveRolePermissionRequest{TenantId: tenantUUID.String(), ActorUserId: innocent, RoleId: roleUUID.String(), PermissionId: permissionUUID.String()})
				return err
			},
		} {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, codes.PermissionDenied, status.Code(call()))
			})
		}
	})

	// Read-only RPCs need no actor and must keep working for a service principal.
	t.Run("read RPCs still serve a service principal", func(t *testing.T) {
		h := NewRoleGRPCHandler(tenantResolver, &mockRoleService{
			getByUUIDFn: func(uuid.UUID, int64) (*RoleServiceDataResult, error) { return &role, nil },
		})

		got, err := h.GetRole(grpcCallerCtx(77), &authv1.GetRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})

		require.NoError(t, err)
		assert.Equal(t, roleUUID.String(), got.Role.RoleId)
	})
}

func validCreateRoleRequest(tenantUUID uuid.UUID) *authv1.CreateRoleRequest {
	return &authv1.CreateRoleRequest{TenantId: tenantUUID.String(), Name: "admin", Description: "Admin role", Status: shared.StatusActive}
}

func validUpdateRoleRequest(tenantUUID uuid.UUID, roleUUID uuid.UUID) *authv1.UpdateRoleRequest {
	return &authv1.UpdateRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Name: "admin", Description: "Admin role", Status: shared.StatusActive}
}
