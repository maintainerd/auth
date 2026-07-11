package server

import authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"

// grpcServicePermissions maps each SERVED gRPC method to the permission the
// caller's service account must hold. A value of "" means the method is
// service-account-authenticated but not PDP-gated (verification-style reads). The
// interceptor default-denies any maintainerd.auth.v1 method absent from this map
// (except bootstrap methods, see grpcBootstrapMethods), so this map defines the
// entire authenticated gRPC surface.
//
// Scope (deliberately narrow):
//   - CORE provisioning: tenant, service, api, permission, policy, role, client.
//   - Peer services / BFF: user + profile reads, authorization (PDP), introspection.
//
// Tenant admin/UX/comms operations (branding, templates, email/SMS config,
// webhooks, IdP, registration flows, invites, security settings, IP rules, audit
// browsing, tenant settings) are REST control-plane only — no gRPC handler is
// registered for them (see grpc.go and grpcUnauthenticatedServices).
var grpcServicePermissions = map[string]string{
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "GetDefaultTenant"):       "",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "ListTenants"):            "tenant:read",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "GetTenant"):              "tenant:read",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "CreateTenant"):           "tenant:create",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "UpdateTenant"):           "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantStatus"):        "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "DeleteTenant"):           "tenant:delete",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "ListTenantMembers"):      "tenant:read",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "AddTenantMember"):        "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "UpdateTenantMemberRole"): "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "RemoveTenantMember"):     "tenant:update",

	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "GetMyPolicyBundle"):   "",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "ListServices"):        "service:read",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "GetService"):          "service:read",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "CreateService"):       "service:create",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "UpdateService"):       "service:update",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "SetServiceStatus"):    "service:update",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "DeleteService"):       "service:delete",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "AssignServicePolicy"): "service:policy:assign",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "RemoveServicePolicy"): "service:policy:remove",

	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "ListAPIs"):     "api:read",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "GetAPI"):       "api:read",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "CreateAPI"):    "api:create",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "UpdateAPI"):    "api:update",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "SetAPIStatus"): "api:update",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "DeleteAPI"):    "api:delete",

	grpcMethod(authv1.PermissionService_ServiceDesc.ServiceName, "ListPermissions"):     "permission:read",
	grpcMethod(authv1.PermissionService_ServiceDesc.ServiceName, "GetPermission"):       "permission:read",
	grpcMethod(authv1.PermissionService_ServiceDesc.ServiceName, "CreatePermission"):    "permission:create",
	grpcMethod(authv1.PermissionService_ServiceDesc.ServiceName, "UpdatePermission"):    "permission:update",
	grpcMethod(authv1.PermissionService_ServiceDesc.ServiceName, "SetPermissionStatus"): "permission:update",
	grpcMethod(authv1.PermissionService_ServiceDesc.ServiceName, "DeletePermission"):    "permission:delete",

	grpcMethod(authv1.PolicyService_ServiceDesc.ServiceName, "ListPolicies"):       "policy:read",
	grpcMethod(authv1.PolicyService_ServiceDesc.ServiceName, "GetPolicy"):          "policy:read",
	grpcMethod(authv1.PolicyService_ServiceDesc.ServiceName, "ListPolicyServices"): "policy:read",
	grpcMethod(authv1.PolicyService_ServiceDesc.ServiceName, "CreatePolicy"):       "policy:create",
	grpcMethod(authv1.PolicyService_ServiceDesc.ServiceName, "UpdatePolicy"):       "policy:update",
	grpcMethod(authv1.PolicyService_ServiceDesc.ServiceName, "SetPolicyStatus"):    "policy:update",
	grpcMethod(authv1.PolicyService_ServiceDesc.ServiceName, "DeletePolicy"):       "policy:delete",

	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "ListRoles"):            "role:read",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "GetRole"):              "role:read",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "CreateRole"):           "role:create",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "UpdateRole"):           "role:update",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "SetRoleStatus"):        "role:update",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "DeleteRole"):           "role:delete",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "ListRolePermissions"):  "role:read",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "AddRolePermissions"):   "role:permission:create",
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "RemoveRolePermission"): "role:permission:delete",

	grpcMethod(authv1.AuthorizationService_ServiceDesc.ServiceName, "Authorize"): "",

	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "ListClients"):               "client:read",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "GetClient"):                 "client:read",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "GetClientSecret"):           "client:secret:read",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "RotateClientSecret"):        "client:secret:rotate",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "GetClientConfig"):           "client:config:read",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "CreateClient"):              "client:create",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "UpdateClient"):              "client:update",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "SetClientStatus"):           "client:update",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "DeleteClient"):              "client:delete",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "ListClientURIs"):            "client:uri:read",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "CreateClientURI"):           "client:uri:create",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "UpdateClientURI"):           "client:uri:update",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "DeleteClientURI"):           "client:uri:delete",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "ListClientAPIs"):            "client:api:read",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "AddClientAPIs"):             "client:api:create",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "RemoveClientAPI"):           "client:api:delete",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "ListClientAPIPermissions"):  "client:api:permission:read",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "AddClientAPIPermissions"):   "client:api:permission:create",
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "RemoveClientAPIPermission"): "client:api:permission:delete",

	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "ListUsers"):               "user:read",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "GetUser"):                 "user:read",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "CreateUser"):              "user:create",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "UpdateUser"):              "user:update",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "SetUserStatus"):           "user:update",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "VerifyUserEmail"):         "user:update",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "VerifyUserPhone"):         "user:update",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "CompleteUserAccount"):     "user:update",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "DeleteUser"):              "user:delete",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "ForceUserPasswordChange"): "user:update",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "ListUserRoles"):           "user:read",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "ListUserIdentities"):      "user:read",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "AssignUserRoles"):         "user:create",
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "RemoveUserRole"):          "user:create",

	grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "ListUserProfiles"):      "user:read",
	grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "GetUserProfile"):        "user:read",
	grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "CreateUserProfile"):     "user:update",
	grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "UpdateUserProfile"):     "user:update",
	grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "SetDefaultUserProfile"): "user:update",
	grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "DeleteUserProfile"):     "user:delete",

	grpcMethod(authv1.OAuthIntrospectionService_ServiceDesc.ServiceName, "Introspect"): "",
}

var grpcStepUpMethods = map[string]struct{}{
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantStatus"):       {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "DeleteTenant"):          {},
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "GetClientSecret"):       {},
	grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "RotateClientSecret"):    {},
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "SetUserStatus"):           {},
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "DeleteUser"):              {},
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "ForceUserPasswordChange"): {},
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "AssignUserRoles"):         {},
	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "RemoveUserRole"):          {},
}

// grpcBootstrapMethods are the SetupService RPCs used by CORE to provision a
// system auth instance (system tenant + admin + control service). They cannot use
// the normal JWT/PDP path because no accounts or service principals exist at first
// boot; instead the interceptor gates them with the pre-shared SETUP_BOOTSTRAP_TOKEN
// (see authorizeSetupBootstrap), and the setup service locks them once the system
// tenant is active (ensureSetupOpen).
var grpcBootstrapMethods = map[string]struct{}{
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "GetSetupStatus"):         {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CreateTenant"):           {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CreateAdmin"):            {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CreateProfile"):          {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "RegisterControlService"): {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CompleteSetup"):          {},
}

func grpcMethod(service string, method string) string {
	return "/" + service + "/" + method
}
