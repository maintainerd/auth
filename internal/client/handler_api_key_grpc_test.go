package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

type testAPIKeyService struct {
	getFn                       func(ctx context.Context, filter APIKeyServiceGetFilter, requestingUserUUID uuid.UUID) (*APIKeyServiceGetResult, error)
	getByUUIDFn                 func(ctx context.Context, akUUID uuid.UUID, tenantID int64, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	getConfigByUUIDFn           func(ctx context.Context, akUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	createFn                    func(ctx context.Context, tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error)
	updateFn                    func(ctx context.Context, akUUID uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	setStatusByUUIDFn           func(ctx context.Context, akUUID uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error)
	deleteFn                    func(ctx context.Context, akUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error)
	getAPIKeyAPIsFn             func(ctx context.Context, tenantID int64, akUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error)
	addAPIKeyAPIsFn             func(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	removeAPIKeyAPIFn           func(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID) error
	getAPIKeyAPIPermissionsFn   func(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	addAPIKeyAPIPermissionsFn   func(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	removeAPIKeyAPIPermissionFn func(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
}

func (m *testAPIKeyService) Get(ctx context.Context, filter APIKeyServiceGetFilter, requestingUserUUID uuid.UUID) (*APIKeyServiceGetResult, error) {
	return m.getFn(ctx, filter, requestingUserUUID)
}
func (m *testAPIKeyService) GetByUUID(ctx context.Context, akUUID uuid.UUID, tenantID int64, requestingUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	return m.getByUUIDFn(ctx, akUUID, tenantID, requestingUserUUID)
}
func (m *testAPIKeyService) GetConfigByUUID(ctx context.Context, akUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	return m.getConfigByUUIDFn(ctx, akUUID, tenantID)
}
func (m *testAPIKeyService) Create(ctx context.Context, tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error) {
	return m.createFn(ctx, tenantID, name, description, config, expiresAt, rateLimit, status)
}
func (m *testAPIKeyService) Update(ctx context.Context, akUUID uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updaterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	return m.updateFn(ctx, akUUID, tenantID, name, description, config, expiresAt, rateLimit, status, updaterUserUUID)
}
func (m *testAPIKeyService) SetStatusByUUID(ctx context.Context, akUUID uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error) {
	return m.setStatusByUUIDFn(ctx, akUUID, tenantID, status)
}
func (m *testAPIKeyService) Delete(ctx context.Context, akUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*APIKeyServiceDataResult, error) {
	return m.deleteFn(ctx, akUUID, tenantID, deleterUserUUID)
}
func (m *testAPIKeyService) GetAPIKeyAPIs(ctx context.Context, tenantID int64, akUUID uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error) {
	return m.getAPIKeyAPIsFn(ctx, tenantID, akUUID, page, limit, sortBy, sortOrder)
}
func (m *testAPIKeyService) AddAPIKeyAPIs(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	return m.addAPIKeyAPIsFn(ctx, tenantID, akUUID, apiUUIDs)
}
func (m *testAPIKeyService) RemoveAPIKeyAPI(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID) error {
	return m.removeAPIKeyAPIFn(ctx, tenantID, akUUID, apiUUID)
}
func (m *testAPIKeyService) GetAPIKeyAPIPermissions(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	return m.getAPIKeyAPIPermissionsFn(ctx, tenantID, akUUID, apiUUID)
}
func (m *testAPIKeyService) AddAPIKeyAPIPermissions(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	return m.addAPIKeyAPIPermissionsFn(ctx, tenantID, akUUID, apiUUID, permissionUUIDs)
}
func (m *testAPIKeyService) RemoveAPIKeyAPIPermission(ctx context.Context, tenantID int64, akUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	return m.removeAPIKeyAPIPermissionFn(ctx, tenantID, akUUID, apiUUID, permissionUUID)
}

func TestAPIKeyGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	now := time.Now()
	resolver := &mockClientTenantResolver{}

	akResult := APIKeyServiceDataResult{
		APIKeyUUID: akUUID, Name: "my-key", Description: "Test key", KeyPrefix: "lula_", Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}

	t.Run("list success", func(t *testing.T) {
		svc := &testAPIKeyService{
			getFn: func(ctx context.Context, filter APIKeyServiceGetFilter, ru uuid.UUID) (*APIKeyServiceGetResult, error) {
				return &APIKeyServiceGetResult{Data: []APIKeyServiceDataResult{akResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewAPIKeyGRPCHandler(resolver, svc)
		res, err := h.ListAPIKeys(ctx, &authv1.ListAPIKeysRequest{TenantUuid: tenantUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.ApiKeys) != 1 {
			t.Fatalf("expected 1 key, got %d", len(res.ApiKeys))
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testAPIKeyService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, ru uuid.UUID) (*APIKeyServiceDataResult, error) {
				return &akResult, nil
			},
		}
		h := NewAPIKeyGRPCHandler(resolver, svc)
		_, err := h.GetAPIKey(ctx, &authv1.GetAPIKeyRequest{TenantUuid: tenantUUID.String(), ApiKeyUuid: akUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testAPIKeyService{
			createFn: func(ctx context.Context, tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error) {
				return &akResult, "raw-key", nil
			},
		}
		h := NewAPIKeyGRPCHandler(resolver, svc)
		res, err := h.CreateAPIKey(ctx, &authv1.CreateAPIKeyRequest{TenantUuid: tenantUUID.String(), Name: "my-key", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Key != "raw-key" {
			t.Errorf("expected raw-key, got %s", res.Key)
		}
	})

	t.Run("addAPIs success", func(t *testing.T) {
		svc := &testAPIKeyService{
			addAPIKeyAPIsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiUUIDs []uuid.UUID) error { return nil },
		}
		h := NewAPIKeyGRPCHandler(resolver, svc)
		res, err := h.AddAPIKeyAPIs(ctx, &authv1.AddAPIKeyAPIsRequest{TenantUuid: tenantUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuids: []string{apiUUID.String()}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("addAPIPermissions success", func(t *testing.T) {
		svc := &testAPIKeyService{
			addAPIKeyAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permIDs []uuid.UUID) error { return nil },
		}
		h := NewAPIKeyGRPCHandler(resolver, svc)
		res, err := h.AddAPIKeyAPIPermissions(ctx, &authv1.AddAPIKeyAPIPermissionsRequest{TenantUuid: tenantUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		svc := &testAPIKeyService{
			getFn: func(ctx context.Context, filter APIKeyServiceGetFilter, ru uuid.UUID) (*APIKeyServiceGetResult, error) {
				return &APIKeyServiceGetResult{}, nil
			},
		}
		h := NewAPIKeyGRPCHandler(resolver, svc)
		_, err := h.ListAPIKeys(ctx, &authv1.ListAPIKeysRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service errors", func(t *testing.T) {
		svcErr := errors.New("db error")
		svc := &testAPIKeyService{
			getFn: func(ctx context.Context, filter APIKeyServiceGetFilter, ru uuid.UUID) (*APIKeyServiceGetResult, error) { return nil, svcErr },
		}
		h := NewAPIKeyGRPCHandler(resolver, svc)
		_, err := h.ListAPIKeys(ctx, &authv1.ListAPIKeysRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})
}

func TestAPIKeyProtoConverters(t *testing.T) {
	assert := require.New(t)
	assert.Nil(toAPIKeyProto(nil))
	ak := &APIKeyServiceDataResult{APIKeyUUID: uuid.New(), Name: "test", Status: "active"}
	proto := toAPIKeyProto(ak)
	assert.Equal("test", proto.Name)
}
