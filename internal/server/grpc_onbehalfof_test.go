package server

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// fakeActorRepo is the narrow grpcOnBehalfOfResolver slice of the user
// repository, so the on-behalf-of rules can be exercised without a database.
type fakeActorRepo struct {
	actor    *user.User
	err      error
	lookedUp any
	calls    int
}

func (f *fakeActorRepo) FindByUUID(id any, _ ...string) (*user.User, error) {
	f.calls++
	f.lookedUp = id
	return f.actor, f.err
}

// actorUser builds a user whose only identity lives in identityTenantID — the
// association ValidateTenantAccess reads to decide tenant membership.
func actorUser(userID int64, userUUID uuid.UUID, identityTenantID int64, identityTenantUUID uuid.UUID, status string) *user.User {
	return &user.User{
		UserID:   userID,
		UserUUID: userUUID,
		TenantID: identityTenantID,
		Email:    "actor@example.com",
		Status:   status,
		UserIdentities: []user.UserIdentity{{
			TenantID: identityTenantID,
			Sub:      userUUID.String(),
			Tenant: &user.Tenant{
				TenantID:    identityTenantID,
				TenantUUID:  identityTenantUUID,
				Name:        "acme",
				DisplayName: "Acme",
			},
		}},
	}
}

func TestGRPCAuthContext_OnBehalfOf(t *testing.T) {
	const callerTenantID = int64(7)
	callerTenantUUID := uuid.New()

	caller := func(onBehalfOf string, tenantID int64) *grpcCaller {
		return &grpcCaller{
			claims: &middleware.JWTClaims{
				Sub:         "svc-core",
				Service:     "core",
				SubjectType: "service",
				TenantID:    tenantID,
				TenantUUID:  callerTenantUUID,
			},
			onBehalfOf: onBehalfOf,
		}
	}

	t.Run("no on_behalf_of still populates the tenant", func(t *testing.T) {
		auth, err := grpcAuthContext(&fakeActorRepo{}, caller("", callerTenantID))
		require.NoError(t, err)
		require.NotNil(t, auth.Tenant)
		assert.Equal(t, callerTenantID, auth.Tenant.TenantID)
		assert.Equal(t, callerTenantUUID, auth.Tenant.TenantUUID)
		// Fails closed: no signed actor, no user on the context.
		assert.Nil(t, auth.User)
	})

	t.Run("a signed actor in the caller's tenant is resolved", func(t *testing.T) {
		actorUUID := uuid.New()
		repo := &fakeActorRepo{actor: actorUser(42, actorUUID, callerTenantID, callerTenantUUID, shared.StatusActive)}

		auth, err := grpcAuthContext(repo, caller(actorUUID.String(), callerTenantID))
		require.NoError(t, err)
		require.NotNil(t, auth.User)
		assert.Equal(t, int64(42), auth.User.UserID)
		assert.Equal(t, actorUUID, auth.User.UserUUID)
		assert.Equal(t, actorUUID, repo.lookedUp)
		// The identity's tenant record is richer than the claim-derived one.
		require.NotNil(t, auth.Tenant)
		assert.Equal(t, "Acme", auth.Tenant.DisplayName)
		assert.Equal(t, callerTenantUUID, auth.Tenant.TenantUUID)
	})

	// The original hole: the actor is both the audit attribution AND the subject
	// of the handlers' tenant-access checks, so naming a victim in another tenant
	// satisfied the check. Moving the actor into the signed token does not remove
	// that risk on its own — the tenant boundary is what does.
	t.Run("an actor in another tenant is refused", func(t *testing.T) {
		actorUUID := uuid.New()
		repo := &fakeActorRepo{actor: actorUser(42, actorUUID, 99, uuid.New(), shared.StatusActive)}

		_, err := grpcAuthContext(repo, caller(actorUUID.String(), callerTenantID))
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "does not belong to this token's tenant")
	})

	t.Run("a deactivated actor is refused", func(t *testing.T) {
		actorUUID := uuid.New()
		repo := &fakeActorRepo{actor: actorUser(42, actorUUID, callerTenantID, callerTenantUUID, "inactive")}

		_, err := grpcAuthContext(repo, caller(actorUUID.String(), callerTenantID))
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "is not active")
	})

	t.Run("an unknown actor is refused", func(t *testing.T) {
		_, err := grpcAuthContext(&fakeActorRepo{}, caller(uuid.NewString(), callerTenantID))
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "does not exist")
	})

	t.Run("a malformed on_behalf_of is refused", func(t *testing.T) {
		repo := &fakeActorRepo{}
		_, err := grpcAuthContext(repo, caller("not-a-uuid", callerTenantID))
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Zero(t, repo.calls)
	})

	// Without a tenant on the token there is no boundary to check the actor
	// against, which is precisely the state the takeover exploited.
	t.Run("an on_behalf_of on a tenant-less token is refused", func(t *testing.T) {
		repo := &fakeActorRepo{}
		_, err := grpcAuthContext(repo, caller(uuid.NewString(), 0))
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "tenant-bound token")
		assert.Zero(t, repo.calls)
	})

	t.Run("an unresolvable actor fails closed rather than running unattributed", func(t *testing.T) {
		_, err := grpcAuthContext(nil, caller(uuid.NewString(), callerTenantID))
		assert.Equal(t, codes.Internal, status.Code(err))

		repo := &fakeActorRepo{err: errors.New("db down")}
		_, err = grpcAuthContext(repo, caller(uuid.NewString(), callerTenantID))
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestGRPCInterceptors_AuthContextAndActorGate(t *testing.T) {
	const callerTenantID = int64(7)

	// serviceToken mints the control-plane's own credential, optionally carrying
	// the signed on_behalf_of claim.
	serviceToken := func(t *testing.T, tenantUUID uuid.UUID, onBehalfOf string) string {
		t.Helper()
		extra := map[string]any{"tenant_id": tenantUUID.String()}
		if onBehalfOf != "" {
			extra[grpcOnBehalfOfClaim] = onBehalfOf
		}
		token, err := jwt.GenerateAccessTokenWithOptions("svc-core", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "core",
			SubjectType: "service",
			ACR:         jwt.ACRLevel2,
			ExtraClaims: extra,
		})
		require.NoError(t, err)
		return token
	}

	withTenantResolver := func(t *testing.T) uuid.UUID {
		t.Helper()
		tenantUUID := uuid.New()
		shared.SetTenantRefResolver(staticServerTenantRef{id: callerTenantID, id2uuid: tenantUUID})
		t.Cleanup(func() { shared.SetTenantRefResolver(nil) })
		return tenantUUID
	}

	t.Run("the tenant claim reaches handlers as an AuthContext", func(t *testing.T) {
		initServerTestJWTKeys(t)
		tenantUUID := withTenantResolver(t)
		const method = "/test.Service/OpenWithTenant"
		grpcServicePermissions[method] = ""
		t.Cleanup(func() { delete(grpcServicePermissions, method) })

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", "Bearer "+serviceToken(t, tenantUUID, "")))

		authCtx, err := authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(2, time.Minute), method)
		require.NoError(t, err)
		auth := middleware.AuthFromContext(authCtx)
		require.NotNil(t, auth.Tenant)
		assert.Equal(t, callerTenantID, auth.Tenant.TenantID)
		assert.Equal(t, tenantUUID, auth.Tenant.TenantUUID)
	})

	// The whole mutation surface returned PermissionDenied with no hint of what
	// an operator had to change; the message now names the missing claim.
	t.Run("an actor-required method names the missing on_behalf_of claim", func(t *testing.T) {
		initServerTestJWTKeys(t)
		// RoleService is core-provisioning, so the actor gate is only reachable on
		// the ecosystem's system instance now; a regular instance refuses the RPC
		// outright before any token is examined.
		withSystemControlPlane(t)
		tenantUUID := withTenantResolver(t)
		method := grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "CreateRole")

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", "Bearer "+serviceToken(t, tenantUUID, "")))

		_, err := authenticateAndAuthorizeGRPC(ctx,
			&Application{AuthorizationService: mockGRPCAuthz{allowed: true}}, newGRPCLimiter(2, time.Minute), method)
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), grpcOnBehalfOfClaim)
		assert.Contains(t, status.Convert(err).Message(), method)
	})

	// Every actor-required RPC must also be a classified method, or the
	// default-deny branch would reject it before the actor gate is ever reached.
	t.Run("every actor-required method is a classified method", func(t *testing.T) {
		for method := range grpcActorRequiredMethods {
			_, classified := grpcServicePermissions[method]
			assert.True(t, classified, "%s is actor-required but not in grpcServicePermissions", method)
		}
	})

	t.Run("on_behalf_of is read off the signed claims", func(t *testing.T) {
		initServerTestJWTKeys(t)
		tenantUUID := withTenantResolver(t)
		actorUUID := uuid.NewString()

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", "Bearer "+serviceToken(t, tenantUUID, actorUUID)))

		caller, err := grpcVerifiedCaller(ctx)
		require.NoError(t, err)
		assert.Equal(t, actorUUID, caller.onBehalfOf)
		assert.Equal(t, callerTenantID, caller.claims.TenantID)
	})

	// A metadata header is caller-controlled, so honouring one would just move
	// the forged-actor hole one hop out of the request body.
	t.Run("an on-behalf-of metadata header is ignored", func(t *testing.T) {
		initServerTestJWTKeys(t)
		tenantUUID := withTenantResolver(t)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			"authorization", "Bearer "+serviceToken(t, tenantUUID, ""),
			grpcOnBehalfOfClaim, uuid.NewString(),
			"x-on-behalf-of", uuid.NewString(),
		))

		caller, err := grpcVerifiedCaller(ctx)
		require.NoError(t, err)
		assert.Empty(t, caller.onBehalfOf)
	})
}

