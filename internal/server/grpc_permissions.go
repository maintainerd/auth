package server

import (
	"strings"

	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
)

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
// Tenant OPERATIONAL settings (rate limit, audit, maintenance) ARE exposed to
// core via TenantSettingService — core owns tenant lifecycle and these are
// management, not security, concerns. SECURITY settings (password, MFA, lockout,
// session, token, threat, IP rules) are REST/console-only, as are tenant
// admin/UX/comms operations (branding, templates, email/SMS config, webhooks,
// IdP, registration flows, invites, audit browsing) — they are no longer
// declared in the proto at all, so there is nothing here to gate for them.
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

	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetRateLimitConfig"):      "tenant-setting:read",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "UpdateRateLimitConfig"):   "tenant-setting:update",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetAuditConfig"):          "tenant-setting:read",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "UpdateAuditConfig"):       "tenant-setting:update",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetMaintenanceConfig"):    "tenant-setting:read",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "UpdateMaintenanceConfig"): "tenant-setting:update",
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

// grpcActorRequiredMethods are the RPCs whose handlers change state ON BEHALF OF
// a human: the acting user is both the audit attribution and the subject of the
// membership/escalation guards (iam ValidateTenantAccess, tenant authorizeManager).
// A bare service token cannot satisfy them, so the interceptor rejects the call
// up front with one message naming the missing on_behalf_of claim, rather than
// letting each handler fail with its own wording after the work has started.
//
// Keep in sync with the handlers that resolve an actor from the token:
// iam/grpc_helpers.go iamActorUserUUID and tenant/handler_tenant_grpc.go
// grpcActorUserID.
var grpcActorRequiredMethods = map[string]struct{}{
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "DeleteTenant"):           {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "AddTenantMember"):        {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "UpdateTenantMemberRole"): {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "RemoveTenantMember"):     {},

	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "CreateRole"):           {},
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "UpdateRole"):           {},
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "SetRoleStatus"):        {},
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "DeleteRole"):           {},
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "AddRolePermissions"):   {},
	grpcMethod(authv1.RoleService_ServiceDesc.ServiceName, "RemoveRolePermission"): {},

	grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "AssignUserRoles"): {},
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
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureControlClient"):    {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureResourceAPI"):      {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureRole"):             {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureConsoleClient"):    {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CompleteSetup"):          {},
}

// grpcCoreProvisioningServices are the gRPC services that exist so CORE can
// provision the maintainerd ecosystem: tenant records and their operational
// settings, the IAM primitives (service/api/permission/policy/role) and clients.
//
// They are served ONLY by the instance declared as the ecosystem's system IAM
// (config.InstanceRoleSystem). Core can provision many auth instances — one
// system IAM plus any number of ordinary, disposable instances a developer runs
// for their own application — and every one of them used to expose this whole
// surface. On an ordinary instance that surface is an orchestrator API nobody
// there is supposed to hold: a tenant admin who can reach the port is talking to
// the same RPCs core uses to mint tenants and clients. Ordinary instances are
// administered through the console/REST surface instead.
//
// Classification is by SERVICE, not by method, so a newly added RPC on one of
// these services is system-only by default. Opting a method out is a deliberate
// entry in grpcPeerServiceMethods.
// Workload identity federation: the trust rules that let a provisioned workload
// authenticate with a platform-issued identity instead of a credential the
// orchestrator had to mint and inject. Each RPC carries its own permission —
// creating a federation is a standing authorization grant, so it is not lumped
// in with reads.
var grpcWorkloadIdentityPermissions = map[string]string{
	grpcMethod(authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName, "ListWorkloadIdentityFederations"):  "workload-identity-federation:read",
	grpcMethod(authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName, "GetWorkloadIdentityFederation"):    "workload-identity-federation:read",
	grpcMethod(authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName, "CreateWorkloadIdentityFederation"): "workload-identity-federation:create",
	grpcMethod(authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName, "UpdateWorkloadIdentityFederation"): "workload-identity-federation:update",
	grpcMethod(authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName, "DeleteWorkloadIdentityFederation"): "workload-identity-federation:delete",
}

