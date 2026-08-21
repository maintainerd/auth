package client

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ===========================================================================
// gRPC handler: service binding + the service-principal actor path
// ===========================================================================

// grpcServiceCallerCtx mirrors what the gRPC interceptor installs for a BARE
// service-principal token (e.g. the core orchestrator's client_credentials
// token): verified claims carrying a `svc` name and a tenant, and NO user in
// the auth context — there is no on_behalf_of human to resolve.
func grpcServiceCallerCtx(tenantID int64, serviceName string) context.Context {
	return middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		TenantID: tenantID,
		Service:  serviceName,
		Sub:      serviceName,
	})
}

// mockSystemClientTenantResolver additionally answers GetSystem, which is what
// lets a system-tenant caller pass the cross-tenant boundary in resolveTenant —
// needed to prove the SERVICE-actor path still pins a bare machine token to its
// own tenant even where a system-tenant USER would be allowed through.
type mockSystemClientTenantResolver struct {
	mockClientTenantResolver
	systemTenantID int64
}

func (m *mockSystemClientTenantResolver) GetSystem(_ context.Context) (*TenantServiceDataResult, error) {
	return &TenantServiceDataResult{TenantID: m.systemTenantID}, nil
}

func TestClientGRPCHandler_ServiceBinding(t *testing.T) {
	resolver := &mockClientTenantResolver{} // resolves every tenant to TenantID 1
	tenantUUID := uuid.New()
	svcUUID := uuid.NewString()
	now := time.Now()

	boundResult := ClientServiceDataResult{
		ClientUUID:  uuid.New(),
		Name:        "core-agent",
		DisplayName: "Core Agent Client",
		ClientType:  shared.ClientTypeM2M,
		Status:      "active",
		ServiceUUID: &svcUUID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	boundCreate := &ClientCreateServiceResult{
		Client:           &boundResult,
		ClientIdentifier: "client-id-123",
		PlaintextSecret:  "secret-123",
	}

	validCreateReq := func() *authv1.CreateClientRequest {
		id := svcUUID
		return &authv1.CreateClientRequest{
			TenantId:    tenantUUID.String(),
			Name:        "core-agent",
			DisplayName: "Core Agent Client",
			ClientType:  shared.ClientTypeM2M,
			Domain:      "svc.example.com",
			Status:      "active",
			ServiceId:   &id,
		}
	}

	t.Run("human actor: the binding travels to the service and the response exposes it", func(t *testing.T) {
		var gotActor ClientActor
		var gotService *string
		svc := &testClientService{
			createFn: func(_ context.Context, _ int64, _, _, _, _ string, _ datatypes.JSON, _ string, _ string, _ *uuid.UUID, _ bool, _ *string, _ *string, _ *bool, _ *bool, actor ClientActor, serviceUUID *string) (*ClientCreateServiceResult, error) {
				gotActor, gotService = actor, serviceUUID
				return boundCreate, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.CreateClient(grpcCallerCtx(1), validCreateReq())
		require.NoError(t, err)
		require.NotNil(t, gotService)
		assert.Equal(t, svcUUID, *gotService)
		require.NotNil(t, gotActor.UserUUID, "a token naming a user stays attributed to that user")
		assert.Empty(t, gotActor.ServiceName)
		assert.Equal(t, svcUUID, res.Client.GetServiceId(), "the response must expose the binding for get-or-create convergence")
	})

	t.Run("service principal: m2m + service-bound create succeeds with no user attribution", func(t *testing.T) {
		var gotActor ClientActor
		svc := &testClientService{
			createFn: func(_ context.Context, _ int64, _, _, _, _ string, _ datatypes.JSON, _ string, _ string, _ *uuid.UUID, _ bool, _ *string, _ *string, _ *bool, _ *bool, actor ClientActor, _ *string) (*ClientCreateServiceResult, error) {
				gotActor = actor
				return boundCreate, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)
		res, err := h.CreateClient(grpcServiceCallerCtx(1, "core"), validCreateReq())
		require.NoError(t, err)
		assert.Nil(t, gotActor.UserUUID, "no user may be fabricated for a machine actor")
		assert.Equal(t, "core", gotActor.ServiceName)
		assert.Equal(t, svcUUID, res.Client.GetServiceId())
	})

	// A lone service token must not be able to mint an UNBOUND credential —
	// the binding is what scopes what the new client can act as.
	t.Run("service principal without a binding is refused with the original error", func(t *testing.T) {
		h := NewClientGRPCHandler(resolver, &testClientService{})
		req := validCreateReq()
		req.ServiceId = nil
		_, err := h.CreateClient(grpcServiceCallerCtx(1, "core"), req)
		assertGRPCErrCode(t, err, codes.PermissionDenied)
		assert.Contains(t, status.Convert(err).Message(), "requires a user principal",
			"the refusal must be the same one the user-actor path always produced")
	})

	// A lone service token must not be able to mint a USER-FACING client either.
	t.Run("service principal with a non-m2m type is refused", func(t *testing.T) {
		h := NewClientGRPCHandler(resolver, &testClientService{})
		req := validCreateReq()
		req.ClientType = shared.ClientTypeSPA
		_, err := h.CreateClient(grpcServiceCallerCtx(1, "core"), req)
		assertGRPCErrCode(t, err, codes.PermissionDenied)
		assert.Contains(t, status.Convert(err).Message(), "requires a user principal")
	})

	// Tenant safety: even a SYSTEM-tenant machine token — which resolveTenant
	// would wave through for a user actor — may not provision clients into
	// another tenant on the bare service-actor path.
	t.Run("service principal from tenant A targeting tenant B is refused", func(t *testing.T) {
		systemResolver := &mockSystemClientTenantResolver{systemTenantID: 2}
		h := NewClientGRPCHandler(systemResolver, &testClientService{})
		_, err := h.CreateClient(grpcServiceCallerCtx(2, "core"), validCreateReq()) // target resolves to tenant 1
		assertGRPCErrCode(t, err, codes.PermissionDenied)
		assert.Contains(t, status.Convert(err).Message(), "own tenant")
	})

	t.Run("update: an absent service_id means unchanged, an explicit one travels through", func(t *testing.T) {
		var gotService *string
		clientResult := boundResult
		svc := &testClientService{
			updateFn: func(_ context.Context, _ uuid.UUID, _ int64, _, _, _, _ string, _ datatypes.JSON, _ string, _ *uuid.UUID, _ *bool, _ *bool, _ *string, _ *string, _ *bool, _ *bool, _ ClientActor, serviceUUID *string) (*ClientServiceDataResult, error) {
				gotService = serviceUUID
				return &clientResult, nil
			},
		}
		h := NewClientGRPCHandler(resolver, svc)

		base := &authv1.UpdateClientRequest{
			TenantId:    tenantUUID.String(),
			ClientId:    boundResult.ClientUUID.String(),
			Name:        "core-agent",
			DisplayName: "Core Agent Client",
			ClientType:  shared.ClientTypeM2M,
			Domain:      "svc.example.com",
			Status:      "active",
		}
		_, err := h.UpdateClient(grpcCallerCtx(1), base)
		require.NoError(t, err)
		assert.Nil(t, gotService, "an absent field must reach the service as nil (= unchanged)")

		id := svcUUID
		base.ServiceId = &id
		_, err = h.UpdateClient(grpcCallerCtx(1), base)
		require.NoError(t, err)
		require.NotNil(t, gotService)
		assert.Equal(t, svcUUID, *gotService)
	})

	// An explicit empty string means "unbind" — a human may do that; a bare
	// service token may not strip the binding that scopes its own credential.
	t.Run("update: an empty service_id as a service principal is refused", func(t *testing.T) {
		h := NewClientGRPCHandler(resolver, &testClientService{})
		empty := ""
		_, err := h.UpdateClient(grpcServiceCallerCtx(1, "core"), &authv1.UpdateClientRequest{
			TenantId:    tenantUUID.String(),
			ClientId:    boundResult.ClientUUID.String(),
			Name:        "core-agent",
			DisplayName: "Core Agent Client",
			ClientType:  shared.ClientTypeM2M,
			Domain:      "svc.example.com",
			Status:      "active",
			ServiceId:   &empty,
		})
		assertGRPCErrCode(t, err, codes.PermissionDenied)
	})

	t.Run("toClientProto exposes the binding and omits it when unbound", func(t *testing.T) {
		proto := toClientProto(&boundResult)
		assert.Equal(t, svcUUID, proto.GetServiceId())

		unbound := boundResult
		unbound.ServiceUUID = nil
		assert.Nil(t, toClientProto(&unbound).ServiceId, "an unbound client must not report a binding")
	})
}

// ===========================================================================
// Service layer: the service-actor rules are enforced again (defense in depth)
// ===========================================================================

func serviceActorIdpRepo(tenantID int64) *mockIdentityProviderRepo {
	return &mockIdentityProviderRepo{
		findByUUIDFn: func(_ any, _ ...string) (*IdentityProvider, error) {
			return &IdentityProvider{IdentityProviderID: 1, Tenant: &Tenant{TenantID: tenantID, IsSystem: true}}, nil
		},
	}
}

func TestClientService_Create_ServiceActor(t *testing.T) {
	tenantID := int64(1)
	serviceUUID := uuid.New()

	newSvc := func(t *testing.T, clientRepo *mockClientRepo) (ClientService, sqlmock.Sqlmock) {
		gormDB, mock := newMockGormDB(t)
		// The user repo would fabricate an actor if consulted; returning nil
		// proves the service-actor path never looks a user up.
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil }}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, serviceActorIdpRepo(tenantID),
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		return svc, mock
	}

	t.Run("refuses a non-m2m client", func(t *testing.T) {
		svc, mock := newSvc(t, &mockClientRepo{})
		mock.ExpectBegin()
		mock.ExpectRollback()
		id := serviceUUID.String()
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", shared.ClientTypeSPA, "example.com", nil, "active", uuid.New().String(), nil, true, nil, nil, nil, nil, ServiceActor("core"), &id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "m2m client bound to a service")
	})

	t.Run("refuses a missing binding", func(t *testing.T) {
		svc, mock := newSvc(t, &mockClientRepo{})
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", shared.ClientTypeM2M, "example.com", nil, "active", uuid.New().String(), nil, true, nil, nil, nil, nil, ServiceActor("core"), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "m2m client bound to a service")
	})

	// Fail closed: a zero actor is neither a user nor a service.
	t.Run("refuses an empty actor entirely", func(t *testing.T) {
		svc, mock := newSvc(t, &mockClientRepo{})
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Create(context.Background(), tenantID, "test", "Test", shared.ClientTypeM2M, "example.com", nil, "active", uuid.New().String(), nil, true, nil, nil, nil, nil, ClientActor{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires an acting principal")
	})

	t.Run("creates a bound m2m client with NULL user attribution", func(t *testing.T) {
		created := clientWithIDP(tenantID)
		created.ClientType = shared.ClientTypeM2M
		clientRepo := &mockClientRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return created, nil },
		}
		svc, mock := newSvc(t, clientRepo)
		mock.ExpectBegin()
		// resolveServiceBinding: same tenant, active, undeleted.
		mock.ExpectQuery(`SELECT \* FROM "services" WHERE service_uuid = \$1 AND tenant_id = \$2 AND deleted_at IS NULL`).
			WithArgs(serviceUUID, tenantID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"service_id", "service_uuid", "tenant_id", "name", "status"}).
				AddRow(int64(7), serviceUUID, tenantID, "core", shared.StatusActive))
		// The IdP connection rows carry NULL created_by/updated_by: there is no
		// user behind the mutation and none may be invented.
		mock.ExpectQuery(`SELECT \* FROM "client_identity_providers" WHERE \(client_id = \$1 AND identity_provider_id = \$2 AND deleted_at IS NULL\).*LIMIT \$3`).
			WithArgs(int64(1), int64(1), 1).
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectQuery(`INSERT INTO "client_identity_providers"`).
			WithArgs(sqlmock.AnyArg(), int64(1), int64(1), int64(1), true, true, 0, nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), nil).
			WillReturnRows(sqlmock.NewRows([]string{"client_identity_provider_id"}).AddRow(int64(1)))
		mock.ExpectCommit()

		id := serviceUUID.String()
		res, err := svc.Create(context.Background(), tenantID, "core-agent", "Core Agent Client", shared.ClientTypeM2M, "svc.example.com", nil, "active", uuid.New().String(), nil, true, nil, nil, nil, nil, ServiceActor("core"), &id)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestClientService_Update_ServiceActor(t *testing.T) {
	tenantID := int64(1)
	cUUID := uuid.New()

	boundServiceID := int64(7)
	boundClient := func() *Client {
		c := clientWithIDP(tenantID)
		c.ClientType = shared.ClientTypeM2M
		c.ServiceID = &boundServiceID
		return c
	}

	newSvc := func(t *testing.T, clientRepo *mockClientRepo) (ClientService, sqlmock.Sqlmock) {
		gormDB, mock := newMockGormDB(t)
		userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil }}
		svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, serviceActorIdpRepo(tenantID),
			&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
			&mockAPIRepo{}, userRepo, &mockTenantRepo{}, nil, nil)
		return svc, mock
	}

	t.Run("refuses changing the type away from m2m", func(t *testing.T) {
		clientRepo := &mockClientRepo{findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return boundClient(), nil }}
		svc, mock := newSvc(t, clientRepo)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), cUUID, tenantID, "test", "Test Name", shared.ClientTypeTraditional, "ex.com", nil, "active", nil, nil, nil, nil, nil, nil, nil, ServiceActor("core"), nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "away from m2m")
	})

	t.Run("refuses unbinding via an explicit empty string", func(t *testing.T) {
		clientRepo := &mockClientRepo{findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return boundClient(), nil }}
		svc, mock := newSvc(t, clientRepo)
		mock.ExpectBegin()
		mock.ExpectRollback()
		empty := ""
		_, err := svc.Update(context.Background(), cUUID, tenantID, "test", "Test Name", shared.ClientTypeM2M, "ex.com", nil, "active", nil, nil, nil, nil, nil, nil, nil, ServiceActor("core"), nil, &empty)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unbind")
	})

	t.Run("refuses updating an UNBOUND client with no binding in the request", func(t *testing.T) {
		unbound := clientWithIDP(tenantID)
		unbound.ClientType = shared.ClientTypeM2M
		clientRepo := &mockClientRepo{findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return unbound, nil }}
		svc, mock := newSvc(t, clientRepo)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := svc.Update(context.Background(), cUUID, tenantID, "test", "Test Name", shared.ClientTypeM2M, "ex.com", nil, "active", nil, nil, nil, nil, nil, nil, nil, ServiceActor("core"), nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bound to a service")
	})

	t.Run("keeps an already-bound client updatable with the binding omitted", func(t *testing.T) {
		clientRepo := &mockClientRepo{findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return boundClient(), nil }}
		svc, mock := newSvc(t, clientRepo)
		mock.ExpectBegin()
		mock.ExpectCommit()
		res, err := svc.Update(context.Background(), cUUID, tenantID, "test", "Test Name", shared.ClientTypeM2M, "ex.com", nil, "active", nil, nil, nil, nil, nil, nil, nil, ServiceActor("core"), nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// The human unbind stays a human capability: an explicit empty string clears the
// binding (resolveServiceBinding returns nil for it), which is what the console
// sends to detach a credential from its service.
func TestClientService_Update_HumanUnbindsWithEmptyString(t *testing.T) {
	tenantID := int64(1)
	cUUID := uuid.New()
	actorUUID := uuid.New()

	boundServiceID := int64(7)
	c := clientWithIDP(tenantID)
	c.ClientType = shared.ClientTypeM2M
	c.ServiceID = &boundServiceID

	var saved *Client
	clientRepo := &mockClientRepo{
		findByUUIDFn: func(_ any, _ ...string) (*Client, error) { return c, nil },
		createOrUpdateFn: func(e *Client) (*Client, error) {
			saved = e
			return e, nil
		},
	}
	gormDB, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()
	svc := NewClientService(gormDB, clientRepo, &mockClientURIRepo{}, serviceActorIdpRepo(tenantID),
		&mockPermissionRepo{}, &mockClientPermissionRepo{}, &mockClientAPIRepo{}, &mockClientRoleRepo{}, &mockRoleRepo{},
		&mockAPIRepo{}, actorUserRepo(tenantID), &mockTenantRepo{}, nil, nil)

	empty := ""
	_, err := svc.Update(context.Background(), cUUID, tenantID, "test", "Test Name", shared.ClientTypeM2M, "ex.com", nil, "active", nil, nil, nil, nil, nil, nil, nil, UserActor(actorUUID), nil, &empty)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Nil(t, saved.ServiceID, "an explicit empty string must clear the binding")
}
