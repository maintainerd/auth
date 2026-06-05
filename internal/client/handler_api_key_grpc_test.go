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
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestAPIKeyGRPCHandler_AllMissingHandlers(t *testing.T) {
	ctx := context.Background()
	tUUID := uuid.New()
	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	now := time.Now()
	okResolver := &mockClientTenantResolver{}

	akResult := APIKeyServiceDataResult{
		APIKeyUUID: akUUID, Name: "my-key", Description: "Test key",
		KeyPrefix: "lula_", Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	cfgStruct, _ := structpb.NewStruct(map[string]any{"key": "value"})

	t.Run("GetAPIKeyConfig success", func(t *testing.T) {
		svc := &testAPIKeyService{
			getConfigByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) {
				return datatypes.JSON(`{"foo":"bar"}`), nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.GetAPIKeyConfig(ctx, &authv1.GetAPIKeyConfigRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Config == nil {
			t.Error("expected config")
		}
	})

	t.Run("GetAPIKeyConfig empty config", func(t *testing.T) {
		svc := &testAPIKeyService{
			getConfigByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) {
				return datatypes.JSON(""), nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.GetAPIKeyConfig(ctx, &authv1.GetAPIKeyConfigRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Config == nil {
			t.Error("expected non-nil config for empty config")
		}
	})

	t.Run("GetAPIKeyConfig invalid JSON config", func(t *testing.T) {
		svc := &testAPIKeyService{
			getConfigByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) {
				return datatypes.JSON("not-json{{{"), nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.GetAPIKeyConfig(ctx, &authv1.GetAPIKeyConfigRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Config == nil {
			t.Error("expected non-nil config proto even for invalid JSON")
		}
	})

	t.Run("UpdateAPIKey success", func(t *testing.T) {
		svc := &testAPIKeyService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updater uuid.UUID) (*APIKeyServiceDataResult, error) {
				return &akResult, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.UpdateAPIKey(ctx, &authv1.UpdateAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), Name: strptr("updated")})
		if err != nil {
			t.Fatal(err)
		}
		if res.ApiKey.Name != "my-key" {
			t.Errorf("expected my-key, got %s", res.ApiKey.Name)
		}
	})

	t.Run("UpdateAPIKey with config, expiresAt, rateLimit, actor", func(t *testing.T) {
		expiry := time.Now().Add(24 * time.Hour)
		rl := int32(100)
		actorUUID := uuid.New()
		svc := &testAPIKeyService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updater uuid.UUID) (*APIKeyServiceDataResult, error) {
				return &akResult, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.UpdateAPIKey(ctx, &authv1.UpdateAPIKeyRequest{
			TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(),
			Config: cfgStruct,
			ExpiresAt: timestamppb.New(expiry),
			RateLimit: &rl,
			ActorUserUuid: actorUUID.String(),
			Status: strptr("inactive"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.ApiKey == nil {
			t.Error("expected non-nil api key")
		}
	})

	t.Run("SetAPIKeyStatus success", func(t *testing.T) {
		svc := &testAPIKeyService{
			setStatusByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error) {
				return &akResult, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.SetAPIKeyStatus(ctx, &authv1.SetAPIKeyStatusRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), Status: "inactive"})
		if err != nil {
			t.Fatal(err)
		}
		if res.ApiKey == nil {
			t.Error("expected non-nil api key")
		}
	})

	t.Run("DeleteAPIKey success", func(t *testing.T) {
		svc := &testAPIKeyService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64, deleter uuid.UUID) (*APIKeyServiceDataResult, error) {
				return &akResult, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.DeleteAPIKey(ctx, &authv1.DeleteAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.ApiKey == nil {
			t.Error("expected non-nil api key")
		}
	})

	t.Run("DeleteAPIKey with actor UUID", func(t *testing.T) {
		actorUUID := uuid.New()
		svc := &testAPIKeyService{
			deleteFn: func(ctx context.Context, id uuid.UUID, tenantID int64, deleter uuid.UUID) (*APIKeyServiceDataResult, error) {
				return &akResult, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.DeleteAPIKey(ctx, &authv1.DeleteAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ActorUserUuid: actorUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.ApiKey == nil {
			t.Error("expected non-nil api key")
		}
	})

	t.Run("ListAPIKeyAPIs success", func(t *testing.T) {
		perm := PermissionServiceDataResult{PermissionUUID: permUUID, Name: "read", Description: "Read", Status: "active", CreatedAt: now, UpdatedAt: now}
		apiResult := APIKeyAPIServiceDataResult{
			APIKeyAPIUUID: apiUUID,
			Api:           APIServiceDataResult{APIUUID: apiUUID, Name: "test-api", DisplayName: "Test", Description: "desc", Status: "active", IsSystem: false, CreatedAt: now, UpdatedAt: now},
			Permissions:   []PermissionServiceDataResult{perm},
			CreatedAt:     now,
		}
		svc := &testAPIKeyService{
			getAPIKeyAPIsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error) {
				return &APIKeyAPIServicePaginatedResult{
					Data: []APIKeyAPIServiceDataResult{apiResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.ListAPIKeyAPIs(ctx, &authv1.ListAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Apis) != 1 {
			t.Fatalf("expected 1 api, got %d", len(res.Apis))
		}
		if len(res.Apis[0].Permissions) != 1 {
			t.Fatalf("expected 1 permission, got %d", len(res.Apis[0].Permissions))
		}
	})

	t.Run("RemoveAPIKeyAPI success", func(t *testing.T) {
		svc := &testAPIKeyService{
			removeAPIKeyAPIFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) error { return nil },
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.RemoveAPIKeyAPI(ctx, &authv1.RemoveAPIKeyAPIRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("ListAPIKeyAPIPermissions success", func(t *testing.T) {
		perm := PermissionServiceDataResult{PermissionUUID: permUUID, Name: "read", Description: "Read", Status: "active", CreatedAt: now, UpdatedAt: now}
		svc := &testAPIKeyService{
			getAPIKeyAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) ([]PermissionServiceDataResult, error) {
				return []PermissionServiceDataResult{perm}, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.ListAPIKeyAPIPermissions(ctx, &authv1.ListAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Permissions) != 1 {
			t.Fatalf("expected 1 permission, got %d", len(res.Permissions))
		}
	})

	t.Run("ListAPIKeyAPIPermissions empty", func(t *testing.T) {
		svc := &testAPIKeyService{
			getAPIKeyAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) ([]PermissionServiceDataResult, error) {
				return []PermissionServiceDataResult{}, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.ListAPIKeyAPIPermissions(ctx, &authv1.ListAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Permissions) != 0 {
			t.Fatalf("expected 0 permissions, got %d", len(res.Permissions))
		}
	})

	t.Run("RemoveAPIKeyAPIPermission success", func(t *testing.T) {
		svc := &testAPIKeyService{
			removeAPIKeyAPIPermissionFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permID uuid.UUID) error { return nil },
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.RemoveAPIKeyAPIPermission(ctx, &authv1.RemoveAPIKeyAPIPermissionRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("CreateAPIKey with expiresAt and rateLimit", func(t *testing.T) {
		expiry := time.Now().Add(24 * time.Hour)
		rl := int32(200)
		svc := &testAPIKeyService{
			createFn: func(ctx context.Context, tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error) {
				return &akResult, "raw-key", nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		res, err := h.CreateAPIKey(ctx, &authv1.CreateAPIKeyRequest{
			TenantUuid: tUUID.String(), Name: "my-key", Status: "active",
			ExpiresAt: timestamppb.New(expiry), RateLimit: &rl,
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Key != "raw-key" {
			t.Errorf("expected raw-key, got %s", res.Key)
		}
	})
}

func TestAPIKeyGRPCHandler_AllErrorPaths(t *testing.T) {
	ctx := context.Background()
	tUUID := uuid.New()
	akUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()

	svcErr := errors.New("svc error")
	tenantErr := errors.New("tenant error")

	failResolver := &mockClientTenantResolver{
		getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, tenantErr
		},
	}
	okResolver := &mockClientTenantResolver{}

	t.Run("resolveTenant error via tenant resolver", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(failResolver, &testAPIKeyService{})
		_, err := h.resolveTenant(ctx, tUUID.String())
		assertGRPCErrCode(t, err, codes.Internal)
	})

	t.Run("tenant errors across handlers", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(failResolver, &testAPIKeyService{})
		_, err := h.GetAPIKey(ctx, &authv1.GetAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.GetAPIKeyConfig(ctx, &authv1.GetAPIKeyConfigRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.CreateAPIKey(ctx, &authv1.CreateAPIKeyRequest{TenantUuid: tUUID.String(), Name: "my-key", Status: "active"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.UpdateAPIKey(ctx, &authv1.UpdateAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.SetAPIKeyStatus(ctx, &authv1.SetAPIKeyStatusRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), Status: "inactive"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.DeleteAPIKey(ctx, &authv1.DeleteAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListAPIKeyAPIs(ctx, &authv1.ListAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddAPIKeyAPIs(ctx, &authv1.AddAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuids: []string{apiUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveAPIKeyAPI(ctx, &authv1.RemoveAPIKeyAPIRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListAPIKeyAPIPermissions(ctx, &authv1.ListAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddAPIKeyAPIPermissions(ctx, &authv1.AddAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveAPIKeyAPIPermission(ctx, &authv1.RemoveAPIKeyAPIPermissionRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
	})

	t.Run("UUID parse errors", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(okResolver, &testAPIKeyService{})
		_, err := h.GetAPIKey(ctx, &authv1.GetAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.GetAPIKeyConfig(ctx, &authv1.GetAPIKeyConfigRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.UpdateAPIKey(ctx, &authv1.UpdateAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.SetAPIKeyStatus(ctx, &authv1.SetAPIKeyStatusRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid", Status: "inactive"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.DeleteAPIKey(ctx, &authv1.DeleteAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.ListAPIKeyAPIs(ctx, &authv1.ListAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.AddAPIKeyAPIs(ctx, &authv1.AddAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid", ApiUuids: []string{apiUUID.String()}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.RemoveAPIKeyAPI(ctx, &authv1.RemoveAPIKeyAPIRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid", ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.ListAPIKeyAPIPermissions(ctx, &authv1.ListAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid", ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.AddAPIKeyAPIPermissions(ctx, &authv1.AddAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid", ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.RemoveAPIKeyAPIPermission(ctx, &authv1.RemoveAPIKeyAPIPermissionRequest{TenantUuid: tUUID.String(), ApiKeyUuid: "bad-uuid", ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("API UUID parse errors", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(okResolver, &testAPIKeyService{})
		_, err := h.RemoveAPIKeyAPI(ctx, &authv1.RemoveAPIKeyAPIRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.ListAPIKeyAPIPermissions(ctx, &authv1.ListAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.AddAPIKeyAPIPermissions(ctx, &authv1.AddAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: "bad-uuid", PermissionUuids: []string{permUUID.String()}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.RemoveAPIKeyAPIPermission(ctx, &authv1.RemoveAPIKeyAPIPermissionRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: "bad-uuid", PermissionUuid: permUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("permission UUID parse errors", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(okResolver, &testAPIKeyService{})
		_, err := h.RemoveAPIKeyAPIPermission(ctx, &authv1.RemoveAPIKeyAPIPermissionRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("actor UUID parse errors", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(okResolver, &testAPIKeyService{})
		_, err := h.UpdateAPIKey(ctx, &authv1.UpdateAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ActorUserUuid: "bad-actor"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.DeleteAPIKey(ctx, &authv1.DeleteAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ActorUserUuid: "bad-actor"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid API UUID in list", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(okResolver, &testAPIKeyService{})
		_, err := h.AddAPIKeyAPIs(ctx, &authv1.AddAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuids: []string{"bad-uuid"}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid Permission UUID in list", func(t *testing.T) {
		h := NewAPIKeyGRPCHandler(okResolver, &testAPIKeyService{})
		_, err := h.AddAPIKeyAPIPermissions(ctx, &authv1.AddAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{"bad-uuid"}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("pagination validation error for ListAPIKeys", func(t *testing.T) {
		svc := &testAPIKeyService{
			getFn: func(ctx context.Context, filter APIKeyServiceGetFilter, ru uuid.UUID) (*APIKeyServiceGetResult, error) {
				return nil, nil
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		_, err := h.ListAPIKeys(ctx, &authv1.ListAPIKeysRequest{TenantUuid: tUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10, SortOrder: "bad"}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("service errors for all handlers", func(t *testing.T) {
		permErrSvc := func() error { return svcErr }
		svc := &testAPIKeyService{
			getFn:                       func(ctx context.Context, filter APIKeyServiceGetFilter, ru uuid.UUID) (*APIKeyServiceGetResult, error) { return nil, svcErr },
			getByUUIDFn:                 func(ctx context.Context, id uuid.UUID, tenantID int64, ru uuid.UUID) (*APIKeyServiceDataResult, error) { return nil, svcErr },
			getConfigByUUIDFn:           func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) { return nil, svcErr },
			createFn:                    func(ctx context.Context, tenantID int64, name, description string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status string) (*APIKeyServiceDataResult, string, error) { return nil, "", svcErr },
			updateFn:                    func(ctx context.Context, id uuid.UUID, tenantID int64, name, description *string, config datatypes.JSON, expiresAt *time.Time, rateLimit *int, status *string, updater uuid.UUID) (*APIKeyServiceDataResult, error) { return nil, svcErr },
			setStatusByUUIDFn:           func(ctx context.Context, id uuid.UUID, tenantID int64, status string) (*APIKeyServiceDataResult, error) { return nil, svcErr },
			deleteFn:                    func(ctx context.Context, id uuid.UUID, tenantID int64, deleter uuid.UUID) (*APIKeyServiceDataResult, error) { return nil, svcErr },
			getAPIKeyAPIsFn:             func(ctx context.Context, tenantID int64, id uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error) { return nil, svcErr },
			addAPIKeyAPIsFn:             func(ctx context.Context, tenantID int64, id uuid.UUID, apiUUIDs []uuid.UUID) error { return svcErr },
			removeAPIKeyAPIFn:           func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) error { return svcErr },
			getAPIKeyAPIPermissionsFn:   func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) ([]PermissionServiceDataResult, error) { return nil, svcErr },
			addAPIKeyAPIPermissionsFn:   func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permIDs []uuid.UUID) error { return svcErr },
			removeAPIKeyAPIPermissionFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permID uuid.UUID) error { return svcErr },
		}
		_ = permErrSvc
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		_, err := h.GetAPIKey(ctx, &authv1.GetAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.GetAPIKeyConfig(ctx, &authv1.GetAPIKeyConfigRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.CreateAPIKey(ctx, &authv1.CreateAPIKeyRequest{TenantUuid: tUUID.String(), Name: "my-key", Status: "active"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.UpdateAPIKey(ctx, &authv1.UpdateAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.SetAPIKeyStatus(ctx, &authv1.SetAPIKeyStatusRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), Status: "inactive"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.DeleteAPIKey(ctx, &authv1.DeleteAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListAPIKeyAPIs(ctx, &authv1.ListAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddAPIKeyAPIs(ctx, &authv1.AddAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuids: []string{apiUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveAPIKeyAPI(ctx, &authv1.RemoveAPIKeyAPIRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListAPIKeyAPIPermissions(ctx, &authv1.ListAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddAPIKeyAPIPermissions(ctx, &authv1.AddAPIKeyAPIPermissionsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveAPIKeyAPIPermission(ctx, &authv1.RemoveAPIKeyAPIPermissionRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
	})

	t.Run("GetAPIKey service error", func(t *testing.T) {
		svc := &testAPIKeyService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, ru uuid.UUID) (*APIKeyServiceDataResult, error) {
				return nil, svcErr
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		_, err := h.GetAPIKey(ctx, &authv1.GetAPIKeyRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
	})

	t.Run("ListAPIKeyAPIs service error", func(t *testing.T) {
		svc := &testAPIKeyService{
			getAPIKeyAPIsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, page, limit int, sortBy, sortOrder string) (*APIKeyAPIServicePaginatedResult, error) {
				return nil, svcErr
			},
		}
		h := NewAPIKeyGRPCHandler(okResolver, svc)
		_, err := h.ListAPIKeyAPIs(ctx, &authv1.ListAPIKeyAPIsRequest{TenantUuid: tUUID.String(), ApiKeyUuid: akUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
	})
}

func TestAPIKeyHelpersFull(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("toAPIKeyProto with expiresAt", func(t *testing.T) {
		expiry := now.Add(24 * time.Hour)
		result := &APIKeyServiceDataResult{
			APIKeyUUID: id, Name: "test", Status: "active",
			ExpiresAt: &expiry, CreatedAt: now, UpdatedAt: now,
		}
		proto := toAPIKeyProto(result)
		if proto.ExpiresAt == nil {
			t.Error("expected non-nil ExpiresAt")
		}
	})

	t.Run("toAPIKeyProto with rateLimit", func(t *testing.T) {
		rl := 100
		result := &APIKeyServiceDataResult{
			APIKeyUUID: id, Name: "test", Status: "active",
			RateLimit: &rl, CreatedAt: now, UpdatedAt: now,
		}
		proto := toAPIKeyProto(result)
		if proto.RateLimit == nil {
			t.Error("expected non-nil RateLimit")
		}
	})

	t.Run("toAPIKeyProto nil input", func(t *testing.T) {
		assert := require.New(t)
		assert.Nil(toAPIKeyProto(nil))
	})

	t.Run("jsonToConfigProto empty config", func(t *testing.T) {
		result := jsonToConfigProto(datatypes.JSON(""))
		if result == nil || len(result.Fields) != 0 {
			t.Error("expected empty struct")
		}
	})

	t.Run("jsonToConfigProto invalid JSON", func(t *testing.T) {
		result := jsonToConfigProto(datatypes.JSON("not-json{{{---"))
		if result == nil || len(result.Fields) != 0 {
			t.Error("expected empty struct for invalid JSON")
		}
	})

	t.Run("jsonToConfigProto valid JSON", func(t *testing.T) {
		result := jsonToConfigProto(datatypes.JSON(`{"key":"value"}`))
		if result == nil {
			t.Error("expected non-nil struct")
		}
		if result.Fields["key"].GetStringValue() != "value" {
			t.Error("expected value")
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

func strptr(s string) *string { return &s }
