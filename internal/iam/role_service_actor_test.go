package iam

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/cache"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ===========================================================================
// gRPC handler: the service-principal actor path for role WIRING
// ===========================================================================
//
// Every maintainerd service ships its own roles/permissions for its routes,
// and the orchestrator — a SERVICE principal with a client_credentials token
// that can never carry a user — provisions them as behind-the-scenes wiring.
// These tests pin the shape of that exception: the four wiring RPCs admit a
// service principal ONLY in the token's own tenant, and role DESTRUCTION
// (SetRoleStatus, DeleteRole) keeps requiring a human.
//
// grpcCallerCtx already models the bare service-principal token the
// interceptor installs (Service claim, tenant, no user in the auth context).

func TestRoleGRPCHandler_ServiceActor(t *testing.T) {
	tenantUUID := uuid.New()
	roleUUID := uuid.New()
	permissionUUID := uuid.New()
	role := RoleServiceDataResult{RoleUUID: roleUUID, Name: "core-agent-role", Status: shared.StatusActive}
	tenantResolver := accessTenantResolver(t, tenantUUID) // resolves to tenant 77
	ownTenantCtx := grpcCallerCtx(77)                     // service "svc-test", token tenant 77

	t.Run("service principal creates a role in its own tenant with no user attribution", func(t *testing.T) {
		var gotActor RoleActor
		svc := &mockRoleService{
			createFn: func(_, _ string, _, _ bool, _, _ string, actor RoleActor) (*RoleServiceDataResult, error) {
				gotActor = actor
				return &role, nil
			},
		}
		h := NewRoleGRPCHandler(tenantResolver, svc)
		res, err := h.CreateRole(ownTenantCtx, validCreateRoleRequest(tenantUUID))
		require.NoError(t, err)
		assert.Nil(t, gotActor.UserUUID, "no user may be fabricated for a machine actor")
		assert.Equal(t, "svc-test", gotActor.ServiceName)
		assert.Equal(t, roleUUID.String(), res.Role.RoleId)
	})

	t.Run("service principal updates a role in its own tenant", func(t *testing.T) {
		var gotActor RoleActor
		svc := &mockRoleService{
			updateFn: func(_ uuid.UUID, _ int64, _, _ string, _, _ bool, _ string, actor RoleActor) (*RoleServiceDataResult, error) {
				gotActor = actor
				return &role, nil
			},
		}
		h := NewRoleGRPCHandler(tenantResolver, svc)
		_, err := h.UpdateRole(ownTenantCtx, validUpdateRoleRequest(tenantUUID, roleUUID))
		require.NoError(t, err)
		assert.Equal(t, "svc-test", gotActor.ServiceName)
	})

	t.Run("service principal wires permissions on and off a role in its own tenant", func(t *testing.T) {
		var addActor, removeActor RoleActor
		svc := &mockRoleService{
			addRolePermsFn: func(_ uuid.UUID, _ int64, _ []uuid.UUID, actor RoleActor) (*RoleServiceDataResult, error) {
				addActor = actor
				return &role, nil
			},
			removeRolePermsFn: func(_ uuid.UUID, _ int64, _ uuid.UUID, actor RoleActor) (*RoleServiceDataResult, error) {
				removeActor = actor
				return &role, nil
			},
		}
		h := NewRoleGRPCHandler(tenantResolver, svc)

		_, err := h.AddRolePermissions(ownTenantCtx, &authv1.AddRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionIds: []string{permissionUUID.String()}})
		require.NoError(t, err)
		assert.Equal(t, "svc-test", addActor.ServiceName)
		assert.Nil(t, addActor.UserUUID)

		// RemoveRolePermission is rewiring, not destruction: role and permission
		// both survive, so the orchestrator may do it too.
		_, err = h.RemoveRolePermission(ownTenantCtx, &authv1.RemoveRolePermissionRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionId: permissionUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, "svc-test", removeActor.ServiceName)
	})

	// Tenant safety: even a SYSTEM-tenant machine token — which resolveIAMTenant
	// would wave through for a user actor — may not wire roles in another tenant
	// on the bare service-actor path. The service is nil so a call that slipped
	// past the boundary would panic rather than pass.
	t.Run("service principal from another tenant is refused", func(t *testing.T) {
		h := NewRoleGRPCHandler(mockTenantResolver{}, nil) // target resolves to tenant 77
		systemTenantCtx := grpcCallerCtx(systemTenantIDForTests)

		for name, call := range map[string]func() error{
			"CreateRole": func() error {
				_, err := h.CreateRole(systemTenantCtx, validCreateRoleRequest(tenantUUID))
				return err
			},
			"UpdateRole": func() error {
				_, err := h.UpdateRole(systemTenantCtx, validUpdateRoleRequest(tenantUUID, roleUUID))
				return err
			},
			"AddRolePermissions": func() error {
				_, err := h.AddRolePermissions(systemTenantCtx, &authv1.AddRolePermissionsRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionIds: []string{permissionUUID.String()}})
				return err
			},
			"RemoveRolePermission": func() error {
				_, err := h.RemoveRolePermission(systemTenantCtx, &authv1.RemoveRolePermissionRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), PermissionId: permissionUUID.String()})
				return err
			},
		} {
			t.Run(name, func(t *testing.T) {
				err := call()
				assert.Equal(t, codes.PermissionDenied, status.Code(err))
				assert.Contains(t, status.Convert(err).Message(), "own tenant")
			})
		}
	})

	// Destruction stays human-only, refused with exactly the error the
	// user-actor path has always produced.
	t.Run("service principal may not destroy: SetRoleStatus and DeleteRole keep the human-actor error", func(t *testing.T) {
		h := NewRoleGRPCHandler(tenantResolver, nil)

		_, err := h.SetRoleStatus(ownTenantCtx, &authv1.SetRoleStatusRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String(), Status: shared.StatusInactive})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "requires a user principal",
			"the refusal must be the same one the user-actor path always produced")

		_, err = h.DeleteRole(ownTenantCtx, &authv1.DeleteRoleRequest{TenantId: tenantUUID.String(), RoleId: roleUUID.String()})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "requires a user principal")
	})
}

