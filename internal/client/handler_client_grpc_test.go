//nolint:staticcheck
package client

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
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
	getFn                       func(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error)
	getByUUIDFn                 func(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error)
	getConfigByUUIDFn           func(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error)
	createFn                    func(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, identityProviderUUID string, brandingUUID *uuid.UUID, allowRegistration bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, serviceUUID *string) (*ClientCreateServiceResult, error)
	updateFn                    func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, brandingUUID *uuid.UUID, allowRegistration *bool, allowMagicLink *bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, serviceUUID *string) (*ClientServiceDataResult, error)
	rotateSecretFn              func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID, gracePeriodHours int) (string, error)
	setStatusByUUIDFn           func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, status string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	deleteByUUIDFn              func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	createURIFn                 func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uri, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	updateURIFn                 func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uriUUID uuid.UUID, uri, uriType string, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	deleteURIFn                 func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, uriUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	getClientAPIsFn             func(ctx context.Context, tenantID int64, clientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error)
	addClientAPIsFn             func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUIDs []uuid.UUID) error
	removeClientAPIFn           func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID) error
	getClientAPIPermissionsFn   func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error)
	addClientAPIPermissionsFn   func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID) error
	removeClientAPIPermissionFn func(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID) error
	getConnectionsFn            func(ctx context.Context, clientUUID uuid.UUID, tenantID int64) ([]ClientIdentityProviderServiceDataResult, error)
	addConnectionFn             func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, identityProviderUUID uuid.UUID, isDefault, enabled bool, displayOrder int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	updateConnectionFn          func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, isDefault, enabled *bool, displayOrder *int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
	removeConnectionFn          func(ctx context.Context, clientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error)
}

func (m *testClientService) IsManagementClient(context.Context, string) bool { return false }
func (m *testClientService) IsFirstPartyClient(context.Context, string) bool { return false }

// No certificate binding by default: these tests exercise the handlers, and the
// binding is enforced in the gRPC interceptor, not here.
func (m *testClientService) BoundCertThumbprint(context.Context, string) string { return "" }

