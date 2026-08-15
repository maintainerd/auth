package iam

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

type mockTenantResolver struct {
	getByUUIDFn func(uuid.UUID) (*tenant.TenantServiceDataResult, error)
	getSystemFn func() (*tenant.TenantServiceDataResult, error)
}

func (m mockTenantResolver) GetByUUID(_ context.Context, tenantUUID uuid.UUID) (*tenant.TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(tenantUUID)
	}
	// Matches grpcCallerCtx's default tenant so the bare resolver represents a caller
	// acting on its OWN tenant; cross-tenant is exercised explicitly below.
	return &tenant.TenantServiceDataResult{TenantID: 77, TenantUUID: tenantUUID}, nil
}

// GetSystem identifies the control-plane tenant. resolveIAMTenant uses it to decide
// whether a cross-tenant request is allowed; without it the check fails closed.
func (m mockTenantResolver) GetSystem(_ context.Context) (*tenant.TenantServiceDataResult, error) {
	if m.getSystemFn != nil {
		return m.getSystemFn()
	}
	return &tenant.TenantServiceDataResult{TenantID: systemTenantIDForTests, IsSystem: true}, nil
}

// systemTenantIDForTests is a tenant id no fixture uses as a target, so the default
// resolver reports "the caller is NOT the control plane".
const systemTenantIDForTests = int64(90001)

// grpcCallerCtx returns a context whose token is bound to tenantID, which is what
// resolveIAMTenant compares the requested tenant against.
func grpcCallerCtx(tenantID int64) context.Context {
	return middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		Service:     "svc-test",
		SubjectType: "service",
		TenantID:    tenantID,
	})
}

// grpcUserCallerCtx is grpcCallerCtx plus a resolved user principal, which the
// mutating role RPCs now require: the actor is read from the verified token, never
// from the request body.
func grpcUserCallerCtx(tenantID int64, actorUUID uuid.UUID) context.Context {
	return middleware.WithAuthContextValue(grpcCallerCtx(tenantID), &authctx.AuthContext{
		User: &authctx.AuthUser{UserID: 1, UserUUID: actorUUID},
	})
}

// The actor used to be req.GetActorUserId(): a request-body field that drove both
// audit attribution AND the ValidateTenantAccess subject, so a caller could pin a
// change on an innocent user and borrow that user's tenant membership.
func TestIAMActorUserUUID(t *testing.T) {
	actorUUID := uuid.New()

	t.Run("prefers the resolved user principal", func(t *testing.T) {
		got, err := iamActorUserUUID(grpcUserCallerCtx(77, actorUUID))
		require.NoError(t, err)
		assert.Equal(t, actorUUID, got)
	})

	// INVERTED. This used to pass with raw JWT claims and no AuthContext, because
	// the helper kept a private fallback to the claims. That fallback made this
	// surface accept a token the tenant surface refused, so it was removed and
	// all three surfaces now share one definition sourced from the
	// interceptor-populated AuthContext.
	t.Run("claims alone are not enough — the actor comes from the auth context", func(t *testing.T) {
		ctx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
			Sub: actorUUID.String(), UserUUID: actorUUID, SubjectType: "user", TenantID: 77,
		})
		_, err := iamActorUserUUID(ctx)
		require.Error(t, err, "a context the interceptor never populated must not authenticate an actor")
	})

	t.Run("accepts a user resolved by the interceptor", func(t *testing.T) {
		ctx := middleware.WithAuthContextValue(context.Background(), &authctx.AuthContext{
			User: &authctx.AuthUser{UserUUID: actorUUID},
		})
		got, err := iamActorUserUUID(ctx)
		require.NoError(t, err)
		assert.Equal(t, actorUUID, got)
	})

	// Fail closed. The interceptor admits service principals, which carry no user —
	// falling back to the body is what created the hole.
	for _, tc := range []struct {
		name string
		ctx  context.Context
	}{
		{name: "service principal", ctx: grpcCallerCtx(77)},
		{name: "no claims at all", ctx: context.Background()},
		{
			name: "service token whose sub happens to parse as a UUID",
			ctx: middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
				Sub: actorUUID.String(), UserUUID: actorUUID, Service: "svc-test", SubjectType: "service", TenantID: 77,
			}),
		},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			got, err := iamActorUserUUID(tc.ctx)
			assert.Equal(t, uuid.Nil, got)
			assert.Equal(t, codes.PermissionDenied, status.Code(err))
		})
	}
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

