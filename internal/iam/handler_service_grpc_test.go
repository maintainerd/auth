package iam

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	tenantpkg "github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServiceGRPCHandler_RPCS(t *testing.T) {
	ctx := grpcCallerCtx(77)
	tenantUUID := uuid.New()
	serviceUUID := uuid.New()
	policyUUID := uuid.New()
	now := time.Now()
	tenantResolver := mockTenantResolver{getByUUIDFn: func(id uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
		assert.Equal(t, tenantUUID, id)
		return &tenantpkg.TenantServiceDataResult{TenantID: 77, TenantUUID: id}, nil
	}}
	serviceResult := ServiceServiceDataResult{
		ServiceUUID: serviceUUID,
		Name:        "auth",
		DisplayName: "Auth Service",
		Description: "Authentication service",
		Version:     "v1",
		Status:      shared.StatusActive,
		APICount:    2,
		PolicyCount: 3,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.Run("success", func(t *testing.T) {
		svc := &mockServiceService{
			getFn: func(f ServiceServiceGetFilter) (*ServiceServiceGetResult, error) {
				require.NotNil(t, f.TenantID)
				assert.Equal(t, int64(77), *f.TenantID)
				assert.Equal(t, []string{shared.StatusActive}, f.Status)
				assert.Equal(t, 2, f.Page)
				assert.Equal(t, 5, f.Limit)
				return &ServiceServiceGetResult{Data: []ServiceServiceDataResult{serviceResult}, Total: 1, Page: 2, Limit: 5, TotalPages: 1}, nil
			},
			getByUUIDFn: func(id uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
				assert.Equal(t, serviceUUID, id)
				assert.Equal(t, int64(77), tenantID)
				return &serviceResult, nil
			},
			createFn: func(name string, displayName string, description string, version string, isSystem bool, serviceStatus string, tenantID int64) (*ServiceServiceDataResult, error) {
				assert.Equal(t, "auth", name)
				assert.False(t, isSystem)
				assert.Equal(t, int64(77), tenantID)
				assert.Equal(t, shared.StatusActive, serviceStatus)
				return &serviceResult, nil
			},
			updateFn: func(id uuid.UUID, tenantID int64, name string, displayName string, description string, version string, isSystem bool, serviceStatus string) (*ServiceServiceDataResult, error) {
				assert.Equal(t, serviceUUID, id)
				assert.Equal(t, "Auth Service", displayName)
				return &serviceResult, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, tenantID int64, serviceStatus string) (*ServiceServiceDataResult, error) {
				assert.Equal(t, serviceUUID, id)
				assert.Equal(t, shared.StatusMaintenance, serviceStatus)
				result := serviceResult
				result.Status = serviceStatus
				return &result, nil
			},
			deleteByUUIDFn: func(id uuid.UUID, tenantID int64) (*ServiceServiceDataResult, error) {
				assert.Equal(t, serviceUUID, id)
				return &serviceResult, nil
			},
			assignPolicyFn: func(svcID uuid.UUID, polID uuid.UUID, tenantID int64) error {
				assert.Equal(t, serviceUUID, svcID)
				assert.Equal(t, policyUUID, polID)
				assert.Equal(t, int64(77), tenantID)
				return nil
			},
			removePolicyFn: func(svcID uuid.UUID, polID uuid.UUID, tenantID int64) error {
				assert.Equal(t, serviceUUID, svcID)
				assert.Equal(t, policyUUID, polID)
				return nil
			},
		}
		h := NewServiceGRPCHandler(tenantResolver, svc, nil)

		list, err := h.ListServices(ctx, &authv1.ListServicesRequest{
			TenantId:   tenantUUID.String(),
			Status:     []string{shared.StatusActive},
			Pagination: &authv1.Pagination{Page: 2, Limit: 5, SortBy: "created_at", SortOrder: SortOrderDesc},
		})
		require.NoError(t, err)
		assert.Len(t, list.Services, 1)
		assert.Equal(t, int32(2), list.Page.Page)
		assert.Equal(t, serviceUUID.String(), list.Services[0].ServiceId)

		got, err := h.GetService(ctx, &authv1.GetServiceRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "auth", got.Service.Name)

		created, err := h.CreateService(ctx, validCreateServiceRequest(tenantUUID))
		require.NoError(t, err)
		assert.Equal(t, "Auth Service", created.Service.DisplayName)

		updated, err := h.UpdateService(ctx, validUpdateServiceRequest(tenantUUID, serviceUUID))
		require.NoError(t, err)
		assert.Equal(t, "v1", updated.Service.Version)

		statusRes, err := h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), Status: shared.StatusMaintenance})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusMaintenance, statusRes.Service.Status)

		deleted, err := h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "auth", deleted.Service.Name)

		assigned, err := h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), PolicyId: policyUUID.String()})
		require.NoError(t, err)
		assert.True(t, assigned.Assigned)

		removed, err := h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), PolicyId: policyUUID.String()})
		require.NoError(t, err)
		assert.True(t, removed.Removed)
	})

	t.Run("validation and dependency errors", func(t *testing.T) {
		h := NewServiceGRPCHandler(mockTenantResolver{}, &mockServiceService{}, nil)
		_, err := h.ListServices(ctx, &authv1.ListServicesRequest{TenantId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListServices(ctx, &authv1.ListServicesRequest{TenantId: tenantUUID.String(), Status: []string{"bad"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetService(ctx, &authv1.GetServiceRequest{TenantId: tenantUUID.String(), ServiceId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreateService(ctx, &authv1.CreateServiceRequest{TenantId: tenantUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreateService := validCreateServiceRequest(tenantUUID)
		badTenantCreateService.TenantId = "bad"
		_, err = h.CreateService(ctx, badTenantCreateService)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateService(ctx, &authv1.UpdateServiceRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantUpdateService := validUpdateServiceRequest(tenantUUID, serviceUUID)
		badTenantUpdateService.TenantId = "bad"
		_, err = h.UpdateService(ctx, badTenantUpdateService)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantId: "bad", ServiceId: serviceUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantId: "bad", ServiceId: serviceUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantId: tenantUUID.String(), ServiceId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: "bad", PolicyId: policyUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), PolicyId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: "bad", PolicyId: policyUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), PolicyId: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		h = NewServiceGRPCHandler(mockTenantResolver{getByUUIDFn: func(uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing tenant")
		}}, &mockServiceService{}, nil)
		_, err = h.ListServices(ctx, &authv1.ListServicesRequest{TenantId: tenantUUID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))

		serviceErr := errors.New("db")
		h = NewServiceGRPCHandler(tenantResolver, &mockServiceService{
			getFn:       func(ServiceServiceGetFilter) (*ServiceServiceGetResult, error) { return nil, serviceErr },
			getByUUIDFn: func(uuid.UUID, int64) (*ServiceServiceDataResult, error) { return nil, serviceErr },
			createFn: func(string, string, string, string, bool, string, int64) (*ServiceServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, string, string, string, bool, string) (*ServiceServiceDataResult, error) {
				return nil, serviceErr
			},
			setStatusByUUIDFn: func(uuid.UUID, int64, string) (*ServiceServiceDataResult, error) { return nil, serviceErr },
			deleteByUUIDFn:    func(uuid.UUID, int64) (*ServiceServiceDataResult, error) { return nil, serviceErr },
			assignPolicyFn:    func(uuid.UUID, uuid.UUID, int64) error { return serviceErr },
			removePolicyFn:    func(uuid.UUID, uuid.UUID, int64) error { return serviceErr },
		}, nil)
		_, err = h.ListServices(ctx, &authv1.ListServicesRequest{TenantId: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetService(ctx, &authv1.GetServiceRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreateService(ctx, validCreateServiceRequest(tenantUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdateService(ctx, validUpdateServiceRequest(tenantUUID, serviceUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), PolicyId: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantId: tenantUUID.String(), ServiceId: serviceUUID.String(), PolicyId: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestServiceGRPCHandler_GetMyPolicyBundle(t *testing.T) {
	generatedAt := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	serviceCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		Sub:         "ignored",
		Service:     "core",
		SubjectType: "service",
		ClientID:    "client-1",
	})
	subjectServiceCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		Sub:         "core-sub",
		SubjectType: "service",
		ClientID:    "client-2",
	})

	t.Run("success returns bundle and etag", func(t *testing.T) {
		h := NewServiceGRPCHandler(nil, nil, &mockAuthorizationService{
			policyBundleFn: func(identity ServiceIdentity) (*PolicyBundle, string, error) {
				assert.Equal(t, ServiceIdentity{ServiceName: "core", ClientID: "client-1"}, identity)
				return &PolicyBundle{
					Service:     "core",
					Version:     "vabc",
					GeneratedAt: generatedAt,
					Policies: []PolicyDocument{{
						Version: "v1",
						Statement: []PolicyStatement{{
							Effect:   "allow",
							Action:   []string{"*"},
							Resource: []string{"*"},
						}},
					}},
				}, `"vabc"`, nil
			},
		})

		resp, err := h.GetMyPolicyBundle(serviceCtx, &authv1.GetMyPolicyBundleRequest{})

		require.NoError(t, err)
		require.NotNil(t, resp.Bundle)
		assert.Equal(t, `"vabc"`, resp.Etag)
		assert.False(t, resp.NotModified)
		assert.Equal(t, "core", resp.Bundle.Service)
		assert.Equal(t, "vabc", resp.Bundle.Version)
		assert.Equal(t, generatedAt, resp.Bundle.GeneratedAt.AsTime())
		require.Len(t, resp.Bundle.Policies, 1)
		assert.Equal(t, "v1", resp.Bundle.Policies[0].Fields["version"].GetStringValue())
	})

	t.Run("not modified skips bundle payload", func(t *testing.T) {
		h := NewServiceGRPCHandler(nil, nil, &mockAuthorizationService{
			policyBundleFn: func(identity ServiceIdentity) (*PolicyBundle, string, error) {
				assert.Equal(t, ServiceIdentity{ServiceName: "core-sub", ClientID: "client-2"}, identity)
				return &PolicyBundle{Service: "core-sub", Version: "v1", GeneratedAt: generatedAt}, `"v1"`, nil
			},
		})

		resp, err := h.GetMyPolicyBundle(subjectServiceCtx, &authv1.GetMyPolicyBundleRequest{IfNoneMatch: `"v1"`})

		require.NoError(t, err)
		assert.Nil(t, resp.Bundle)
		assert.True(t, resp.NotModified)
		assert.Equal(t, `"v1"`, resp.Etag)
	})

	t.Run("requires service identity", func(t *testing.T) {
		h := NewServiceGRPCHandler(nil, nil, &mockAuthorizationService{})
		_, err := h.GetMyPolicyBundle(context.Background(), &authv1.GetMyPolicyBundleRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))

		userCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{Sub: uuid.NewString()})
		_, err = h.GetMyPolicyBundle(userCtx, &authv1.GetMyPolicyBundleRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("requires authorization service", func(t *testing.T) {
		h := NewServiceGRPCHandler(nil, nil, nil)
		_, err := h.GetMyPolicyBundle(serviceCtx, &authv1.GetMyPolicyBundleRequest{})
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("propagates service error", func(t *testing.T) {
		h := NewServiceGRPCHandler(nil, nil, &mockAuthorizationService{
			policyBundleFn: func(ServiceIdentity) (*PolicyBundle, string, error) {
				return nil, "", apperror.NewNotFoundWithReason("service principal not found or inactive")
			},
		})

		_, err := h.GetMyPolicyBundle(serviceCtx, &authv1.GetMyPolicyBundleRequest{})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("nil bundle is allowed", func(t *testing.T) {
		got := servicePolicyBundleProto(nil)
		assert.Nil(t, got)
	})
}

func validCreateServiceRequest(tenantUUID uuid.UUID) *authv1.CreateServiceRequest {
	return &authv1.CreateServiceRequest{
		TenantId:    tenantUUID.String(),
		Name:        "auth",
		DisplayName: "Auth Service",
		Description: "Authentication service",
		Version:     "v1",
		Status:      shared.StatusActive,
	}
}

func validUpdateServiceRequest(tenantUUID uuid.UUID, serviceUUID uuid.UUID) *authv1.UpdateServiceRequest {
	return &authv1.UpdateServiceRequest{
		TenantId:    tenantUUID.String(),
		ServiceId:   serviceUUID.String(),
		Name:        "auth",
		DisplayName: "Auth Service",
		Description: "Authentication service",
		Version:     "v1",
		Status:      shared.StatusActive,
	}
}
