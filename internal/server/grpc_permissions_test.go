package server

import (
	"testing"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// TestGRPCServicePermissions_EveryAppRPCIsRegistered walks every RPC defined in
// the maintainerd.auth.v1 proto package and asserts it is present in
// grpcServicePermissions or grpcBootstrapMethods. Without this guard, adding a
// new RPC to a proto/handler without a matching registry entry would silently
// expose a fully unauthenticated endpoint — the opposite of the documented
// default-deny.
//
// There is deliberately no "unregistered service" allowlist any more. It used to
// exempt the twelve REST-only services whose RPCs the contract declared but no
// handler served; those service blocks are gone from the protos, so every RPC
// that reaches this loop is one the server actually answers and every one of
// them must be classified.
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
			methods := svc.Methods()
			for j := 0; j < methods.Len(); j++ {
				method := grpcMethod(serviceName, string(methods.Get(j).Name()))
				checked++
				_, registered := grpcServicePermissions[method]
				_, bootstrap := grpcBootstrapMethods[method]
				if !registered && !bootstrap {
					t.Errorf("RPC %s has no grpcServicePermissions entry and is not a bootstrap method; the authz interceptor default-denies it. Add a permission string (\"\" for a service-account-only read) or a grpcBootstrapMethods entry — and if the RPC is not meant to be served at all, delete it from the proto rather than shipping an UNIMPLEMENTED promise.", method)
				}
			}
		}
		return true
	})

	// Guard against the discovery loop silently finding nothing (e.g. descriptors
	// not linked), which would make the whole test vacuously pass.
	assert.Greater(t, checked, 80, "expected to discover the full maintainerd.auth.v1 RPC surface")
}

// The admin/UX/comms surfaces are REST control-plane only. They must not come
// back as bare proto declarations: this test used to assert only that they had no
// entry in grpcServicePermissions, which was satisfied by a service that was
// declared, unregistered and answering UNIMPLEMENTED — exactly the state being
// fixed. Assert on the contract itself instead, so re-adding one of these
// services forces a handler (TestGRPCContractIsFullyServed) and a permission
// entry (TestGRPCServicePermissions_EveryAppRPCIsRegistered) in the same change.
func TestGRPCContract_RESTOnlyServicesAreNotDeclared(t *testing.T) {
	declared := make(map[string]struct{})
	for _, name := range grpcContractServiceNames(t) {
		declared[name] = struct{}{}
	}
	require.NotEmpty(t, declared, "the generated auth protos must be linked into this test binary")

	for _, name := range []string{
		"AuthEventService",
		"BrandingService",
		"EmailConfigService",
		"EmailTemplateService",
		"IPRestrictionRuleService",
		"IdentityProviderService",
		"InviteService",
		"RegistrationFlowService",
		"SMSConfigService",
		"SMSTemplateService",
		"SecuritySettingService",
		"WebhookEndpointService",
	} {
		t.Run(name, func(t *testing.T) {
			_, found := declared["maintainerd.auth.v1."+name]
			assert.False(t, found, "REST-only surface must not be declared in the gRPC contract unless a handler is registered for it")
		})
	}
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
