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
)

type mockTenantResolver struct {
	getByUUIDFn func(uuid.UUID) (*tenantpkg.TenantServiceDataResult, error)
}

func (m mockTenantResolver) GetByUUID(_ context.Context, tenantUUID uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(tenantUUID)
	}
	return &tenantpkg.TenantServiceDataResult{TenantID: 99, TenantUUID: tenantUUID}, nil
}

func TestServiceGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
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
		h := NewServiceGRPCHandler(tenantResolver, svc)

		list, err := h.ListServices(ctx, &authv1.ListServicesRequest{
			TenantUuid: tenantUUID.String(),
			Status:     []string{shared.StatusActive},
			Pagination: &authv1.Pagination{Page: 2, Limit: 5, SortBy: "created_at", SortOrder: SortOrderDesc},
		})
		require.NoError(t, err)
		assert.Len(t, list.Services, 1)
		assert.Equal(t, int32(2), list.Page.Page)
		assert.Equal(t, serviceUUID.String(), list.Services[0].ServiceUuid)

		got, err := h.GetService(ctx, &authv1.GetServiceRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "auth", got.Service.Name)

		created, err := h.CreateService(ctx, validCreateServiceRequest(tenantUUID))
		require.NoError(t, err)
		assert.Equal(t, "Auth Service", created.Service.DisplayName)

		updated, err := h.UpdateService(ctx, validUpdateServiceRequest(tenantUUID, serviceUUID))
		require.NoError(t, err)
		assert.Equal(t, "v1", updated.Service.Version)

		statusRes, err := h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), Status: shared.StatusMaintenance})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusMaintenance, statusRes.Service.Status)

		deleted, err := h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "auth", deleted.Service.Name)

		assigned, err := h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), PolicyUuid: policyUUID.String()})
		require.NoError(t, err)
		assert.True(t, assigned.Assigned)

		removed, err := h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), PolicyUuid: policyUUID.String()})
		require.NoError(t, err)
		assert.True(t, removed.Removed)
	})

	t.Run("validation and dependency errors", func(t *testing.T) {
		h := NewServiceGRPCHandler(mockTenantResolver{}, &mockServiceService{})
		_, err := h.ListServices(ctx, &authv1.ListServicesRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListServices(ctx, &authv1.ListServicesRequest{TenantUuid: tenantUUID.String(), Status: []string{"bad"}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetService(ctx, &authv1.GetServiceRequest{TenantUuid: tenantUUID.String(), ServiceUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreateService(ctx, &authv1.CreateServiceRequest{TenantUuid: tenantUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreateService := validCreateServiceRequest(tenantUUID)
		badTenantCreateService.TenantUuid = "bad"
		_, err = h.CreateService(ctx, badTenantCreateService)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateService(ctx, &authv1.UpdateServiceRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantUpdateService := validUpdateServiceRequest(tenantUUID, serviceUUID)
		badTenantUpdateService.TenantUuid = "bad"
		_, err = h.UpdateService(ctx, badTenantUpdateService)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantUuid: "bad", ServiceUuid: serviceUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantUuid: "bad", ServiceUuid: serviceUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantUuid: tenantUUID.String(), ServiceUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: "bad", PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: "bad", PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), PolicyUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		h = NewServiceGRPCHandler(mockTenantResolver{getByUUIDFn: func(uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing tenant")
		}}, &mockServiceService{})
		_, err = h.ListServices(ctx, &authv1.ListServicesRequest{TenantUuid: tenantUUID.String()})
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
		})
		_, err = h.ListServices(ctx, &authv1.ListServicesRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.GetService(ctx, &authv1.GetServiceRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreateService(ctx, validCreateServiceRequest(tenantUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdateService(ctx, validUpdateServiceRequest(tenantUUID, serviceUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetServiceStatus(ctx, &authv1.SetServiceStatusRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeleteService(ctx, &authv1.DeleteServiceRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.AssignServicePolicy(ctx, &authv1.AssignServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.RemoveServicePolicy(ctx, &authv1.RemoveServicePolicyRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String(), PolicyUuid: policyUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestAPIGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	apiUUID := uuid.New()
	serviceUUID := uuid.New()
	now := time.Now()
	tenantResolver := mockTenantResolver{getByUUIDFn: func(id uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
		assert.Equal(t, tenantUUID, id)
		return &tenantpkg.TenantServiceDataResult{TenantID: 77, TenantUUID: id}, nil
	}}
	serviceResult := &ServiceServiceDataResult{
		ServiceUUID: serviceUUID,
		Name:        "auth",
		DisplayName: "Auth Service",
		Description: "Authentication service",
		Version:     "v1",
		Status:      shared.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	apiResult := APIServiceDataResult{
		APIUUID:     apiUUID,
		Name:        "login",
		DisplayName: "Login API",
		Description: "Login API endpoint",
		APIType:     shared.APITypeRest,
		Identifier:  "auth.login",
		Status:      shared.StatusActive,
		Service:     serviceResult,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.Run("success", func(t *testing.T) {
		apiSvc := &mockAPIService{
			getServiceIDByUUIDFn: func(id uuid.UUID) (int64, error) {
				assert.Equal(t, serviceUUID, id)
				return 42, nil
			},
			getFn: func(f APIServiceGetFilter) (*APIServiceGetResult, error) {
				assert.Equal(t, int64(77), f.TenantID)
				require.NotNil(t, f.ServiceID)
				assert.Equal(t, int64(42), *f.ServiceID)
				assert.Equal(t, []string{shared.StatusActive}, f.Status)
				return &APIServiceGetResult{Data: []APIServiceDataResult{apiResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
			getByUUIDFn: func(id uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
				assert.Equal(t, apiUUID, id)
				assert.Equal(t, int64(77), tenantID)
				return &apiResult, nil
			},
			createFn: func(tenantID int64, name, displayName, description, apiType, apiStatus string, isSystem bool, svcUUID string) (*APIServiceDataResult, error) {
				assert.Equal(t, int64(77), tenantID)
				assert.Equal(t, "login", name)
				assert.Equal(t, serviceUUID.String(), svcUUID)
				assert.False(t, isSystem)
				return &apiResult, nil
			},
			updateFn: func(id uuid.UUID, tenantID int64, name, displayName, description, apiType, apiStatus, svcUUID string) (*APIServiceDataResult, error) {
				assert.Equal(t, apiUUID, id)
				assert.Equal(t, shared.APITypeRest, apiType)
				return &apiResult, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, tenantID int64, apiStatus string) (*APIServiceDataResult, error) {
				assert.Equal(t, apiUUID, id)
				result := apiResult
				result.Status = apiStatus
				return &result, nil
			},
			deleteByUUIDFn: func(id uuid.UUID, tenantID int64) (*APIServiceDataResult, error) {
				assert.Equal(t, apiUUID, id)
				return &apiResult, nil
			},
		}
		h := NewAPIGRPCHandler(tenantResolver, apiSvc)

		list, err := h.ListAPIs(ctx, &authv1.ListAPIsRequest{
			TenantUuid:  tenantUUID.String(),
			ServiceUuid: serviceUUID.String(),
			Status:      []string{shared.StatusActive},
			Pagination:  &authv1.Pagination{Page: 1, Limit: 10},
		})
		require.NoError(t, err)
		assert.Len(t, list.Apis, 1)
		assert.Equal(t, "auth", list.Apis[0].Service.Name)

		got, err := h.GetAPI(ctx, &authv1.GetAPIRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "login", got.Api.Name)

		created, err := h.CreateAPI(ctx, validCreateAPIRequest(tenantUUID, serviceUUID))
		require.NoError(t, err)
		assert.Equal(t, shared.APITypeRest, created.Api.ApiType)

		updated, err := h.UpdateAPI(ctx, validUpdateAPIRequest(tenantUUID, apiUUID, serviceUUID))
		require.NoError(t, err)
		assert.Equal(t, "auth.login", updated.Api.Identifier)

		statusRes, err := h.SetAPIStatus(ctx, &authv1.SetAPIStatusRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String(), Status: shared.StatusInactive})
		require.NoError(t, err)
		assert.Equal(t, shared.StatusInactive, statusRes.Api.Status)

		deleted, err := h.DeleteAPI(ctx, &authv1.DeleteAPIRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "login", deleted.Api.Name)
	})

	t.Run("validation and dependency errors", func(t *testing.T) {
		h := NewAPIGRPCHandler(mockTenantResolver{}, &mockAPIService{})
		_, err := h.ListAPIs(ctx, &authv1.ListAPIsRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListAPIs(ctx, &authv1.ListAPIsRequest{TenantUuid: tenantUUID.String(), ApiType: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.ListAPIs(ctx, &authv1.ListAPIsRequest{TenantUuid: tenantUUID.String(), ServiceUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.GetAPI(ctx, &authv1.GetAPIRequest{TenantUuid: tenantUUID.String(), ApiUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.CreateAPI(ctx, &authv1.CreateAPIRequest{TenantUuid: tenantUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantCreateAPI := validCreateAPIRequest(tenantUUID, serviceUUID)
		badTenantCreateAPI.TenantUuid = "bad"
		_, err = h.CreateAPI(ctx, badTenantCreateAPI)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateAPI(ctx, &authv1.UpdateAPIRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		badTenantUpdateAPI := validUpdateAPIRequest(tenantUUID, apiUUID, serviceUUID)
		badTenantUpdateAPI.TenantUuid = "bad"
		_, err = h.UpdateAPI(ctx, badTenantUpdateAPI)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetAPIStatus(ctx, &authv1.SetAPIStatusRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String(), Status: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetAPIStatus(ctx, &authv1.SetAPIStatusRequest{TenantUuid: "bad", ApiUuid: apiUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteAPI(ctx, &authv1.DeleteAPIRequest{TenantUuid: "bad", ApiUuid: apiUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteAPI(ctx, &authv1.DeleteAPIRequest{TenantUuid: tenantUUID.String(), ApiUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		h = NewAPIGRPCHandler(mockTenantResolver{getByUUIDFn: func(uuid.UUID) (*tenantpkg.TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing tenant")
		}}, &mockAPIService{})
		_, err = h.ListAPIs(ctx, &authv1.ListAPIsRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))

		serviceErr := errors.New("db")
		h = NewAPIGRPCHandler(tenantResolver, &mockAPIService{
			getServiceIDByUUIDFn: func(uuid.UUID) (int64, error) { return 0, serviceErr },
			getFn:                func(APIServiceGetFilter) (*APIServiceGetResult, error) { return nil, serviceErr },
			getByUUIDFn:          func(uuid.UUID, int64) (*APIServiceDataResult, error) { return nil, serviceErr },
			createFn: func(int64, string, string, string, string, string, bool, string) (*APIServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, string, string, string, string, string) (*APIServiceDataResult, error) {
				return nil, serviceErr
			},
			setStatusByUUIDFn: func(uuid.UUID, int64, string) (*APIServiceDataResult, error) { return nil, serviceErr },
			deleteByUUIDFn:    func(uuid.UUID, int64) (*APIServiceDataResult, error) { return nil, serviceErr },
		})
		_, err = h.ListAPIs(ctx, &authv1.ListAPIsRequest{TenantUuid: tenantUUID.String(), ServiceUuid: serviceUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))

		h = NewAPIGRPCHandler(tenantResolver, &mockAPIService{getFn: func(APIServiceGetFilter) (*APIServiceGetResult, error) { return nil, serviceErr }})
		_, err = h.ListAPIs(ctx, &authv1.ListAPIsRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))

		h = NewAPIGRPCHandler(tenantResolver, &mockAPIService{
			getByUUIDFn: func(uuid.UUID, int64) (*APIServiceDataResult, error) { return nil, serviceErr },
			createFn: func(int64, string, string, string, string, string, bool, string) (*APIServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, string, string, string, string, string) (*APIServiceDataResult, error) {
				return nil, serviceErr
			},
			setStatusByUUIDFn: func(uuid.UUID, int64, string) (*APIServiceDataResult, error) { return nil, serviceErr },
			deleteByUUIDFn:    func(uuid.UUID, int64) (*APIServiceDataResult, error) { return nil, serviceErr },
		})
		_, err = h.GetAPI(ctx, &authv1.GetAPIRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.CreateAPI(ctx, validCreateAPIRequest(tenantUUID, serviceUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.UpdateAPI(ctx, validUpdateAPIRequest(tenantUUID, apiUUID, serviceUUID))
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.SetAPIStatus(ctx, &authv1.SetAPIStatusRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = h.DeleteAPI(ctx, &authv1.DeleteAPIRequest{TenantUuid: tenantUUID.String(), ApiUuid: apiUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestIAMGRPCHelpers(t *testing.T) {
	id := uuid.New()
	parsed, err := iamParseUUID(id.String(), "Test UUID")
	require.NoError(t, err)
	assert.Equal(t, id, parsed)
	_, err = iamParseUUID("", "Test UUID")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = iamParseUUID("bad", "Test UUID")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	defaultPage := iamPaginationDTO(nil)
	assert.Equal(t, 1, defaultPage.Page)
	pagination := iamPaginationDTO(&authv1.Pagination{SortBy: "created_at", SortOrder: SortOrderDesc})
	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, "created_at", pagination.SortBy)
	assert.Equal(t, int32(3), iamPageProto(12, 2, 5, 3).TotalPages)
	assert.Nil(t, iamOptionalString(""))
	require.NotNil(t, iamOptionalString("value"))

	assert.Nil(t, serviceProto(nil))
	assert.Nil(t, apiProto(nil))
	assert.NotNil(t, serviceProto(&ServiceServiceDataResult{ServiceUUID: id}))
	assert.NotNil(t, apiProto(&APIServiceDataResult{APIUUID: id, Service: &ServiceServiceDataResult{ServiceUUID: id}}))
}

func validCreateServiceRequest(tenantUUID uuid.UUID) *authv1.CreateServiceRequest {
	return &authv1.CreateServiceRequest{
		TenantUuid:  tenantUUID.String(),
		Name:        "auth",
		DisplayName: "Auth Service",
		Description: "Authentication service",
		Version:     "v1",
		Status:      shared.StatusActive,
	}
}

func validUpdateServiceRequest(tenantUUID uuid.UUID, serviceUUID uuid.UUID) *authv1.UpdateServiceRequest {
	return &authv1.UpdateServiceRequest{
		TenantUuid:  tenantUUID.String(),
		ServiceUuid: serviceUUID.String(),
		Name:        "auth",
		DisplayName: "Auth Service",
		Description: "Authentication service",
		Version:     "v1",
		Status:      shared.StatusActive,
	}
}

func validCreateAPIRequest(tenantUUID uuid.UUID, serviceUUID uuid.UUID) *authv1.CreateAPIRequest {
	return &authv1.CreateAPIRequest{
		TenantUuid:  tenantUUID.String(),
		Name:        "login",
		DisplayName: "Login API",
		Description: "Login API endpoint",
		ApiType:     shared.APITypeRest,
		Status:      shared.StatusActive,
		ServiceUuid: serviceUUID.String(),
	}
}

func validUpdateAPIRequest(tenantUUID uuid.UUID, apiUUID uuid.UUID, serviceUUID uuid.UUID) *authv1.UpdateAPIRequest {
	return &authv1.UpdateAPIRequest{
		TenantUuid:  tenantUUID.String(),
		ApiUuid:     apiUUID.String(),
		Name:        "login",
		DisplayName: "Login API",
		Description: "Login API endpoint",
		ApiType:     shared.APITypeRest,
		Status:      shared.StatusActive,
		ServiceUuid: serviceUUID.String(),
	}
}