func (m *testClientService) Get(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
	return m.getFn(ctx, filter)
}
func (m *testClientService) GetByUUID(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
	return m.getByUUIDFn(ctx, clientUUID, tenantID)
}
func (m *testClientService) GetConfigByUUID(ctx context.Context, clientUUID uuid.UUID, tenantID int64) (datatypes.JSON, error) {
	return m.getConfigByUUIDFn(ctx, clientUUID, tenantID)
}
func (m *testClientService) Create(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, identityProviderUUID string, brandingUUID *uuid.UUID, allowRegistration bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, serviceUUID *string) (*ClientCreateServiceResult, error) {
	return m.createFn(ctx, tenantID, name, displayName, clientType, domain, config, status, identityProviderUUID, brandingUUID, allowRegistration, backchannelLogoutURI, frontchannelLogoutURI, backchannelLogoutSessionRequired, dPoPRequired, actorUserUUID, serviceUUID)
}
func (m *testClientService) Update(ctx context.Context, clientUUID uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, brandingUUID *uuid.UUID, allowRegistration *bool, allowMagicLink *bool, backchannelLogoutURI *string, frontchannelLogoutURI *string, backchannelLogoutSessionRequired *bool, dPoPRequired *bool, actorUserUUID uuid.UUID, expectedUpdatedAt *time.Time, serviceUUID *string) (*ClientServiceDataResult, error) {
	return m.updateFn(ctx, clientUUID, tenantID, name, displayName, clientType, domain, config, status, brandingUUID, allowRegistration, allowMagicLink, backchannelLogoutURI, frontchannelLogoutURI, backchannelLogoutSessionRequired, dPoPRequired, actorUserUUID, serviceUUID)
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
func (m *testClientService) GetConnections(ctx context.Context, clientUUID uuid.UUID, tenantID int64) ([]ClientIdentityProviderServiceDataResult, error) {
	if m.getConnectionsFn != nil {
		return m.getConnectionsFn(ctx, clientUUID, tenantID)
	}
	return nil, nil
}
func (m *testClientService) AddConnection(ctx context.Context, clientUUID uuid.UUID, tenantID int64, identityProviderUUID uuid.UUID, isDefault, enabled bool, displayOrder int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	if m.addConnectionFn != nil {
		return m.addConnectionFn(ctx, clientUUID, tenantID, identityProviderUUID, isDefault, enabled, displayOrder, actorUserUUID)
	}
	return nil, nil
}
func (m *testClientService) UpdateConnection(ctx context.Context, clientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, isDefault, enabled *bool, displayOrder *int, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	if m.updateConnectionFn != nil {
		return m.updateConnectionFn(ctx, clientUUID, tenantID, connectionUUID, isDefault, enabled, displayOrder, actorUserUUID)
	}
	return nil, nil
}
func (m *testClientService) RemoveConnection(ctx context.Context, clientUUID uuid.UUID, tenantID int64, connectionUUID uuid.UUID, actorUserUUID uuid.UUID) (*ClientServiceDataResult, error) {
	if m.removeConnectionFn != nil {
		return m.removeConnectionFn(ctx, clientUUID, tenantID, connectionUUID, actorUserUUID)
	}
	return nil, nil
}
func (m *testClientService) GetClientAPIs(ctx context.Context, tenantID int64, clientUUID uuid.UUID) ([]ClientAPIServiceDataResult, error) {
	return m.getClientAPIsFn(ctx, tenantID, clientUUID)
}
func (m *testClientService) AddClientAPIs(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUIDs []uuid.UUID, actorUUID uuid.UUID) error {
	return m.addClientAPIsFn(ctx, tenantID, clientUUID, apiUUIDs)
}
func (m *testClientService) RemoveClientAPI(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, actorUUID uuid.UUID) error {
	return m.removeClientAPIFn(ctx, tenantID, clientUUID, apiUUID)
}
func (m *testClientService) GetClientAPIPermissions(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID) ([]PermissionServiceDataResult, error) {
	return m.getClientAPIPermissionsFn(ctx, tenantID, clientUUID, apiUUID)
}
func (m *testClientService) AddClientAPIPermissions(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUIDs []uuid.UUID, actorUUID uuid.UUID) error {
	return m.addClientAPIPermissionsFn(ctx, tenantID, clientUUID, apiUUID, permissionUUIDs)
}
func (m *testClientService) RemoveClientAPIPermission(ctx context.Context, tenantID int64, clientUUID uuid.UUID, apiUUID uuid.UUID, permissionUUID uuid.UUID, actorUUID uuid.UUID) error {
	return m.removeClientAPIPermissionFn(ctx, tenantID, clientUUID, apiUUID, permissionUUID)
}

// grpcCallerCtx gives a test the same thing the interceptor gives production: a
// tenant-bound token carrying a USER identity. The handlers now refuse a
// body-supplied actor and fail closed without one, so a bare context.Background()
// correctly yields PermissionDenied.
func grpcCallerCtx(tenantID int64) context.Context {
	return grpcCallerCtxAs(tenantID, uuid.New())
}

// grpcCallerCtxAs mirrors what the gRPC interceptor actually installs: BOTH the
// verified claims and the resolved AuthContext.
//
// The handlers used to fall back to raw JWT claims when the AuthContext was
// empty, which let this surface accept a token the tenant surface refused. That
// fallback is gone — the actor now comes only from the interceptor-populated
// AuthContext — so a fixture that sets claims alone no longer authenticates,
// which is exactly the production behaviour.
func grpcCallerCtxAs(tenantID int64, actor uuid.UUID) context.Context {
	// Sub carries the same value as UserUUID, as it does on a real token.
	ctx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		TenantID: tenantID,
		UserUUID: actor,
		Sub:      actor.String(),
	})
	return middleware.WithAuthContextValue(ctx, &authctx.AuthContext{
		User:   &authctx.AuthUser{UserUUID: actor},
		Tenant: &authctx.AuthTenant{TenantID: tenantID},
	})
}