var grpcCoreProvisioningServices = map[string]struct{}{
	authv1.TenantService_ServiceDesc.ServiceName:        {},
	authv1.TenantSettingService_ServiceDesc.ServiceName: {},
	authv1.ServiceService_ServiceDesc.ServiceName:       {},
	authv1.APIService_ServiceDesc.ServiceName:           {},
	authv1.PermissionService_ServiceDesc.ServiceName:    {},
	authv1.PolicyService_ServiceDesc.ServiceName:        {},
	authv1.RoleService_ServiceDesc.ServiceName:          {},
	authv1.ClientService_ServiceDesc.ServiceName:        {},
	// Registering a federation says "tokens from this issuer may act as this
	// client". On an ordinary instance that is an orchestrator capability nobody
	// there should hold.
	authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName: {},
}

// grpcPeerServiceMethods live on a core-provisioning service but are read-only
// calls a PEER SERVICE makes to run itself, not provisioning: a service fetching
// its own policy bundle, or resolving the instance's default tenant. Every auth
// instance has peer services, so restricting these to the system instance would
// break the ordinary instances rather than protect them.
//
// Both are already "" in grpcServicePermissions — service-account-authenticated,
// non-mutating. Do not add a mutating RPC here.
var grpcPeerServiceMethods = map[string]struct{}{
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "GetMyPolicyBundle"): {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "GetDefaultTenant"):   {},
}

// grpcSystemInstanceOnlyMethods are system-only RPCs that sit on a service which
// is otherwise available to every instance.
//
// RegisterControlService registers CORE ITSELF (and later maintainerd-docker,
// maintainerd-k8s, …) as a service principal of this IAM. That only ever means
// something on the ecosystem's system IAM; on an ordinary instance it would
// install an ecosystem control principal in a developer's throwaway IAM. The
// rest of SetupService stays open to any role because it is how core bootstraps
// EVERY instance it provisions, and it is separately gated by the per-instance
// single-use bootstrap credential and locked once the system tenant is active.
var grpcSystemInstanceOnlyMethods = map[string]struct{}{
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "RegisterControlService"): {},
	// The orchestrator-provisioning RPCs install the ecosystem's control principal
	// and its policy. On an ordinary, disposable instance that is an orchestrator
	// takeover surface in a developer's throwaway IAM, so they are confined to the
	// instance declared as the ecosystem's system IAM — the same reasoning as
	// RegisterControlService, which they complete.
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureControlClient"): {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureResourceAPI"):   {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureRole"):          {},
	grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "EnsureConsoleClient"): {},
}

// grpcRequiresSystemInstance reports whether method may only be served by the
// ecosystem's system auth instance.
func grpcRequiresSystemInstance(method string) bool {
	if _, peer := grpcPeerServiceMethods[method]; peer {
		return false
	}
	if _, systemOnly := grpcSystemInstanceOnlyMethods[method]; systemOnly {
		return true
	}
	_, coreProvisioning := grpcCoreProvisioningServices[grpcServiceOfMethod(method)]
	return coreProvisioning
}

// grpcServiceOfMethod extracts the service name from a "/pkg.Service/Method"
// path, returning "" for anything that is not one — an unparsable path must not
// accidentally match a service entry.
func grpcServiceOfMethod(method string) string {
	trimmed := strings.TrimPrefix(method, "/")
	if len(trimmed) == len(method) {
		return ""
	}
	slash := strings.LastIndex(trimmed, "/")
	if slash <= 0 {
		return ""
	}
	return trimmed[:slash]
}

func grpcMethod(service string, method string) string {
	return "/" + service + "/" + method
}

// The workload-identity permissions are merged into the enforced map rather than
// written inline, so the federation surface is declared in one place next to the
// reasoning for it and cannot be half-registered.
func init() {
	for method, permission := range grpcWorkloadIdentityPermissions {
		grpcServicePermissions[method] = permission
	}
}
