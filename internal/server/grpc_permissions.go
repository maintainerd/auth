package server

import authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"

var grpcServicePermissions = map[string]string{
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "GetDefaultTenant"):       "",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "GetTenantByIdentifier"):  "",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "ListTenants"):            "tenant:read",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "GetTenant"):              "tenant:read",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "CreateTenant"):           "tenant:create",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "UpdateTenant"):           "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantStatus"):        "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantPublic"):        "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "DeleteTenant"):           "tenant:delete",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "ListTenantMembers"):      "tenant:read",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "AddTenantMember"):        "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "UpdateTenantMemberRole"): "tenant:update",
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "RemoveTenantMember"):     "tenant:update",

	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetRateLimitConfig"):      "tenant-setting:read",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "UpdateRateLimitConfig"):   "tenant-setting:update",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetAuditConfig"):          "tenant-setting:read",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "UpdateAuditConfig"):       "tenant-setting:update",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetMaintenanceConfig"):    "tenant-setting:read",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "UpdateMaintenanceConfig"): "tenant-setting:update",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "GetFeatureFlags"):         "tenant-setting:read",
	grpcMethod(authv1.TenantSettingService_ServiceDesc.ServiceName, "UpdateFeatureFlags"):      "tenant-setting:update",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "ListServices"):                  "service:read",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "GetService"):                    "service:read",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "CreateService"):                 "service:create",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "UpdateService"):                 "service:update",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "SetServiceStatus"):              "service:update",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "DeleteService"):                 "service:delete",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "AssignServicePolicy"):           "service:policy:assign",
	grpcMethod(authv1.ServiceService_ServiceDesc.ServiceName, "RemoveServicePolicy"):           "service:policy:remove",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "ListAPIs"):                          "api:read",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "GetAPI"):                            "api:read",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "CreateAPI"):                         "api:create",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "UpdateAPI"):                         "api:update",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "SetAPIStatus"):                      "api:update",
	grpcMethod(authv1.APIService_ServiceDesc.ServiceName, "DeleteAPI"):                         "api:delete",
}

var grpcStepUpMethods = map[string]struct{}{
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantStatus"): {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantPublic"): {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "DeleteTenant"):    {},
}

func grpcMethod(service string, method string) string {
	return "/" + service + "/" + method
}
