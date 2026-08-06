package server

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"

	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/setup/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The control policy has to be exactly as wide as the surface core drives — no
// wider, and crucially no narrower.
//
// Narrower is the failure that hand-maintained lists produce: a method gets a
// new permission, nobody updates the policy, and core fails with PermissionDenied
// on a call it must make — after setup has closed, so it cannot repair itself.
// Rather than trust a list, this derives the requirement from the same two maps
// the interceptor enforces (grpcServicePermissions and
// grpcCoreProvisioningServices) and checks the policy against the real PDP.
func TestControlPolicyCoversEveryCoreProvisioningMethod(t *testing.T) {
	required := requiredCoreProvisioningPermissions()
	require.NotEmpty(t, required, "the core-provisioning surface cannot be empty")

	policy := []iam.PolicyDocument{{
		Version: "v1",
		Statement: []iam.PolicyStatement{{
			Effect:   "allow",
			Action:   seeder.DefaultControlActions,
			Resource: []string{"*"},
		}},
	}}

	var uncovered []string
	for _, permission := range required {
		decision := iam.Evaluate(policy, iam.AuthzRequest{Action: permission, Resource: "*"})
		if !decision.Allowed {
			uncovered = append(uncovered, permission)
		}
	}
	assert.Empty(t, uncovered,
		"the default control policy does not grant these permissions, so core would be denied methods it must call")
}

// The other direction: every action the policy grants must correspond to a
// permission something actually enforces. An allow that matches nothing is a
// standing pre-authorisation for whatever is later created under that name.
//
// Checking only the gRPC surface is correct BECAUSE gRPC is the only surface a
// machine caller can reach — TestRESTIsUnreachableToMachineCallers below pins
// that. But the correct conclusion from "this family is REST-only" is NOT "the
// orchestrator does not need it"; it is "the orchestrator CANNOT have it, so if
// it needs it, that family needs a gRPC service". Reading it the first way is
// how workload-identity-federation:* was briefly removed from the policy,
// leaving an orchestrator unable to configure keyless workloads at all.
func TestControlPolicyGrantsNothingBeyondTheEnforcedSurface(t *testing.T) {
	enforced := make(map[string]struct{})
	for _, permission := range grpcServicePermissions {
		if permission != "" {
			enforced[permission] = struct{}{}
		}
	}

	for _, action := range seeder.DefaultControlActions {
		t.Run(action, func(t *testing.T) {
			policy := []iam.PolicyDocument{{
				Version:   "v1",
				Statement: []iam.PolicyStatement{{Effect: "allow", Action: []string{action}, Resource: []string{"*"}}},
			}}
			for permission := range enforced {
				if iam.Evaluate(policy, iam.AuthzRequest{Action: permission, Resource: "*"}).Allowed {
					return
				}
			}
			t.Errorf("%q grants a permission family no core-provisioning RPC enforces; an allow that matches nothing is a standing pre-authorisation for whatever is later created under that name", action)
		})
	}
}

// requiredCoreProvisioningPermissions is every non-empty permission guarding a
// method on a core-provisioning service, excluding the peer-service reads that
// every instance serves.
func requiredCoreProvisioningPermissions() []string {
	seen := make(map[string]struct{})
	for method, permission := range grpcServicePermissions {
		if permission == "" {
			continue
		}
		if _, isPeer := grpcPeerServiceMethods[method]; isPeer {
			continue
		}
		service := grpcServiceNameFromMethod(method)
		if _, isProvisioning := grpcCoreProvisioningServices[service]; !isProvisioning {
			continue
		}
		seen[permission] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for permission := range seen {
		out = append(out, permission)
	}
	sort.Strings(out)
	return out
}

// grpcServiceNameFromMethod turns "/pkg.Service/Method" into "pkg.Service".
func grpcServiceNameFromMethod(method string) string {
	trimmed := strings.TrimPrefix(method, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx < 0 {
		return trimmed
	}
	return trimmed[:idx]
}

// The permission middleware on the REST management surface requires a resolved
// USER principal. An orchestrator is a machine and has none, so REST is closed
// to it entirely — a permission granted for a REST-only family can never be
// exercised by a machine caller, no matter what the policy says.
//
// This is pinned as a test because it is the fact that makes the coverage check
// above sufficient, and because it is not obvious from either map: the policy
// looks like it grants the capability, and the route looks like it accepts the
// permission.
func TestRESTIsUnreachableToMachineCallers(t *testing.T) {
	handler := middleware.PermissionMiddleware([]string{"workload-identity-federation:read"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	)

	// A service principal: verified claims, a tenant, and deliberately no user.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workload-identity-federations", nil)
	req = req.WithContext(middleware.WithAuthContextValue(req.Context(), &authctx.AuthContext{
		Tenant: &authctx.AuthTenant{TenantID: 1},
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"if this ever passes, machine callers can reach REST and the gRPC-only coverage check above is no longer sufficient")
}
