package iam

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	tenantpkg "github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPermissionGRPCHandler_RPCS(t *testing.T) {
	ctx := grpcCallerCtx(77)
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
		API:            &APIServiceDataResult{APIUUID: apiUUID, Name: "users"},
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

		list, err := h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantId: tenantUUID.String(), Name: "read:users", Status: shared.StatusActive, Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		require.NoError(t, err)
		assert.Equal(t, "users", list.Permissions[0].Api.Name)
		got, err := h.GetPermission(ctx, &authv1.GetPermissionRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "read:users", got.Permission.Name)
		created, err := h.CreatePermission(ctx, validCreatePermissionRequest(tenantUUID, apiUUID))
		require.NoError(t, err)
		assert.Equal(t, permissionUUID.String(), created.Permission.PermissionId)
		updated, err := h.UpdatePermission(ctx, validUpdatePermissionRequest(tenantUUID, permissionUUID))
		require.NoError(t, err)
		assert.Equal(t, "read:users", updated.Permission.Name)
		statusRes, err := h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String(), Status: shared.StatusInactive})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, statusRes.Permission.Status)
		deleted, err := h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, permissionUUID.String(), deleted.Permission.PermissionId)
	})

	t.Run("validation and service errors", func(t *testing.T) {
		h := NewPermissionGRPCHandler(mockTenantResolver{}, &mockPermissionService{})
		_, err := h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantId: tenantUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetPermission(ctx, &authv1.GetPermissionRequest{TenantId: tenantUUID.String(), PermissionId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreatePermission(ctx, &authv1.CreatePermissionRequest{TenantId: tenantUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreate := validCreatePermissionRequest(tenantUUID, apiUUID)
		badTenantCreate.TenantId = "bad"
		_, err = h.CreatePermission(ctx, badTenantCreate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdatePermission(ctx, &authv1.UpdatePermissionRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantUpdate := validUpdatePermissionRequest(tenantUUID, permissionUUID)
		badTenantUpdate.TenantId = "bad"
		_, err = h.UpdatePermission(ctx, badTenantUpdate)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantId: "bad", PermissionId: permissionUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantId: "bad", PermissionId: permissionUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantId: tenantUUID.String(), PermissionId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		h = NewPermissionGRPCHandler(mockTenantResolver{getByUUIDFn: func(uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing tenant")
		}}, &mockPermissionService{})
		_, err = h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantId: tenantUUID.String()})
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
		_, err = h.ListPermissions(ctx, &authv1.ListPermissionsRequest{TenantId: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetPermission(ctx, &authv1.GetPermissionRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreatePermission(ctx, validCreatePermissionRequest(tenantUUID, apiUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdatePermission(ctx, validUpdatePermissionRequest(tenantUUID, permissionUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetPermissionStatus(ctx, &authv1.SetPermissionStatusRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeletePermission(ctx, &authv1.DeletePermissionRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func validCreatePermissionRequest(tenantUUID uuid.UUID, apiUUID uuid.UUID) *authv1.CreatePermissionRequest {
	return &authv1.CreatePermissionRequest{TenantId: tenantUUID.String(), Name: "read:users", Description: "Read users permission", Status: shared.StatusActive, ApiId: apiUUID.String()}
}

func validUpdatePermissionRequest(tenantUUID uuid.UUID, permissionUUID uuid.UUID) *authv1.UpdatePermissionRequest {
	return &authv1.UpdatePermissionRequest{TenantId: tenantUUID.String(), PermissionId: permissionUUID.String(), Name: "read:users", Description: "Read users permission", Status: shared.StatusActive}
}
