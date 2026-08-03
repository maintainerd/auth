package server

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
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
	changes := grpcAuditChanges("/svc/CreateService", &middleware.JWTClaims{Service: "core"})
	assert.Contains(t, changes, `"principal":"core"`)
	assert.Contains(t, changes, `"method":"/svc/CreateService"`)

	// A token with no svc claim falls back to the subject rather than logging blank.
	changes = grpcAuditChanges("/svc/CreateService", &middleware.JWTClaims{Sub: "client-abc"})
	assert.Contains(t, changes, `"principal":"client-abc"`)
}