// ===========================================================================
// Service layer: the service-actor path skips user checks and attributes NULL
// ===========================================================================

func TestRoleService_Create_ServiceActor(t *testing.T) {
	tenantID := int64(1)
	tenantUUID := uuid.New()

	// The user repo would fabricate an actor if consulted; returning nil proves
	// the service-actor path never looks a user up.
	userRepo := &mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil }}
	tenantRepo := &mockTenantRepo{findByUUIDFn: func(_ any, _ ...string) (*Tenant, error) {
		return &Tenant{TenantID: tenantID, TenantUUID: tenantUUID}, nil
	}}

	t.Run("creates the role with NULL user attribution and the service named in the trail", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		authEvents := &recordingAuthEventService{}
		events := &recordingEventService{}
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, userRepo, tenantRepo, cache.NopInvalidator{}, authEvents, events)

		result, err := svc.Create(context.Background(), "core-agent-role", "desc", false, false, shared.StatusActive, tenantUUID.String(), ServiceActor("core"))
		require.NoError(t, err)
		assert.Equal(t, "core-agent-role", result.Name)

		require.Len(t, events.emitted, 1)
		assert.Nil(t, events.emitted[0].ActorUserID, "no user may be fabricated for a machine actor")
		require.Len(t, authEvents.events, 1)
		assert.Nil(t, authEvents.events[0].ActorUserID)
		// With actor_user_id NULL the description is the only place the record
		// answers "who did this".
		assert.Contains(t, *authEvents.events[0].Description, "(by service core)")
	})

	// Fail closed: a zero actor is neither a user nor a service.
	t.Run("refuses an empty actor entirely", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewRoleService(db, &mockRoleRepo{}, &mockPermissionRepo{}, &mockRolePermissionRepo{}, userRepo, tenantRepo, cache.NopInvalidator{}, nil, nil)

		_, err := svc.Create(context.Background(), "core-agent-role", "desc", false, false, shared.StatusActive, tenantUUID.String(), RoleActor{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires an acting principal")
	})
}

func TestRoleService_Update_ServiceActor(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()

	db, mock := newMockGormDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()
	events := &recordingEventService{}
	svc := NewRoleService(db, &mockRoleRepo{
		findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return newRole(1, "core-agent-role", tenantID), nil },
	}, &mockPermissionRepo{}, &mockRolePermissionRepo{},
		&mockUserRepo{findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil }},
		&mockTenantRepo{}, cache.NopInvalidator{}, nil, events)

	result, err := svc.Update(context.Background(), roleUUID, tenantID, "renamed", "desc", false, false, shared.StatusActive, ServiceActor("core"))
	require.NoError(t, err)
	assert.Equal(t, "renamed", result.Name)
	require.Len(t, events.emitted, 1)
	assert.Nil(t, events.emitted[0].ActorUserID)
}

func TestRoleService_RolePermissions_ServiceActor(t *testing.T) {
	tenantID := int64(1)
	roleUUID := uuid.New()
	permissionUUID := uuid.New()

	// An elevated permission plus a user repo that refuses to answer: if the
	// service-actor path ever consulted the privilege-escalation guard (a
	// user-actor containment rule), this test would fail rather than silently
	// treat the machine as holding everything.
	elevated := Permission{PermissionID: 1, PermissionUUID: permissionUUID, Name: "tenant:delete", TenantID: tenantID}
	userRepo := &mockUserRepo{
		findByUUIDFn: func(_ any, _ ...string) (*User, error) { return nil, nil },
		effectivePermissionNamesFn: func(int64, int64) ([]string, error) {
			return nil, errors.New("must not be consulted for a service actor")
		},
	}

	t.Run("attaches permissions with NULL user attribution", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		events := &recordingEventService{}
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return newRole(1, "core-agent-role", tenantID), nil },
		}, &mockPermissionRepo{
			findByUUIDsFn: func([]string, ...string) ([]Permission, error) { return []Permission{elevated}, nil },
		}, &mockRolePermissionRepo{}, userRepo, &mockTenantRepo{}, cache.NopInvalidator{}, nil, events)

		result, err := svc.AddRolePermissions(context.Background(), roleUUID, tenantID, []uuid.UUID{permissionUUID}, ServiceActor("core"))
		require.NoError(t, err)
		assert.NotNil(t, result)
		require.Len(t, events.emitted, 1)
		assert.Nil(t, events.emitted[0].ActorUserID)
	})

	t.Run("detaches a permission with NULL user attribution", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		events := &recordingEventService{}
		svc := NewRoleService(db, &mockRoleRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Role, error) { return newRole(1, "core-agent-role", tenantID), nil },
		}, &mockPermissionRepo{
			findByUUIDFn: func(_ any, _ ...string) (*Permission, error) { return &elevated, nil },
		}, &mockRolePermissionRepo{
			findByRoleAndPermissionFn: func(int64, int64) (*RolePermission, error) {
				return &RolePermission{RoleID: 1, PermissionID: 1}, nil
			},
		}, userRepo, &mockTenantRepo{}, cache.NopInvalidator{}, nil, events)

		result, err := svc.RemoveRolePermissions(context.Background(), roleUUID, tenantID, permissionUUID, ServiceActor("core"))
		require.NoError(t, err)
		assert.NotNil(t, result)
		require.Len(t, events.emitted, 1)
		assert.Nil(t, events.emitted[0].ActorUserID)
	})
}
