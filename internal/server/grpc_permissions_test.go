package server

import (
	"testing"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// grpcUnauthenticatedServices lists gRPC services whose RPCs are intentionally
// NOT in grpcServicePermissions AND not bootstrap-token gated. The interceptor
// default-denies any unclassified maintainerd.auth.v1 method, so a service belongs
// here ONLY when no handler is registered for it (the server returns UNIMPLEMENTED
// before any interceptor runs — zero exposure). This list must stay tiny.
//
// APIKeyService is decommissioned (replaced by M2M OAuth client credentials); the
// proto is retained for wire compat but no handler is registered.
//
// The following are admin/UX/comms operations that were REMOVED from the gRPC
// surface (they live on the REST control plane consumed by the console). Their
// proto/handlers are retained in-package but no gRPC handler is registered, so
// they return UNIMPLEMENTED before any interceptor runs.
//
// SetupService is NOT here: it is bootstrap-token gated via grpcBootstrapMethods.
var grpcUnauthenticatedServices = map[string]struct{}{
	authv1.APIKeyService_ServiceDesc.ServiceName:            {},
	authv1.TenantSettingService_ServiceDesc.ServiceName:     {},
	authv1.IdentityProviderService_ServiceDesc.ServiceName:  {},
	authv1.RegistrationFlowService_ServiceDesc.ServiceName:  {},
	authv1.InviteService_ServiceDesc.ServiceName:            {},
	authv1.SecuritySettingService_ServiceDesc.ServiceName:   {},
	authv1.IPRestrictionRuleService_ServiceDesc.ServiceName: {},
	authv1.BrandingService_ServiceDesc.ServiceName:          {},
	authv1.EmailTemplateService_ServiceDesc.ServiceName:     {},
	authv1.SMSTemplateService_ServiceDesc.ServiceName:       {},
	authv1.EmailConfigService_ServiceDesc.ServiceName:       {},
	authv1.SMSConfigService_ServiceDesc.ServiceName:         {},
	authv1.WebhookEndpointService_ServiceDesc.ServiceName:   {},
	authv1.AuthEventService_ServiceDesc.ServiceName:         {},
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
				_, registered := grpcServicePermissions[method]
				_, bootstrap := grpcBootstrapMethods[method]
				if !registered && !bootstrap {
					t.Errorf("RPC %s has no grpcServicePermissions entry and is not a bootstrap method; the authz interceptor default-denies it. Add a permission string (\"\" for a service-account-only read), a grpcBootstrapMethods entry, or an allowlist entry if the handler is unregistered.", method)
				}
			}
		}
		return true
	})

	// Guard against the discovery loop silently finding nothing (e.g. descriptors
	// not linked), which would make the whole test vacuously pass.
	assert.Greater(t, checked, 80, "expected to discover the full maintainerd.auth.v1 RPC surface")
}

func TestGRPCServicePermissions_RemovedAdminServicesAreUnregistered(t *testing.T) {
	// These admin/UX/comms services were removed from the gRPC surface: they must
	// have NO permission entry (they are allowlisted as unregistered → UNIMPLEMENTED).
	for _, method := range []string{
		grpcMethod(authv1.InviteService_ServiceDesc.ServiceName, "SendInvite"),
		grpcMethod(authv1.BrandingService_ServiceDesc.ServiceName, "UpdateBranding"),
		grpcMethod(authv1.WebhookEndpointService_ServiceDesc.ServiceName, "CreateWebhookEndpoint"),
		grpcMethod(authv1.SecuritySettingService_ServiceDesc.ServiceName, "UpdateMFAConfig"),
		grpcMethod(authv1.AuthEventService_ServiceDesc.ServiceName, "ListAuthEvents"),
		grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetFeatureFlags"),
	} {
		t.Run(method, func(t *testing.T) {
			_, protected := grpcServicePermissions[method]
			assert.False(t, protected, "removed admin RPC must not be in grpcServicePermissions")
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
