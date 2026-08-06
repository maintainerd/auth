package tenant

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

// grpcAuthCtx builds the context a request carries after the gRPC auth
// interceptor runs for a USER principal: the issuer-stamped tenant claim plus
// the AuthContext the handlers now read the acting user from.
func grpcAuthCtx(tenantID int64, actorUserID int64) context.Context {
	// Sub and UserUUID are the SAME value on a real token — UserUUID is parsed out
	// of sub — so setting only one builds a principal the issuer cannot mint.
	subjectUUID := uuid.New()
	ctx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		TenantID: tenantID,
		UserUUID: subjectUUID,
		Sub:      subjectUUID.String(),
	})
	return middleware.WithAuthContextValue(ctx, &authctx.AuthContext{
		User: &authctx.AuthUser{UserID: actorUserID, UserUUID: uuid.New()},
	})
}

// grpcServiceCtx builds the context for a SERVICE principal: a tenant claim
// with no user identity behind it, which is what the gRPC interceptor produces
// for the control plane.
func grpcServiceCtx(tenantID int64) context.Context {
	return middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{TenantID: tenantID})
}

func TestTenantGRPCHandler_TenantRPCs(t *testing.T) {
	tenantUUID := uuid.New()
	now := time.Now().UTC()
	sample := &TenantServiceDataResult{
		TenantID:    44,
		TenantUUID:  tenantUUID,
		Name:        "tenant-one",
		DisplayName: "Tenant One",
		Description: "Tenant description",
		Status:      shared.StatusActive,
		Metadata:    datatypes.JSON(`{"plan":"pro"}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Tenant create/update/status now require an authenticated actor. A
	// system-tenant principal is authorized for all of them (parity with HTTP);
	// build a matching claims context and a GetSystem stub that agrees on the ID.
	const sysTID = int64(1)
	// sysSubject: sub and UserUUID are one value on a real token.
	sysSubject := uuid.New()
	sysCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{TenantID: sysTID, UserUUID: sysSubject, Sub: sysSubject.String()})
	getSys := func() (*TenantServiceDataResult, error) {
		return &TenantServiceDataResult{TenantID: sysTID, IsSystem: true}, nil
	}

	t.Run("get default success and error", func(t *testing.T) {
		h := NewTenantGRPCHandler(&mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return sample, nil
		}}, nil)
		resp, err := h.GetDefaultTenant(context.Background(), &authv1.GetDefaultTenantRequest{})
		require.NoError(t, err)
		assert.Equal(t, tenantUUID.String(), resp.Tenant.TenantUuid)
		assert.Equal(t, "pro", resp.Tenant.Metadata.AsMap()["plan"])

		_, err = NewTenantGRPCHandler(&mockTenantService{getSystemFn: func() (*TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing")
		}}, nil).GetDefaultTenant(context.Background(), &authv1.GetDefaultTenantRequest{})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("list validates filters and maps rows", func(t *testing.T) {
		h := NewTenantGRPCHandler(&mockTenantService{getSystemFn: getSys, getFn: func(filter TenantServiceGetFilter) (*TenantServiceGetResult, error) {
			require.NotNil(t, filter.Name)
			assert.Equal(t, "tenant", *filter.Name)
			assert.Equal(t, 2, filter.Page)
			assert.Equal(t, 5, filter.Limit)
			// A system-tenant principal stays unscoped so it can still enumerate
			// every tenant.
			assert.Empty(t, filter.TenantIDs)
			return &TenantServiceGetResult{Data: []TenantServiceDataResult{*sample}, Total: 1, Page: 2, Limit: 5, TotalPages: 3}, nil
		}}, nil)
		resp, err := h.ListTenants(sysCtx, &authv1.ListTenantsRequest{
			Name:       "tenant",
			Status:     []string{shared.StatusActive},
			Pagination: &authv1.Pagination{Page: 2, Limit: 5, SortBy: "name", SortOrder: SortOrderAsc},
		})
		require.NoError(t, err)
		assert.Len(t, resp.Tenants, 1)
		assert.Equal(t, int64(1), resp.Page.Total)

		_, err = h.ListTenants(sysCtx, &authv1.ListTenantsRequest{Pagination: &authv1.Pagination{Page: -1, Limit: 5}})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		_, err = NewTenantGRPCHandler(&mockTenantService{getSystemFn: getSys, getFn: func(TenantServiceGetFilter) (*TenantServiceGetResult, error) {
			return nil, errors.New("db")
		}}, nil).ListTenants(sysCtx, &authv1.ListTenantsRequest{})
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	// Regression: ListTenants did no scoping at all, so any tenant-bound token
	// read every tenant record over gRPC while the HTTP handler scoped.
	t.Run("list is scoped to a non-system caller's own tenant", func(t *testing.T) {
		var gotFilter TenantServiceGetFilter
		h := NewTenantGRPCHandler(&mockTenantService{getSystemFn: getSys, getFn: func(f TenantServiceGetFilter) (*TenantServiceGetResult, error) {
			gotFilter = f
			return &TenantServiceGetResult{}, nil
		}}, nil)

		_, err := h.ListTenants(grpcAuthCtx(777, 5), &authv1.ListTenantsRequest{})
		require.NoError(t, err)
		assert.Equal(t, []int64{777}, gotFilter.TenantIDs)

		_, err = h.ListTenants(context.Background(), &authv1.ListTenantsRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("get tenant parses uuid and maps errors", func(t *testing.T) {
		h := NewTenantGRPCHandler(&mockTenantService{getSystemFn: getSys, getByUUIDFn: func(id uuid.UUID) (*TenantServiceDataResult, error) {
			assert.Equal(t, tenantUUID, id)
			return sample, nil
		}}, nil)
		resp, err := h.GetTenant(sysCtx, &authv1.GetTenantRequest{TenantUuid: tenantUUID.String()})
		require.NoError(t, err)
		assert.Equal(t, tenantUUID.String(), resp.Tenant.TenantUuid)

		_, err = h.GetTenant(sysCtx, &authv1.GetTenantRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		_, err = NewTenantGRPCHandler(&mockTenantService{getSystemFn: getSys, getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing")
		}}, nil).GetTenant(sysCtx, &authv1.GetTenantRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	// Regression: GetTenant handed any tenant record to any authenticated
	// caller. sample belongs to tenant 44.
	t.Run("get tenant is scoped to the caller's own tenant", func(t *testing.T) {
		h := NewTenantGRPCHandler(&mockTenantService{getSystemFn: getSys, getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return sample, nil
		}}, nil)

		_, err := h.GetTenant(grpcAuthCtx(777, 5), &authv1.GetTenantRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = h.GetTenant(grpcAuthCtx(sample.TenantID, 5), &authv1.GetTenantRequest{TenantUuid: tenantUUID.String()})
		require.NoError(t, err)

		_, err = h.GetTenant(context.Background(), &authv1.GetTenantRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("create tenant validates request and maps service", func(t *testing.T) {
		h := NewTenantGRPCHandler(&mockTenantService{
			getSystemFn: getSys,
			createFn: func(name, displayName, description, status string) (*TenantServiceDataResult, error) {
				assert.Equal(t, "tenant-one", name)
				return sample, nil
			}}, nil)
		resp, err := h.CreateTenant(sysCtx, &authv1.TenantServiceCreateTenantRequest{
			Name: "tenant-one", DisplayName: "Tenant One", Description: "Long enough description", Status: shared.StatusActive,
		})
		require.NoError(t, err)
		assert.Equal(t, "tenant-one", resp.Tenant.Name)

		_, err = h.CreateTenant(sysCtx, &authv1.TenantServiceCreateTenantRequest{Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		_, err = NewTenantGRPCHandler(&mockTenantService{
			getSystemFn: getSys,
			createFn: func(string, string, string, string) (*TenantServiceDataResult, error) {
				return nil, apperror.NewConflict("exists")
			}}, nil).CreateTenant(sysCtx, &authv1.TenantServiceCreateTenantRequest{
			Name: "tenant-one", Description: "Long enough description", Status: shared.StatusActive,
		})
		assert.Equal(t, codes.AlreadyExists, status.Code(err))

		// Boundary: a NON-system principal is refused at the gRPC surface even
		// though it may hold tenant:create in its own tenant (the closed hole).
		regularCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{TenantID: 777})
		_, err = h.CreateTenant(regularCtx, &authv1.TenantServiceCreateTenantRequest{
			Name: "tenant-one", DisplayName: "Tenant One", Description: "Long enough description", Status: shared.StatusActive,
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("update status public and delete", func(t *testing.T) {
		h := NewTenantGRPCHandler(&mockTenantService{
			getSystemFn: getSys,
			updateFn: func(id uuid.UUID, name, displayName, description, status string) (*TenantServiceDataResult, error) {
				assert.Equal(t, tenantUUID, id)
				return sample, nil
			},
			setStatusByUUIDFn: func(id uuid.UUID, status string) (*TenantServiceDataResult, error) {
				assert.Equal(t, shared.StatusSuspended, status)
				return sample, nil
			},
			deleteByUUIDFn: func(id uuid.UUID) (*TenantServiceDataResult, error) {
				assert.Equal(t, tenantUUID, id)
				return sample, nil
			},
		}, &mockTenantMemberService{})

		updateResp, err := h.UpdateTenant(sysCtx, &authv1.TenantServiceUpdateTenantRequest{
			TenantUuid: tenantUUID.String(), Name: "tenant-one", DisplayName: "Tenant One", Description: "Long enough description", Status: shared.StatusActive,
		})
		require.NoError(t, err)
		assert.Equal(t, sample.Name, updateResp.Tenant.Name)

		statusResp, err := h.SetTenantStatus(sysCtx, &authv1.SetTenantStatusRequest{TenantUuid: tenantUUID.String(), Status: shared.StatusSuspended})
		require.NoError(t, err)
		assert.Equal(t, sample.Name, statusResp.Tenant.Name)

		// The actor now comes from the token; the body's actor_user_uuid is
		// ignored, so a spoofed one cannot stand in for a real principal.
		sysUserCtx := grpcAuthCtx(sysTID, 7)
		deleteResp, err := h.DeleteTenant(sysUserCtx, &authv1.DeleteTenantRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: uuid.New().String()})
		require.NoError(t, err)
		assert.Equal(t, sample.Name, deleteResp.Tenant.Name)

		// Boundary parity with HTTP: only a system-tenant principal may delete.
		_, err = h.DeleteTenant(grpcAuthCtx(777, 7), &authv1.DeleteTenantRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		// A service principal carries no user identity to attribute the delete to.
		_, err = h.DeleteTenant(grpcServiceCtx(sysTID), &authv1.DeleteTenantRequest{TenantUuid: tenantUUID.String(), ActorUserUuid: uuid.New().String()})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = h.UpdateTenant(sysCtx, &authv1.TenantServiceUpdateTenantRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateTenant(sysCtx, &authv1.TenantServiceUpdateTenantRequest{TenantUuid: tenantUUID.String(), Name: "x"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetTenantStatus(sysCtx, &authv1.SetTenantStatusRequest{TenantUuid: tenantUUID.String(), Status: "deleted"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.SetTenantStatus(sysCtx, &authv1.SetTenantStatusRequest{TenantUuid: "bad", Status: shared.StatusActive})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.DeleteTenant(sysUserCtx, &authv1.DeleteTenantRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		// Boundary: a principal bound to another tenant (not a member of the target,
		// not system) cannot manage this tenant.
		deniedH := NewTenantGRPCHandler(&mockTenantService{getSystemFn: getSys}, &mockTenantMemberService{
			canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil },
		})
		regularCtx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{TenantID: 777, UserUUID: uuid.New()})
		_, err = deniedH.UpdateTenant(regularCtx, &authv1.TenantServiceUpdateTenantRequest{TenantUuid: tenantUUID.String(), Name: "tenant-one", Description: "Long enough description", Status: shared.StatusActive})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		errorSvc := &mockTenantService{
			getSystemFn: getSys,
			updateFn: func(uuid.UUID, string, string, string, string) (*TenantServiceDataResult, error) {
				return nil, errors.New("db")
			},
			setStatusByUUIDFn: func(uuid.UUID, string) (*TenantServiceDataResult, error) { return nil, errors.New("db") },
			deleteByUUIDFn:    func(uuid.UUID) (*TenantServiceDataResult, error) { return nil, errors.New("db") },
		}
		errHandler := NewTenantGRPCHandler(errorSvc, &mockTenantMemberService{})
		_, err = errHandler.UpdateTenant(sysCtx, &authv1.TenantServiceUpdateTenantRequest{TenantUuid: tenantUUID.String(), Name: "tenant-one", Description: "Long enough description", Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = errHandler.SetTenantStatus(sysCtx, &authv1.SetTenantStatusRequest{TenantUuid: tenantUUID.String(), Status: shared.StatusActive})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = errHandler.DeleteTenant(sysUserCtx, &authv1.DeleteTenantRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestTenantGRPCHandler_MemberRPCs(t *testing.T) {
	tenantUUID := uuid.New()
	memberUUID := uuid.New()
	userUUID := uuid.New()
	tenantResult := &TenantServiceDataResult{TenantID: 12, TenantUUID: tenantUUID, Name: "tenant-one"}
	member := &TenantMemberServiceDataResult{
		TenantMemberUUID: memberUUID,
		TenantID:         12,
		Role:             "owner",
		User:             &MemberUser{UserUUID: userUUID, Username: "ada", Metadata: datatypes.JSON(`{"team":"core"}`)},
	}
	// The acting user now comes from the token, so every call below runs as this
	// user no matter what actor_user_uuid the request body carries.
	const tokenActorUserID = int64(7)
	ctx := grpcAuthCtx(memberRPCSystemTenantID, tokenActorUserID)
	members := &mockTenantMemberService{
		listByTenantFn: func(filter TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error) {
			assert.Equal(t, int64(12), filter.TenantID)
			require.NotNil(t, filter.Role)
			return &TenantMemberServiceListResult{Data: []TenantMemberServiceDataResult{*member}, Total: 1, Page: 1, Limit: 20, TotalPages: 1}, nil
		},
		createByUserUUIDFn: func(tenantID int64, id uuid.UUID, role string) (*TenantMemberServiceDataResult, error) {
			assert.Equal(t, int64(12), tenantID)
			assert.Equal(t, userUUID, id)
			assert.Equal(t, "owner", role)
			return member, nil
		},
		updateRoleFn: func(tenantID int64, id uuid.UUID, role string) (*TenantMemberServiceDataResult, error) {
			assert.Equal(t, int64(12), tenantID)
			assert.Equal(t, memberUUID, id)
			assert.Equal(t, "member", role)
			return member, nil
		},
		deleteByUUIDFn: func(tenantID int64, id uuid.UUID) error {
			assert.Equal(t, int64(12), tenantID)
			assert.Equal(t, memberUUID, id)
			return nil
		},
	}
	handler := NewTenantGRPCHandler(
		&mockTenantService{
			getSystemFn: memberRPCSystemTenant,
			getByUUIDFn: func(id uuid.UUID) (*TenantServiceDataResult, error) {
				assert.Equal(t, tenantUUID, id)
				return tenantResult, nil
			},
		},
		members,
	)

	listResp, err := handler.ListTenantMembers(ctx, &authv1.ListTenantMembersRequest{
		TenantUuid: tenantUUID.String(), Role: "owner",
	})
	require.NoError(t, err)
	require.Len(t, listResp.Members, 1)
	assert.Equal(t, "core", listResp.Members[0].User.Metadata.AsMap()["team"])

	// Every write still sends a body actor_user_uuid naming a different user;
	// the handler must ignore it in favour of the token's user.
	spoofedActor := uuid.New().String()

	addResp, err := handler.AddTenantMember(ctx, &authv1.AddTenantMemberRequest{
		TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Role: "owner", ActorUserUuid: spoofedActor,
	})
	require.NoError(t, err)
	assert.Equal(t, memberUUID.String(), addResp.Member.TenantMemberUuid)
	assert.Equal(t, tokenActorUserID, members.lastActorUserID)

	updateResp, err := handler.UpdateTenantMemberRole(ctx, &authv1.UpdateTenantMemberRoleRequest{
		TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), Role: "member", ActorUserUuid: spoofedActor,
	})
	require.NoError(t, err)
	assert.Equal(t, "owner", updateResp.Member.Role)
	assert.Equal(t, tokenActorUserID, members.lastActorUserID)

	removeResp, err := handler.RemoveTenantMember(ctx, &authv1.RemoveTenantMemberRequest{
		TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), ActorUserUuid: spoofedActor,
	})
	require.NoError(t, err)
	assert.True(t, removeResp.Removed)
	assert.Equal(t, tokenActorUserID, members.lastActorUserID)
}

// memberRPCSystemTenantID is the tenant the member-RPC tests authenticate
// against. It is the system tenant so the management gate passes and the tests
// can focus on the actor source and the boundary.
const memberRPCSystemTenantID = int64(1)

func memberRPCSystemTenant() (*TenantServiceDataResult, error) {
	return &TenantServiceDataResult{TenantID: memberRPCSystemTenantID, IsSystem: true}, nil
}

// Regression: the member RPCs took the acting principal from
// req.actor_user_uuid and performed no tenant-boundary check, so a tenant-B
// token could name a tenant-A member and have authorizeManager grant it
// management of tenant A — a full cross-tenant takeover.
func TestTenantGRPCHandler_MemberRPCsEnforceBoundaryAndTokenActor(t *testing.T) {
	tenantUUID := uuid.New()
	memberUUID := uuid.New()
	userUUID := uuid.New()
	tenantSvc := func() *mockTenantService {
		return &mockTenantService{
			getSystemFn: memberRPCSystemTenant,
			getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
				return &TenantServiceDataResult{TenantID: 12, TenantUUID: tenantUUID}, nil
			},
		}
	}

	t.Run("a principal of another tenant is denied every member RPC", func(t *testing.T) {
		h := NewTenantGRPCHandler(tenantSvc(), &mockTenantMemberService{
			canManageTenantFn: func(int64, uuid.UUID) (bool, error) { return false, nil },
		})
		ctx := grpcAuthCtx(777, 9)

		_, err := h.ListTenantMembers(ctx, &authv1.ListTenantMembersRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = h.AddTenantMember(ctx, &authv1.AddTenantMemberRequest{
			TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Role: "owner", ActorUserUuid: userUUID.String(),
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = h.UpdateTenantMemberRole(ctx, &authv1.UpdateTenantMemberRoleRequest{
			TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), Role: "owner", ActorUserUuid: userUUID.String(),
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = h.RemoveTenantMember(ctx, &authv1.RemoveTenantMemberRequest{
			TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), ActorUserUuid: userUUID.String(),
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("a service principal has no user identity and is refused", func(t *testing.T) {
		// The interceptor authenticates service tokens: they carry a tenant claim
		// but no user, so member writes fail closed rather than running as the
		// body-supplied actor. Reads inside the boundary stay allowed.
		h := NewTenantGRPCHandler(tenantSvc(), &mockTenantMemberService{})
		ctx := grpcServiceCtx(memberRPCSystemTenantID)

		_, err := h.AddTenantMember(ctx, &authv1.AddTenantMemberRequest{
			TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Role: "owner", ActorUserUuid: userUUID.String(),
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = h.UpdateTenantMemberRole(ctx, &authv1.UpdateTenantMemberRoleRequest{
			TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), Role: "owner", ActorUserUuid: userUUID.String(),
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = h.RemoveTenantMember(ctx, &authv1.RemoveTenantMemberRequest{
			TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), ActorUserUuid: userUUID.String(),
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("an unauthenticated caller is refused", func(t *testing.T) {
		h := NewTenantGRPCHandler(tenantSvc(), &mockTenantMemberService{})
		_, err := h.ListTenantMembers(context.Background(), &authv1.ListTenantMembersRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestTenantGRPCHandler_MemberRPCErrors(t *testing.T) {
	tenantUUID := uuid.New()
	memberUUID := uuid.New()
	userUUID := uuid.New()
	// A system-tenant user principal: it clears the management gate, so these
	// cases still exercise parsing, validation, and service-error mapping.
	ctx := grpcAuthCtx(memberRPCSystemTenantID, 7)

	t.Run("resolve tenant and list errors", func(t *testing.T) {
		_, err := NewTenantGRPCHandler(&mockTenantService{getSystemFn: memberRPCSystemTenant}, nil).ListTenantMembers(ctx, &authv1.ListTenantMembersRequest{TenantUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		_, err = NewTenantGRPCHandler(&mockTenantService{getSystemFn: memberRPCSystemTenant, getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, apperror.NewNotFound("missing")
		}}, nil).ListTenantMembers(ctx, &authv1.ListTenantMembersRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))

		_, err = NewTenantGRPCHandler(&mockTenantService{getSystemFn: memberRPCSystemTenant, getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}, &mockTenantMemberService{}).ListTenantMembers(ctx, &authv1.ListTenantMembersRequest{TenantUuid: tenantUUID.String(), Role: "superuser"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		_, err = NewTenantGRPCHandler(&mockTenantService{getSystemFn: memberRPCSystemTenant, getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}, &mockTenantMemberService{listByTenantFn: func(TenantMemberServiceListFilter) (*TenantMemberServiceListResult, error) {
			return nil, errors.New("db")
		}}).ListTenantMembers(ctx, &authv1.ListTenantMembersRequest{TenantUuid: tenantUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("add update remove validation and service errors", func(t *testing.T) {
		baseTenant := &mockTenantService{getSystemFn: memberRPCSystemTenant, getByUUIDFn: func(uuid.UUID) (*TenantServiceDataResult, error) {
			return &TenantServiceDataResult{TenantID: 1}, nil
		}}
		h := NewTenantGRPCHandler(baseTenant, &mockTenantMemberService{})

		_, err := h.AddTenantMember(ctx, &authv1.AddTenantMemberRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad", Role: "owner"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddTenantMember(ctx, &authv1.AddTenantMemberRequest{TenantUuid: "bad", UserUuid: userUUID.String(), Role: "owner"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.AddTenantMember(ctx, &authv1.AddTenantMemberRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Role: "superuser"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateTenantMemberRole(ctx, &authv1.UpdateTenantMemberRoleRequest{TenantUuid: tenantUUID.String(), TenantMemberUuid: "bad", Role: "owner"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateTenantMemberRole(ctx, &authv1.UpdateTenantMemberRoleRequest{TenantUuid: "bad", TenantMemberUuid: memberUUID.String(), Role: "owner"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.UpdateTenantMemberRole(ctx, &authv1.UpdateTenantMemberRoleRequest{TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), Role: "superuser"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveTenantMember(ctx, &authv1.RemoveTenantMemberRequest{TenantUuid: tenantUUID.String(), TenantMemberUuid: "bad"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		_, err = h.RemoveTenantMember(ctx, &authv1.RemoveTenantMemberRequest{TenantUuid: "bad", TenantMemberUuid: memberUUID.String()})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		errMembers := &mockTenantMemberService{
			createByUserUUIDFn: func(int64, uuid.UUID, string) (*TenantMemberServiceDataResult, error) { return nil, errors.New("db") },
			updateRoleFn:       func(int64, uuid.UUID, string) (*TenantMemberServiceDataResult, error) { return nil, errors.New("db") },
			deleteByUUIDFn:     func(int64, uuid.UUID) error { return errors.New("db") },
		}
		errHandler := NewTenantGRPCHandler(baseTenant, errMembers)
		_, err = errHandler.AddTenantMember(ctx, &authv1.AddTenantMemberRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Role: "owner", ActorUserUuid: userUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = errHandler.UpdateTenantMemberRole(ctx, &authv1.UpdateTenantMemberRoleRequest{TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), Role: "owner", ActorUserUuid: userUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
		_, err = errHandler.RemoveTenantMember(ctx, &authv1.RemoveTenantMemberRequest{TenantUuid: tenantUUID.String(), TenantMemberUuid: memberUUID.String(), ActorUserUuid: userUUID.String()})
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestTenantGRPCHandlerHelpers(t *testing.T) {
	id := uuid.New()
	parsed, err := parseGRPCUUID(id.String(), "Thing UUID")
	require.NoError(t, err)
	assert.Equal(t, id, parsed)
	_, err = parseGRPCUUID("", "Thing UUID")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = parseGRPCUUID("bad", "Thing UUID")
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	assert.Equal(t, 1, paginationDTO(nil).Page)
	assert.Equal(t, paginationDTO(&authv1.Pagination{}).Limit, paginationDTO(nil).Limit)
	assert.Nil(t, tenantProto(nil))
	assert.Nil(t, tenantMemberProto(nil))
	assert.Nil(t, tenantMemberUserProto(nil))
	assert.Nil(t, jsonStruct(nil))
	assert.Nil(t, jsonStruct([]byte(`{bad}`)))
	require.NotNil(t, jsonStruct([]byte(`{"ok":true}`)))

	value := true
	assert.Same(t, &value, optionalBool(&value))
	assert.Nil(t, optionalBool(nil))
	assert.Nil(t, optionalString(""))
	require.NotNil(t, optionalString("x"))
}