// A nil AuthorizationService used to SKIP the policy decision entirely and let
// the RPC run, so a mis-wired instance served its whole permission-gated gRPC
// surface with no policy check at all. Inverted: it now denies.
func TestGRPCInterceptors_MissingPDPFailsClosed(t *testing.T) {
	initServerTestJWTKeys(t)
	tenantUUID := uuid.New()
	shared.SetTenantRefResolver(staticServerTenantRef{id: 7, id2uuid: tenantUUID})
	t.Cleanup(func() { shared.SetTenantRefResolver(nil) })

	const method = "/test.Service/NeedsPDP"
	grpcServicePermissions[method] = "tenant:read"
	t.Cleanup(func() { delete(grpcServicePermissions, method) })

	token, err := jwt.GenerateAccessTokenWithOptions("svc-core", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
		Service:     "core",
		SubjectType: "service",
		ExtraClaims: map[string]any{"tenant_id": tenantUUID.String()},
	})
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

	_, err = authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(2, time.Minute), method)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.True(t, strings.Contains(status.Convert(err).Message(), "authorization service unavailable"))

	// And the handler must never run.
	handlerRan := false
	interceptor := grpcAuthUnaryInterceptor(&Application{}, newGRPCLimiter(2, time.Minute))
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, func(context.Context, any) (any, error) {
		handlerRan = true
		return "ok", nil
	})
	require.Error(t, err)
	assert.False(t, handlerRan)
}