func TestClientGRPCHandler_RPCS(t *testing.T) {
	// mockClientTenantResolver resolves every tenant to TenantID 1.
	ctx := grpcCallerCtx(1)
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

	// This used to assert err == nil and a non-empty Message — i.e. that
	// GetClientSecret SUCCEEDS. A generated client then hands its caller an empty
	// ClientSecret with no error, which reads as "this client has no secret".
	// Secrets are bcrypt hashed and unreadable, so the only honest answer is
	// UNIMPLEMENTED.
	t.Run("getSecret is unimplemented, not a successful empty response", func(t *testing.T) {
		svc := &testClientService{}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.GetClientSecret(ctx, &authv1.GetClientSecretRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String()})
		if err == nil {
			t.Fatal("expected an error")
		}
		if res != nil {
			t.Error("expected no response body")
		}
		if got := status.Code(err); got != codes.Unimplemented {
			t.Errorf("expected Unimplemented, got %v", got)
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
			createFn: func(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, ipUUID string, _ *uuid.UUID, _ bool, _ *string, _ *string, _ *bool, _ *bool, actor uuid.UUID, _ *string) (*ClientCreateServiceResult, error) {
				return clientCreate, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.CreateClient(ctx, &authv1.CreateClientRequest{TenantUuid: tenantUUID.String(), Name: "my-app", DisplayName: "My Application", ClientType: "spa", Domain: "app.example.com", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Credentials.ClientSecret != "secret-123" {
			t.Errorf("expected secret-123, got %s", res.Credentials.ClientSecret)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testClientService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, _ *uuid.UUID, _ *bool, _ *bool, _ *string, _ *string, _ *bool, _ *bool, actor uuid.UUID, _ *string) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.UpdateClient(ctx, &authv1.UpdateClientRequest{TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), Name: "my-app", DisplayName: "My Application", ClientType: "spa", Domain: "app.example.com", Status: "active"})
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
		res, err := h.AddClientAPIs(ctx, &authv1.AddClientAPIsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuids: []string{apiUUID.String()}})
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
		res, err := h.RemoveClientAPI(ctx, &authv1.RemoveClientAPIRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuid: apiUUID.String()})
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
		res, err := h.AddClientAPIPermissions(ctx, &authv1.AddClientAPIPermissionsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
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
		res, err := h.RemoveClientAPIPermission(ctx, &authv1.RemoveClientAPIPermissionRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tenantUUID.String(), ClientUuid: clientUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
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

func assertGRPCErrCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	if code := status.Code(err); code != expected {
		t.Errorf("expected %v, got %v", expected, code)
	}
}

func TestClientGRPCHandler_AllErrorPaths(t *testing.T) {
	ctx := grpcCallerCtx(1)
	tUUID := uuid.New()
	cUUID := uuid.New()
	uriUUID := uuid.New()
	apiUUID := uuid.New()
	permUUID := uuid.New()
	now := time.Now()

	svcErr := errors.New("svc error")
	tenantErr := errors.New("tenant error")

	failResolver := &mockClientTenantResolver{
		getByUUIDFn: func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, tenantErr
		},
	}
	okResolver := &mockClientTenantResolver{}

	clientResult := ClientServiceDataResult{
		ClientUUID: cUUID, Name: "my-app", DisplayName: "My App",
		ClientType: "public", Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	clientCreate := &ClientCreateServiceResult{
		Client: &clientResult, ClientIdentifier: "client-id-123", PlaintextSecret: "secret-123",
	}
	uriResult := ClientURIServiceDataResult{
		ClientURIUUID: uriUUID, URI: "https://example.com", Type: "redirect",
		CreatedAt: now, UpdatedAt: now,
	}
	clientWithURI := clientResult
	uris := []ClientURIServiceDataResult{uriResult}
	clientWithURI.ClientURIs = &uris

	cfgStruct, _ := structpb.NewStruct(map[string]any{"foo": "bar"})

	t.Run("resolveTenant error on tenant resolver failure", func(t *testing.T) {
		h := NewClientGRPCHandler(failResolver, &testClientService{})
		_, err := h.resolveTenant(ctx, tUUID.String())
		assertGRPCErrCode(t, err, codes.Internal)
	})

	t.Run("tenant resolver errors across handlers", func(t *testing.T) {
		h := NewClientGRPCHandler(failResolver, &testClientService{})
		_, err := h.ListClients(ctx, &authv1.ListClientsRequest{TenantUuid: tUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.GetClient(ctx, &authv1.GetClientRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RotateClientSecret(ctx, &authv1.RotateClientSecretRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.GetClientConfig(ctx, &authv1.GetClientConfigRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.CreateClient(ctx, &authv1.CreateClientRequest{TenantUuid: tUUID.String(), Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Status: "active", IdentityProviderUuid: uuid.New().String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.UpdateClient(ctx, &authv1.UpdateClientRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Status: "active"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.SetClientStatus(ctx, &authv1.SetClientStatusRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Status: "inactive"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.DeleteClient(ctx, &authv1.DeleteClientRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListClientURIs(ctx, &authv1.ListClientURIsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.CreateClientURI(ctx, &authv1.CreateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Uri: "https://example.com", Type: "redirect"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.UpdateClientURI(ctx, &authv1.UpdateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ClientUriUuid: uriUUID.String(), Uri: "https://example.com", Type: "redirect"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.DeleteClientURI(ctx, &authv1.DeleteClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ClientUriUuid: uriUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListClientAPIs(ctx, &authv1.ListClientAPIsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddClientAPIs(ctx, &authv1.AddClientAPIsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuids: []string{apiUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveClientAPI(ctx, &authv1.RemoveClientAPIRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListClientAPIPermissions(ctx, &authv1.ListClientAPIPermissionsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddClientAPIPermissions(ctx, &authv1.AddClientAPIPermissionsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveClientAPIPermission(ctx, &authv1.RemoveClientAPIPermissionRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
	})

	t.Run("UUID parse errors", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.GetClient(ctx, &authv1.GetClientRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.RotateClientSecret(ctx, &authv1.RotateClientSecretRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.GetClientConfig(ctx, &authv1.GetClientConfigRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.UpdateClient(ctx, &authv1.UpdateClientRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Status: "active"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.SetClientStatus(ctx, &authv1.SetClientStatusRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", Status: "inactive"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.DeleteClient(ctx, &authv1.DeleteClientRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.ListClientURIs(ctx, &authv1.ListClientURIsRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.CreateClientURI(ctx, &authv1.CreateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", Uri: "https://example.com", Type: "redirect"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.UpdateClientURI(ctx, &authv1.UpdateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", ClientUriUuid: uriUUID.String(), Uri: "https://example.com", Type: "redirect"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.DeleteClientURI(ctx, &authv1.DeleteClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", ClientUriUuid: uriUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.ListClientAPIs(ctx, &authv1.ListClientAPIsRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.AddClientAPIs(ctx, &authv1.AddClientAPIsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", ApiUuids: []string{apiUUID.String()}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.RemoveClientAPI(ctx, &authv1.RemoveClientAPIRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.ListClientAPIPermissions(ctx, &authv1.ListClientAPIPermissionsRequest{TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.AddClientAPIPermissions(ctx, &authv1.AddClientAPIPermissionsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
		_, err = h.RemoveClientAPIPermission(ctx, &authv1.RemoveClientAPIPermissionRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: "bad-uuid", ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	// INVERTED. This used to assert that a malformed body `actor_user_uuid`
	// produced InvalidArgument — i.e. that the field was parsed and trusted. That
	// was the vulnerability: the body actor drove both audit attribution and the
	// tenant-access check, so naming a user in another tenant satisfied it. The
	// field is now ignored entirely and the actor comes from the verified token,
	// so garbage in the body must change nothing.
	t.Run("a body-supplied actor_user_uuid is ignored", func(t *testing.T) {
		var gotActor uuid.UUID
		svc := &testClientService{
			setStatusByUUIDFn: func(_ context.Context, _ uuid.UUID, _ int64, _ string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				gotActor = actor
				return &ClientServiceDataResult{ClientUUID: cUUID}, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)

		callerUUID := uuid.New()
		callerCtx := grpcCallerCtxAs(1, callerUUID)

		_, err := h.SetClientStatus(callerCtx, &authv1.SetClientStatusRequest{
			TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Status: "inactive",
			ActorUserUuid: "bad-actor-uuid",
		})
		if err != nil {
			t.Fatalf("a malformed body actor must not fail the call: %v", err)
		}
		if gotActor != callerUUID {
			t.Fatalf("actor must come from the token (%s), got %s", callerUUID, gotActor)
		}
	})

	// A caller with no user identity — every service token — must be refused.
	t.Run("a token with no user identity cannot mutate a client", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		serviceCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
			TenantID: 1,
			Service:  "control-plane",
		})
		_, err := h.SetClientStatus(serviceCtx, &authv1.SetClientStatusRequest{
			TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Status: "inactive",
		})
		assertGRPCErrCode(t, err, codes.PermissionDenied)
	})

	t.Run("service errors for all handlers", func(t *testing.T) {
		svc := &testClientService{
			getFn: func(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
				return nil, svcErr
			},
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
			rotateSecretFn: func(ctx context.Context, id uuid.UUID, tenantID int64, actor uuid.UUID, hours int) (string, error) {
				return "", svcErr
			},
			getConfigByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) { return nil, svcErr },
			createFn: func(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, ipUUID string, _ *uuid.UUID, _ bool, _ *string, _ *string, _ *bool, _ *bool, actor uuid.UUID, _ *string) (*ClientCreateServiceResult, error) {
				return nil, svcErr
			},
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, _ *uuid.UUID, _ *bool, _ *bool, _ *string, _ *string, _ *bool, _ *bool, actor uuid.UUID, _ *string) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
			setStatusByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, status string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
			deleteByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
			createURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
			updateURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
			deleteURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uriID uuid.UUID, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return nil, svcErr
			},
			getClientAPIsFn: func(ctx context.Context, tenantID int64, id uuid.UUID) ([]ClientAPIServiceDataResult, error) {
				return nil, svcErr
			},
			addClientAPIsFn:   func(ctx context.Context, tenantID int64, id uuid.UUID, apiUUIDs []uuid.UUID) error { return svcErr },
			removeClientAPIFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) error { return svcErr },
			getClientAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) ([]PermissionServiceDataResult, error) {
				return nil, svcErr
			},
			addClientAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permIDs []uuid.UUID) error {
				return svcErr
			},
			removeClientAPIPermissionFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID, permID uuid.UUID) error {
				return svcErr
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		_, err := h.RotateClientSecret(ctx, &authv1.RotateClientSecretRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.GetClientConfig(ctx, &authv1.GetClientConfigRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.CreateClient(ctx, &authv1.CreateClientRequest{TenantUuid: tUUID.String(), Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Status: "active", IdentityProviderUuid: uuid.New().String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.UpdateClient(ctx, &authv1.UpdateClientRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Status: "active"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.SetClientStatus(ctx, &authv1.SetClientStatusRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Status: "inactive"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.DeleteClient(ctx, &authv1.DeleteClientRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListClientURIs(ctx, &authv1.ListClientURIsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.CreateClientURI(ctx, &authv1.CreateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Uri: "https://example.com", Type: "redirect"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.UpdateClientURI(ctx, &authv1.UpdateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ClientUriUuid: uriUUID.String(), Uri: "https://example.com", Type: "redirect"})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.DeleteClientURI(ctx, &authv1.DeleteClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ClientUriUuid: uriUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListClientAPIs(ctx, &authv1.ListClientAPIsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddClientAPIs(ctx, &authv1.AddClientAPIsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuids: []string{apiUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveClientAPI(ctx, &authv1.RemoveClientAPIRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.ListClientAPIPermissions(ctx, &authv1.ListClientAPIPermissionsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.AddClientAPIPermissions(ctx, &authv1.AddClientAPIPermissionsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{permUUID.String()}})
		assertGRPCErrCode(t, err, codes.Internal)
		_, err = h.RemoveClientAPIPermission(ctx, &authv1.RemoveClientAPIPermissionRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: permUUID.String()})
		assertGRPCErrCode(t, err, codes.Internal)
	})

	t.Run("pagination validation error", func(t *testing.T) {
		svc := &testClientService{
			getFn: func(ctx context.Context, filter ClientServiceGetFilter) (*ClientServiceGetResult, error) {
				return nil, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		_, err := h.ListClients(ctx, &authv1.ListClientsRequest{TenantUuid: tUUID.String(), Pagination: &authv1.Pagination{Page: 1, Limit: 10, SortOrder: "bad", SortBy: "created_at"}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid API UUID in AddClientAPIs list", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.AddClientAPIs(ctx, &authv1.AddClientAPIsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuids: []string{"bad-api-uuid"}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid Permission UUID in AddClientAPIPermissions list", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.AddClientAPIPermissions(ctx, &authv1.AddClientAPIPermissionsRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String(), PermissionUuids: []string{"bad-perm-uuid"}})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid URI UUID in UpdateClientURI", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.UpdateClientURI(ctx, &authv1.UpdateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ClientUriUuid: "bad-uri-uuid", Uri: "https://example.com", Type: "redirect"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid URI UUID in DeleteClientURI", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.DeleteClientURI(ctx, &authv1.DeleteClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ClientUriUuid: "bad-uri-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid API UUID in RemoveClientAPI", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.RemoveClientAPI(ctx, &authv1.RemoveClientAPIRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: "bad-api-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid API UUID in ListClientAPIPermissions", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.ListClientAPIPermissions(ctx, &authv1.ListClientAPIPermissionsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: "bad-api-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid perm UUID in RemoveClientAPIPermission", func(t *testing.T) {
		h := NewClientGRPCHandler(okResolver, &testClientService{})
		_, err := h.RemoveClientAPIPermission(ctx, &authv1.RemoveClientAPIPermissionRequest{ActorUserUuid: uuid.New().String(), TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String(), PermissionUuid: "bad-perm-uuid"})
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("createClientURI with empty URIs result", func(t *testing.T) {
		svc := &testClientService{
			createURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &ClientServiceDataResult{ClientUUID: cUUID}, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.CreateClientURI(ctx, &authv1.CreateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Uri: "https://example.com", Type: "redirect"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Uri != nil {
			t.Error("expected nil uri for empty result")
		}
	})

	t.Run("createClientURI with non-empty URIs", func(t *testing.T) {
		svc := &testClientService{
			createURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientWithURI, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.CreateClientURI(ctx, &authv1.CreateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Uri: "https://example.com", Type: "redirect"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Uri == nil {
			t.Error("expected non-nil uri")
		}
	})

	t.Run("updateClientURI with URI not found in result", func(t *testing.T) {
		differentUUID := uuid.New()
		diffURI := ClientURIServiceDataResult{ClientURIUUID: differentUUID, URI: "https://other.com", Type: "origin", CreatedAt: now, UpdatedAt: now}
		clientWithDiffURI := clientResult
		diffURIs := []ClientURIServiceDataResult{diffURI}
		clientWithDiffURI.ClientURIs = &diffURIs

		svc := &testClientService{
			updateURIFn: func(ctx context.Context, id uuid.UUID, tenantID int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*ClientServiceDataResult, error) {
				return &clientWithDiffURI, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.UpdateClientURI(ctx, &authv1.UpdateClientURIRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ClientUriUuid: uriUUID.String(), Uri: "https://updated.com", Type: "redirect"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Uri != nil {
			t.Error("expected nil uri when URI UUID not found in response")
		}
	})

	t.Run("getClientConfig with invalid JSON returns nil struct", func(t *testing.T) {
		svc := &testClientService{
			getConfigByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) {
				return datatypes.JSON("not-valid-json{{{"), nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.GetClientConfig(ctx, &authv1.GetClientConfigRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Config == nil {
			t.Error("expected non-nil config struct")
		}
	})

	t.Run("getClientConfig with empty config", func(t *testing.T) {
		svc := &testClientService{
			getConfigByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (datatypes.JSON, error) {
				return datatypes.JSON(""), nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.GetClientConfig(ctx, &authv1.GetClientConfigRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Config == nil {
			t.Error("expected non-nil config proto even for empty config")
		}
	})

	t.Run("listClientAPIs with populated permissions", func(t *testing.T) {
		perm := PermissionServiceDataResult{PermissionUUID: permUUID, Name: "read", Description: "Read access", Status: "active", CreatedAt: now, UpdatedAt: now}
		apiResult := ClientAPIServiceDataResult{
			ClientAPIUUID: apiUUID,
			Api:           APIServiceDataResult{APIUUID: apiUUID, Name: "test-api", DisplayName: "Test API", Description: "Test", Status: "active", IsSystem: false, CreatedAt: now, UpdatedAt: now},
			Permissions:   []PermissionServiceDataResult{perm},
			CreatedAt:     now,
		}
		svc := &testClientService{
			getClientAPIsFn: func(ctx context.Context, tenantID int64, id uuid.UUID) ([]ClientAPIServiceDataResult, error) {
				return []ClientAPIServiceDataResult{apiResult}, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.ListClientAPIs(ctx, &authv1.ListClientAPIsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
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

	t.Run("listClientAPIPermissions with populated data", func(t *testing.T) {
		perm := PermissionServiceDataResult{PermissionUUID: permUUID, Name: "read", Description: "Read access", Status: "active", CreatedAt: now, UpdatedAt: now}
		svc := &testClientService{
			getClientAPIPermissionsFn: func(ctx context.Context, tenantID int64, id uuid.UUID, apiID uuid.UUID) ([]PermissionServiceDataResult, error) {
				return []PermissionServiceDataResult{perm}, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.ListClientAPIPermissions(ctx, &authv1.ListClientAPIPermissionsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), ApiUuid: apiUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Permissions) != 1 {
			t.Fatalf("expected 1 perm, got %d", len(res.Permissions))
		}
	})

	t.Run("listClientURIs with nil ClientURIs", func(t *testing.T) {
		svc := &testClientService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.ListClientURIs(ctx, &authv1.ListClientURIsRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Uris) != 0 {
			t.Fatalf("expected 0 uris for nil ClientURIs, got %d", len(res.Uris))
		}
	})

	t.Run("createClient with config", func(t *testing.T) {
		svc := &testClientService{
			createFn: func(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, ipUUID string, _ *uuid.UUID, _ bool, _ *string, _ *string, _ *bool, _ *bool, actor uuid.UUID, _ *string) (*ClientCreateServiceResult, error) {
				return clientCreate, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.CreateClient(ctx, &authv1.CreateClientRequest{TenantUuid: tUUID.String(), Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Config: cfgStruct, Status: "active", IdentityProviderUuid: uuid.New().String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Credentials.ClientSecret != "secret-123" {
			t.Errorf("expected secret-123, got %s", res.Credentials.ClientSecret)
		}
	})

	t.Run("updateClient with config", func(t *testing.T) {
		svc := &testClientService{
			updateFn: func(ctx context.Context, id uuid.UUID, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, _ *uuid.UUID, _ *bool, _ *bool, _ *string, _ *string, _ *bool, _ *bool, actor uuid.UUID, _ *string) (*ClientServiceDataResult, error) {
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		res, err := h.UpdateClient(ctx, &authv1.UpdateClientRequest{TenantUuid: tUUID.String(), ClientUuid: cUUID.String(), Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Config: cfgStruct, Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Client.Name != "my-app" {
			t.Errorf("expected my-app, got %s", res.Client.Name)
		}
	})

	// INVERTED: the actor is the CALLER from the verified token, not whoever the
	// request body names. The body value here is a decoy and must be ignored.
	t.Run("createClient attributes to the token's user, not the body", func(t *testing.T) {
		callerUUID := uuid.New()
		decoyUUID := uuid.New()
		svc := &testClientService{
			createFn: func(ctx context.Context, tenantID int64, name, displayName, clientType, domain string, config datatypes.JSON, status string, ipUUID string, _ *uuid.UUID, _ bool, _ *string, _ *string, _ *bool, _ *bool, actor uuid.UUID, _ *string) (*ClientCreateServiceResult, error) {
				if actor != callerUUID {
					t.Errorf("expected the token's user %s, got %s", callerUUID, actor)
				}
				return clientCreate, nil
			},
		}
		h := NewClientGRPCHandler(okResolver, svc)
		callerCtx := grpcCallerCtxAs(1, callerUUID)
		_, err := h.CreateClient(callerCtx, &authv1.CreateClientRequest{TenantUuid: tUUID.String(), Name: "app", DisplayName: "My Application", ClientType: "spa", Domain: "example.com", Status: "active", IdentityProviderUuid: uuid.New().String(), ActorUserUuid: decoyUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestClientHelpersFull(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	t.Run("parseUUID empty string", func(t *testing.T) {
		_, err := parseUUID("", "Test")
		assertGRPCErrCode(t, err, codes.InvalidArgument)
	})

	t.Run("optionalStr non-empty", func(t *testing.T) {
		result := optionalStr("hello")
		if result == nil || *result != "hello" {
			t.Error("expected pointer to 'hello'")
		}
	})

	t.Run("grpcPagination zero page and limit", func(t *testing.T) {
		dto := grpcPagination(&authv1.Pagination{Page: 0, Limit: 0})
		if dto.Page != 1 {
			t.Errorf("expected page 1, got %d", dto.Page)
		}
		if dto.Limit != 10 {
			t.Errorf("expected limit 10, got %d", dto.Limit)
		}
	})

	t.Run("grpcPagination nil request", func(t *testing.T) {
		dto := grpcPagination(nil)
		if dto.Page != 1 || dto.Limit != 10 {
			t.Error("expected defaults")
		}
	})

	t.Run("structProtoToMap non-nil", func(t *testing.T) {
		s, _ := structpb.NewStruct(map[string]any{"key": "value"})
		result := structProtoToMap(s)
		if result == nil || result["key"] != "value" {
			t.Error("expected map with key=value")
		}
	})

	t.Run("structProtoToMap nil", func(t *testing.T) {
		if structProtoToMap(nil) != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("mapToJSON non-empty success", func(t *testing.T) {
		result, err := mapToJSON(map[string]any{"a": "b"})
		if err != nil {
			t.Fatal(err)
		}
		if string(result) != `{"a":"b"}` {
			t.Errorf("expected {\"a\":\"b\"}, got %s", string(result))
		}
	})

	t.Run("mapToJSON with NaN returns error", func(t *testing.T) {
		_, err := mapToJSON(map[string]any{"x": math.NaN()})
		if err == nil {
			t.Error("expected error for NaN value")
		}
	})

	t.Run("mapToJSON nil map", func(t *testing.T) {
		result, err := mapToJSON(nil)
		if err != nil {
			t.Fatal(err)
		}
		if result != nil {
			t.Error("expected nil result for nil map")
		}
	})

	t.Run("mapToJSON empty map", func(t *testing.T) {
		result, err := mapToJSON(map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		if result != nil {
			t.Error("expected nil result for empty map")
		}
	})

	t.Run("toClientProto with IdentityProvider", func(t *testing.T) {
		idp := IdentityProviderServiceDataResult{
			IdentityProviderUUID: id, Name: "google", DisplayName: "Google",
			Provider: "google", ProviderType: "oidc", Identifier: "google-id",
			Status: "active", IsSystem: false,
			CreatedAt: now, UpdatedAt: now,
		}
		result := &ClientServiceDataResult{
			ClientUUID: id, Name: "test", Status: "active",
			IdentityProvider: &idp, CreatedAt: now, UpdatedAt: now,
		}
		proto := toClientProto(result)
		if proto.IdentityProvider == nil {
			t.Error("expected non-nil identity provider")
		}
		if proto.IdentityProvider.Name != "google" {
			t.Errorf("expected google, got %s", proto.IdentityProvider.Name)
		}
	})

	t.Run("toClientProto with ClientURIs", func(t *testing.T) {
		uri := ClientURIServiceDataResult{ClientURIUUID: id, URI: "https://example.com", Type: "redirect", CreatedAt: now, UpdatedAt: now}
		uris := []ClientURIServiceDataResult{uri}
		result := &ClientServiceDataResult{
			ClientUUID: id, Name: "test", Status: "active",
			ClientURIs: &uris, CreatedAt: now, UpdatedAt: now,
		}
		proto := toClientProto(result)
		if len(proto.Uris) != 1 {
			t.Fatalf("expected 1 uri, got %d", len(proto.Uris))
		}
	})

	t.Run("toClientProto with non-nil Domain", func(t *testing.T) {
		domain := "example.com"
		result := &ClientServiceDataResult{
			ClientUUID: id, Name: "test", Status: "active",
			Domain: &domain, CreatedAt: now, UpdatedAt: now,
		}
		proto := toClientProto(result)
		if proto.Domain != "example.com" {
			t.Errorf("expected example.com, got %s", proto.Domain)
		}
	})

	t.Run("toClientAPIPermissionProto non-nil", func(t *testing.T) {
		perm := &PermissionServiceDataResult{
			PermissionUUID: id, Name: "read", Description: "Read access",
			Status: "active", IsSystem: false,
			CreatedAt: now, UpdatedAt: now,
		}
		proto := toClientAPIPermissionProto(perm)
		if proto == nil {
			t.Fatal("expected non-nil proto")
		}
		if proto.Name != "read" {
			t.Errorf("expected read, got %s", proto.Name)
		}
	})

	t.Run("stringPtr non-nil", func(t *testing.T) {
		s := "test"
		if stringPtr(&s) != "test" {
			t.Error("expected 'test'")
		}
	})

	t.Run("stringPtr nil", func(t *testing.T) {
		if stringPtr(nil) != "" {
			t.Error("expected empty string")
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

func (m *testClientService) AssignClientRole(context.Context, uuid.UUID, uuid.UUID, int64, uuid.UUID) (*ClientRole, error) {
	return nil, nil
}
func (m *testClientService) RemoveClientRole(context.Context, uuid.UUID, uuid.UUID, int64, uuid.UUID) error {
	return nil
}
func (m *testClientService) ListClientRoles(context.Context, uuid.UUID, int64) ([]ClientRole, error) {
	return nil, nil
}
