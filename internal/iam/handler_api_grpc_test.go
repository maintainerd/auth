package iam

import (
	"context"
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
			createFn: func(tenantID int64, name, displayName, description, apiStatus string, isSystem bool, svcUUID string) (*APIServiceDataResult, error) {
				assert.Equal(t, int64(77), tenantID)
				assert.Equal(t, "login", name)
				assert.Equal(t, serviceUUID.String(), svcUUID)
				assert.False(t, isSystem)
				return &apiResult, nil
			},
			updateFn: func(id uuid.UUID, tenantID int64, name, displayName, description, apiStatus, svcUUID string) (*APIServiceDataResult, error) {
				assert.Equal(t, apiUUID, id)
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
		assert.Equal(t, "login", created.Api.Name)

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
			createFn: func(int64, string, string, string, string, bool, string) (*APIServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, string, string, string, string) (*APIServiceDataResult, error) {
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
			createFn: func(int64, string, string, string, string, bool, string) (*APIServiceDataResult, error) {
				return nil, serviceErr
			},
			updateFn: func(uuid.UUID, int64, string, string, string, string, string) (*APIServiceDataResult, error) {
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

func validCreateAPIRequest(tenantUUID uuid.UUID, serviceUUID uuid.UUID) *authv1.CreateAPIRequest {
	return &authv1.CreateAPIRequest{
		TenantUuid:  tenantUUID.String(),
		Name:        "login",
		DisplayName: "Login API",
		Description: "Login API endpoint",
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
		Status:      shared.StatusActive,
		ServiceUuid: serviceUUID.String(),
	}
}
