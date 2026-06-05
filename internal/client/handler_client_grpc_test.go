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

type mockClientTenantResolver struct {
	getByUUIDFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *mockClientTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(ctx, tenantUUID)
	}
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tenantUUID}, nil
}

type testClientService struct {
	getFn                      func(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error)
	getByUUIDFn                func(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error)
	getSecretByUUIDFn          func(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (*ClientSecretServiceDataResult, error)
	getConfigByUUIDFn          func(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	createFn                   func(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*ClientCreateServiceResult, error)
	updateFn                   func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	rotateSecretFn             func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, gracePeriodHours int) (string, error)
	setStatusByUUIDFn          func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	deleteByUUIDFn             func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	createURIFn                func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uri, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	updateURIFn                func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uriUUID uuid.UUID, uri, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	deleteURIFn                func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uriUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	getClientAPIsFn            func(ctx context.Context, tenantID int64, clientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error)
	addClientAPIsFn            func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	removeClientAPIFn          func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID) error
	getClientAPIPermissionsFn  func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	addClientAPIPermissionsFn  func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	removeClientAPIPermissionFn func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
}

func (m *testClientService) Get(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
	return m.getFn(ctx, filter)
}
func (m *testClientService) GetByUUID(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
	return m.getByUUIDFn(ctx, clientUUID, tenantID)
}
func (m *testClientService) GetSecretByUUID(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (*ClientSecretServiceDataResult, error) {
	return m.getSecretByUUIDFn(ctx, clientUUID, tenantID)
}
func (m *testClientService) GetConfigByUUID(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	return m.getConfigByUUIDFn(ctx, clientUUID, tenantID)
}
func (m *testClientService) Create(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, isDefault bool, identityProviderUUID string, actorUserUUID uuid.UUID) (*ClientCreateServiceResult, error) {
	return m.createFn(ctx, tenantID, name, displayName, clientType, domain, config, status, isDefault, identityProviderUUID, actorUserUUID)
}
func (m *testClientService) Update(ctx context.Context, clientUUID uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, isDefault bool, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	return m.updateFn(ctx, clientUUID, tenantID, name, displayName, clientType, domain, config, status, isDefault, actorUserUUID)
}
func (m *testClientService) RotateSecret(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, gracePeriodHours int) (string, error) {
	return m.rotateSecretFn(ctx, clientUUID, tenantID, actorUserUUID, gracePeriodHours)
}
func (m *testClientService) SetStatusByUUID(ctx context.Context, clientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	return m.setStatusByUUIDFn(ctx, clientUUID, tenantID, status, actorUserUUID)
}
func (m *testClientService) DeleteByUUID(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	return m.deleteByUUIDFn(ctx, clientUUID, tenantID, actorUserUUID)
}
func (m *testClientService) CreateURI(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uri, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	return m.createURIFn(ctx, clientUUID, tenantID, uri, uriType, actorUserUUID)
}
func (m *testClientService) UpdateURI(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uriUUID uuid.UUID, uri, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	return m.updateURIFn(ctx, clientUUID, tenantID, uriUUID, uri, uriType, actorUserUUID)
}
func (m *testClientService) DeleteURI(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uriUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	return m.deleteURIFn(ctx, clientUUID, tenantID, uriUUID, actorUserUUID)
}
func (m *testClientService) GetClientAPIs(ctx context.Context, tenantID int64, clientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error) {
	return m.getClientAPIsFn(ctx, tenantID, clientUUID)
}
func (m *testClientService) AddClientAPIs(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUIDs []uuid.UUID) error {
	return m.addClientAPIsFn(ctx, tenantID, clientUUID, apiUUIDs)
}
func (m *testClientService) RemoveClientAPI(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID) error {
	return m.removeClientAPIFn(ctx, tenantID, clientUUID, apiUUID)
}
func (m *testClientService) GetClientAPIPermissions(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	return m.getClientAPIPermissionsFn(ctx, tenantID, clientUUID, apiUUID)
}
func (m *testClientService) AddClientAPIPermissions(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error {
	return m.addClientAPIPermissionsFn(ctx, tenantID, clientUUID, apiUUID, permissionUUIDs)
}
func (m *testClientService) RemoveClientAPIPermission(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error {
	return m.removeClientAPIPermissionFn(ctx, tenantID, clientUUID, apiUUID, permissionUUID)
}

func TestClientGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	clientUUID := uuid.New()
	uriUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	now := time.Now()
	resolver := &mockClientTenantResolver{}

	clientResult := ClientServiceDataResult{
		ClientUUID:  clientUUID,
		Name:        "my-app",
		DisplayName: "My App",
		ClientType:  "public",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	clientCreate := &ClientCreateServiceResult{
		Client:           &clientResult,
		ClientIdentifier: "client-id-123",
		PlaintextSecret:  "secret-123",
	}
	uriResult := ClientURIServiceDataResult{
		ClientURIUUID: uriUUID,
		URI:           "https://example.com/callback",
		Type:          "redirect",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	clientWithURI := clientResult
	clientWithURI.ClientURIs = &[]ClientURIServiceDataResult{uriResult}

	t.Run("list success", func(t *testing.T) {
		svc := &testClientService{
			getFn: func(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
				return &ClientServiceGetResult{Data: []ClientServiceDataResult{clientResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.ListClients(ctx, &authv1.ListClientsRequest{TenantUuid: tenantUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10}})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Clients) != 1 {
			t.Fatalf("expected 1 client, got %d", len(res.Clients))
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testClientService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.GetClient(ctx, &authv1.GetClientRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Client.Name != "my-app" {
			t.Errorf("expected name my-app, got %s", res.Client.Name)
		}
	})

	t.Run("getSecret message", func(t *testing.T) {
		svc := &testClientService{}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.GetClientSecret(ctx, &authv1.GetClientSecretRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("rotateSecret success", func(t *testing.T) {
		svc := &testClientService{
			rotateSecretFn: func(ctx context.Context, id uuid.UUID, tenantID int64, actor uuid.UUID, hours int) (string, error) {
				return "new-secret", nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.RotateClientSecret(ctx, &authv1.RotateClientSecretRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ActorUserUuid: uuid.New().String(), GracePeriodHours: 24})
		if err != nil {
			t.Fatal(err)
		}
		if res.ClientSecret != "new-secret" {
			t.Errorf("expected new-secret, got %s", res.ClientSecret)
		}
	})

	t.Run("getConfig success", func(t *testing.T) {
		svc := &testClientService{
			getConfigByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) {
				return datatypes.JSON(`{"foo":"bar"}`), nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.GetClientConfig(ctx, &authv1.GetClientConfigRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Config == nil {
			t.Error("expected config")
		}
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testClientService{
			createFn: func(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, isDefault bool, ipUUID string, actor uuid.UUID) (*ClientCreateServiceResult, error) {
				return clientCreate, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.CreateClient(ctx, &authv1.CreateClientRequest{TenantUuid: tenantUUID.String(), Name: "my-app", DisplayName: "My App", ClientType: "public", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Credentials.ClientSecret != "secret-123" {
			t.Errorf("expected secret-123, got %s", res.Credentials.ClientSecret)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testClientService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, isDefault bool, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.UpdateClient(ctx, &authv1.UpdateClientRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), Name: "my-app", DisplayName: "My App", ClientType: "public", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Client.Name != "my-app" {
			t.Errorf("expected name my-app, got %s", res.Client.Name)
		}
	})

	t.Run("setStatus success", func(t *testing.T) {
		svc := &testClientService{
			setStatusByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.SetClientStatus(ctx, &authv1.SetClientStatusRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), Status: "inactive"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Client.Name != "my-app" {
			t.Errorf("expected client name")
		}
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testClientService{
			deleteByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.DeleteClient(ctx, &authv1.DeleteClientRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Client.Name != "my-app" {
			t.Errorf("expected client name")
		}
	})

	t.Run("deleteURI success", func(t *testing.T) {
		svc := &testClientService{
			deleteURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uriID uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.DeleteClientURI(ctx, &authv1.DeleteClientURIRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ClientUriUuid: uriUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Client.Name != "my-app" {
			t.Errorf("expected client name")
		}
	})

	t.Run("listURIs success", func(t *testing.T) {
		svc := &testClientService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
				return &clientWithURI, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.ListClientURIs(ctx, &authv1.ListClientURIsRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Uris) != 1 {
			t.Fatalf("expected 1 uri, got %d", len(res.Uris))
		}
	})

	t.Run("createURI success", func(t *testing.T) {
		svc := &testClientService{
			createURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientWithURI, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.CreateClientURI(ctx, &authv1.CreateClientURIRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), Uri: "https://example.com/callback", Type: "redirect"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Uri.Uri != "https://example.com/callback" {
			t.Errorf("expected uri")
		}
	})

	t.Run("updateURI success", func(t *testing.T) {
		svc := &testClientService{
			updateURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientWithURI, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.UpdateClientURI(ctx, &authv1.UpdateClientURIRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ClientUriUuid: uriUUID.String(), Uri: "https://example.com/callback", Type: "redirect"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Uri.Uri != "https://example.com/callback" {
			t.Errorf("expected uri")
		}
	})

	t.Run("listAPIs success", func(t *testing.T) {
		svc := &testClientService{
			getClientAPIsFn: func(ctx context.Context, tenantID int64, id uuid.UUID) ([]ClientAPIServiceDataResult, error) {
				return []ClientAPIServiceDataResult{}, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.ListClientAPIs(ctx, &authv1.ListClientAPIsRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Apis == nil {
			t.Error("expected apis")
		}
	})

	t.Run("addAPIs success", func(t *testing.T) {
		svc := &testClientService{
			addClientAPIsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiUUIDs []uuid.UUID) error {
				return nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.AddClientAPIs(ctx, &authv1.AddClientAPIsRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuids: []string{apiUUID.String()}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("removeAPI success", func(t *testing.T) {
		svc := &testClientService{
			removeClientAPIFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) error {
				return nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.RemoveClientAPI(ctx, &authv1.RemoveClientAPIRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuid: apiUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("listAPIPermissions success", func(t *testing.T) {
		svc := &testClientService{
			getClientAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) ([]PermissionServiceDataResult, error) {
				return []PermissionServiceDataResult{}, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.ListClientAPIPermissions(ctx, &authv1.ListClientAPIPermissionsRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuid: apiUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Permissions == nil {
			t.Error("expected permissions")
		}
	})

	t.Run("addAPIPermissions success", func(t *testing.T) {
		svc := &testClientService{
			addClientAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permIDs []uuid.UUID) error {
				return nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.AddClientAPIPermissions(ctx, &authv1.AddClientAPIPermissionsRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("removeAPIPermission success", func(t *testing.T) {
		svc := &testClientService{
			removeClientAPIPermissionFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permID uuid.UUID) error {
				return nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.RemoveClientAPIPermission(ctx, &authv1.RemoveClientAPIPermissionRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		svc := &testClientService{
			getFn: func(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
				return &ClientServiceGetResult{}, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		_, err := h.ListClients(ctx, &authv1.ListClientsRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
		_, err = h.GetClient(ctx, &authv1.GetClientRequest{TenantUuid: tenantUUID.String(), ClientUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service errors", func(t *testing.T) {
		svcErr := errors.New("db error")
		svc := &testClientService{
			getFn: func(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
				return nil, svcErr
			},
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		_, err := h.ListClients(ctx, &authv1.ListClientsRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
		_, err = h.GetClient(ctx, &authv1.GetClientRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})
}

func TestClientProtoConverters(t *testing.T) {
	id := uuid.New()
	assert := require.New(t)
	assert.Nil(toClientProto(nil))
	assert.Nil(toClientURIProto(nil))
	assert.Nil(toClientAPIPermissionProto(nil))
	c := &ClientServiceDataResult{ClientUUID: id, Name: "test", Status: "active"}
	proto := toClientProto(c)
	assert.Equal("test", proto.Name)
	assert.Equal(id.String(), proto.ClientUuid)
}