func accessTenantResolver(t *testing.T, tenantUUID uuid.UUID) mockTenantResolver {
	t.Helper()
	return mockTenantResolver{getByUUIDFn: func(id uuid.UUID) (*tenant.TenantServiceDataResult, error) {
		assert.Equal(t, tenantUUID, id)
		return &tenant.TenantServiceDataResult{TenantID: 77, TenantUUID: id}, nil
	}}
}

func TestAccessGRPCHelpers(t *testing.T) {
	assert.Nil(t, permissionProto(nil))
	assert.Nil(t, policyProto(nil))
	assert.Nil(t, policyServiceProto(nil))
	assert.Nil(t, roleProto(nil))
	assert.Empty(t, policyDocumentProto(nil).AsMap())
	assert.Empty(t, policyDocumentProto(datatypes.JSON(`bad`)).AsMap())
	_, err := policyDocumentJSON(nil)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	ids, err := parseIAMUUIDs([]string{testResourceUUID.String()}, "Permission UUID")
	require.NoError(t, err)
	assert.Equal(t, testResourceUUID, ids[0])
}

// The gRPC surface takes the target tenant from the REQUEST BODY, and the interceptor
// authorizes an action only — it never compares that tenant against the token. So
// existence was the only check: a principal holding e.g. service:update in its own
// tenant could pass another tenant's UUID and mutate that tenant's resources.
func TestResolveIAMTenant_EnforcesTheTenantBoundary(t *testing.T) {
	targetUUID := uuid.New()
	const targetTenantID = int64(77)
	const systemTenantID = int64(1)

	resolver := mockTenantResolver{
		getByUUIDFn: func(id uuid.UUID) (*tenant.TenantServiceDataResult, error) {
			return &tenant.TenantServiceDataResult{TenantID: targetTenantID, TenantUUID: id}, nil
		},
		getSystemFn: func() (*tenant.TenantServiceDataResult, error) {
			return &tenant.TenantServiceDataResult{TenantID: systemTenantID, IsSystem: true}, nil
		},
	}

	t.Run("a caller may act on its own tenant", func(t *testing.T) {
		scope, err := resolveIAMTenant(grpcCallerCtx(targetTenantID), resolver, targetUUID.String())
		require.NoError(t, err)
		assert.Equal(t, targetTenantID, scope.TenantID)
	})

	t.Run("a caller may NOT act on another tenant", func(t *testing.T) {
		_, err := resolveIAMTenant(grpcCallerCtx(4242), resolver, targetUUID.String())
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	// The control plane lives in the system tenant and must be able to configure any
	// tenant remotely — that is the whole point of the gRPC surface.
	t.Run("the system tenant may act on any tenant", func(t *testing.T) {
		scope, err := resolveIAMTenant(grpcCallerCtx(systemTenantID), resolver, targetUUID.String())
		require.NoError(t, err)
		assert.Equal(t, targetTenantID, scope.TenantID)
	})

	t.Run("a token with no tenant is refused", func(t *testing.T) {
		_, err := resolveIAMTenant(context.Background(), resolver, targetUUID.String())
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = resolveIAMTenant(grpcCallerCtx(0), resolver, targetUUID.String())
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	// Fail closed: if the system tenant cannot be identified, cross-tenant access
	// cannot be justified.
	t.Run("cross-tenant is refused when the system tenant cannot be resolved", func(t *testing.T) {
		broken := mockTenantResolver{
			getByUUIDFn: func(id uuid.UUID) (*tenant.TenantServiceDataResult, error) {
				return &tenant.TenantServiceDataResult{TenantID: targetTenantID, TenantUUID: id}, nil
			},
			getSystemFn: func() (*tenant.TenantServiceDataResult, error) {
				return nil, assert.AnError
			},
		}
		_, err := resolveIAMTenant(grpcCallerCtx(4242), broken, targetUUID.String())
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})
}

// Guards the failure mode this fix actually had: the boundary lived in ONE resolver
// while two handlers resolved tenants themselves and were unaffected. Any new tenant
// entry point on the IAM gRPC surface must apply the check.
func TestEveryIAMGRPCTenantEntryPointChecksTheBoundary(t *testing.T) {
	for _, path := range []string{
		"grpc_helpers.go",
		"handler_service_grpc.go",
		"handler_api_grpc.go",
	} {
		body, err := os.ReadFile(path)
		require.NoError(t, err, path)
		source := string(body)

		if !strings.Contains(source, "tenantService.GetByUUID(ctx, parsed)") {
			continue // no longer a tenant entry point
		}
		assert.Contains(t, source, "assertCallerMayActOnTenant",
			"%s resolves a tenant from the request but never checks the caller may act on it", path)
	}
}
