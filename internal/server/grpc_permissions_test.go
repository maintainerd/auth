package server

import (
	"testing"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// grpcUnauthenticatedServices lists the gRPC services whose RPCs are
// intentionally NOT in grpcServicePermissions. The authz interceptor treats any
// method absent from the registry as unprotected (fail-open, no authentication),
// so this allowlist must stay deliberately tiny and well-justified.
//
// SetupService is bootstrap-only: its mutating RPCs are gated by the persisted
// setup-complete lock inside the setup service (ensureSetupOpen), not the PDP,
// because no policy can exist before setup runs.
var grpcUnauthenticatedServices = map[string]struct{}{
	authv1.SetupService_ServiceDesc.ServiceName: {},
}

// TestGRPCServicePermissions_EveryAppRPCIsRegistered walks every RPC defined in
// the maintainerd.auth.v1 proto package and asserts it is present in
// grpcServicePermissions, unless its service is on the explicit
// grpcUnauthenticatedServices allowlist. Without this guard, adding a new RPC to
// a proto/handler without a matching registry entry would silently expose a
// fully unauthenticated endpoint — the opposite of the documented default-deny.
func TestGRPCServicePermissions_EveryAppRPCIsRegistered(t *testing.T) {
	const authPackage = "maintainerd.auth.v1"
	checked := 0

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Package() != authPackage {
			return true
		}
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			serviceName := string(svc.FullName())
			if _, allowed := grpcUnauthenticatedServices[serviceName]; allowed {
				continue
			}
			methods := svc.Methods()
			for j := 0; j < methods.Len(); j++ {
				method := grpcMethod(serviceName, string(methods.Get(j).Name()))
				checked++
				if _, registered := grpcServicePermissions[method]; !registered {
					t.Errorf("RPC %s has no grpcServicePermissions entry; the authz interceptor would serve it with NO authentication (fail-open). Add a permission string, or \"\" for a service-account-only verification read.", method)
				}
			}
		}
		return true
	})

	// Guard against the discovery loop silently finding nothing (e.g. descriptors
	// not linked), which would make the whole test vacuously pass.
	assert.Greater(t, checked, 100, "expected to discover the full maintainerd.auth.v1 RPC surface")
}

func TestGRPCServicePermissions_InviteIsPolicyGated(t *testing.T) {
	method := grpcMethod(authv1.InviteService_ServiceDesc.ServiceName, "SendInvite")

	permission, protected := grpcServicePermissions[method]

	assert.True(t, protected)
	assert.Equal(t, "user:invite", permission)
}

func TestGRPCServicePermissions_UserProfilesArePolicyGated(t *testing.T) {
	for method, expected := range map[string]string{
		"ListUserProfiles":      "user:read",
		"GetUserProfile":        "user:read",
		"CreateUserProfile":     "user:update",
		"UpdateUserProfile":     "user:update",
		"SetDefaultUserProfile": "user:update",
		"DeleteUserProfile":     "user:delete",
	} {
		t.Run(method, func(t *testing.T) {
			permission, protected := grpcServicePermissions[grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, method)]

			assert.True(t, protected)
			assert.Equal(t, expected, permission)
		})
	}
}

func TestGRPCServicePermissions_VerificationReadsStayServiceAccountOnly(t *testing.T) {
	for name, method := range map[string]string{
		"authorize":         grpcMethod(authv1.AuthorizationService_ServiceDesc.ServiceName, "Authorize"),
		"introspect":        grpcMethod(authv1.OAuthIntrospectionService_ServiceDesc.ServiceName, "Introspect"),
		"defaultTenant":     grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "GetDefaultTenant"),
		"getMyPolicyBundle": grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "GetMyPolicyBundle"),
	} {
		t.Run(name, func(t *testing.T) {
			permission, protected := grpcServicePermissions[method]

			assert.True(t, protected)
			assert.Empty(t, permission)
		})
	}
}
