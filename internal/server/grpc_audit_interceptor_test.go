package server

import (
	"context"
	"net"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/auditlog"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

// None of the 12 gRPC handlers logged an audit row, so the control plane — the actor
// most worth auditing — was invisible in management_audit_log. Doing it centrally
// means a new RPC is covered by construction rather than by remembering.
func TestGRPCAuditTarget(t *testing.T) {
	mutating := map[string][2]string{
		"/maintainerd.auth.v1.ServiceService/CreateService":        {"create", "service"},
		"/maintainerd.auth.v1.ServiceService/UpdateService":        {"update", "service"},
		"/maintainerd.auth.v1.ServiceService/DeleteService":        {"delete", "service"},
		"/maintainerd.auth.v1.ServiceService/SetServiceStatus":     {"update", "servicestatus"},
		"/maintainerd.auth.v1.ServiceService/AssignPolicy":         {"assign", "policy"},
		"/maintainerd.auth.v1.ServiceService/RemovePolicy":         {"remove", "policy"},
		"/maintainerd.auth.v1.ClientService/RotateClientSecret":    {"rotate", "clientsecret"},
		"/maintainerd.auth.v1.SetupService/RegisterControlService": {"create", "controlservice"},
	}
	for method, want := range mutating {
		action, resourceType, ok := grpcAuditTarget(method)
		assert.True(t, ok, method)
		assert.Equal(t, want[0], action, method)
		assert.Equal(t, want[1], resourceType, method)
	}

	// Reads are not audited: they would swamp the log, and the REST surface does not
	// audit them either.
	for _, method := range []string{
		"/maintainerd.auth.v1.ServiceService/ListServices",
		"/maintainerd.auth.v1.ServiceService/GetService",
		"/maintainerd.auth.v1.AuthorizationService/Authorize",
		"/maintainerd.auth.v1.OAuthService/Introspect",
	} {
		_, _, ok := grpcAuditTarget(method)
		assert.False(t, ok, method)
	}

	// Malformed method strings must not panic or produce a bogus row.
	for _, method := range []string{"", "no-slash", "/"} {
		_, _, ok := grpcAuditTarget(method)
		assert.False(t, ok, method)
	}
}

// The row has to name the calling principal, or "who changed this" is unanswerable
// for the control plane.
func TestGRPCAuditChanges_NamesThePrincipal(t *testing.T) {
	changes := grpcAuditChanges("/svc/CreateService", grpcAuditPrincipal(&middleware.JWTClaims{Service: "core"}))
	assert.Contains(t, changes, `"principal":"core"`)
	assert.Contains(t, changes, `"method":"/svc/CreateService"`)

	// A token with no svc claim falls back to the subject rather than logging blank.
	changes = grpcAuditChanges("/svc/CreateService", grpcAuditPrincipal(&middleware.JWTClaims{Sub: "client-abc"}))
	assert.Contains(t, changes, `"principal":"client-abc"`)
}

// captureAuditLogger records every entry the interceptor writes.
type captureAuditLogger struct {
	entries []auditlog.LogEntry
}

func (c *captureAuditLogger) Log(_ context.Context, entry auditlog.LogEntry) error {
	c.entries = append(c.entries, entry)
	return nil
}

// The SetupService bootstrap RPCs authenticate by x-setup-token + mTLS — no JWT —
// so the nil-claims bail dropped them and tenant/admin/control-service creation,
// the most security-relevant mutations of an install's life, never reached
// management_audit_log. With nil claims a setup mutation must still produce a
// row, attributed to the mTLS peer identity and with no tenant (none exists yet).
func TestGRPCAuditInterceptor_SetupWindowIsAudited(t *testing.T) {
	logger := &captureAuditLogger{}
	interceptor := grpcAuditUnaryInterceptor(&Application{AuditLogger: logger})
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	// No JWT claims in context; the peer is what the setup interceptor verified.
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 7), Port: 4443},
	})

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CreateTenant"),
	}, handler)
	require.NoError(t, err)

	require.Len(t, logger.entries, 1, "a setup-window mutation must produce an audit row")
	entry := logger.entries[0]
	assert.Equal(t, "create", entry.Action)
	assert.Equal(t, "tenant", entry.ResourceType)
	assert.Equal(t, "success", entry.Outcome)
	// No tenant exists during the setup window; the logger records NULL for 0.
	assert.Zero(t, entry.TenantID)
	assert.Contains(t, entry.Changes, `"principal":"peer:10.0.0.7:4443"`)

	// The Ensure* convergence RPCs mutate too (get-or-create); they must not fall
	// through the prefix map into the "not mutating" bucket.
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureConsoleClient"),
	}, handler)
	require.NoError(t, err)
	require.Len(t, logger.entries, 2)
	assert.Equal(t, "create", logger.entries[1].Action)
	assert.Equal(t, "consoleclient", logger.entries[1].ResourceType)

	// A peer-less context (no mTLS info survived) still names the credential
	// class rather than logging a blank principal.
	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CreateAdmin"),
	}, handler)
	require.NoError(t, err)
	require.Len(t, logger.entries, 3)
	assert.Contains(t, logger.entries[2].Changes, `"principal":"setup-token"`)
}

// The setup exemption is exactly that — an exemption for the bootstrap method
// set. Any other claims-less call keeps the existing bail: it was rejected
// before reaching a handler, so there is no mutation to audit.
func TestGRPCAuditInterceptor_RegularUnauthenticatedCallIsNotAudited(t *testing.T) {
	logger := &captureAuditLogger{}
	interceptor := grpcAuditUnaryInterceptor(&Application{AuditLogger: logger})
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "CreateClient"),
	}, handler)
	require.NoError(t, err)
	assert.Empty(t, logger.entries, "a claims-less non-setup call must not produce an audit row")
}

// An authenticated mutation keeps its tenant and token principal — the setup
// path must not have changed the claims path.
func TestGRPCAuditInterceptor_ClaimsPathUnchanged(t *testing.T) {
	logger := &captureAuditLogger{}
	interceptor := grpcAuditUnaryInterceptor(&Application{AuditLogger: logger})
	handler := func(ctx context.Context, req any) (any, error) { return "ok", nil }

	ctx := middleware.ContextWithJWTClaims(context.Background(), &middleware.JWTClaims{
		Service:  "core",
		TenantID: 42,
	})
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "CreateClient"),
	}, handler)
	require.NoError(t, err)
	require.Len(t, logger.entries, 1)
	assert.Equal(t, int64(42), logger.entries[0].TenantID)
	assert.Contains(t, logger.entries[0].Changes, `"principal":"core"`)
}
