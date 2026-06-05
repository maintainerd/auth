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
}

var grpcStepUpMethods = map[string]struct{}{
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantStatus"): {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "SetTenantPublic"): {},
	grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "DeleteTenant"):    {},
}

func grpcMethod(service string, method string) string {
	return "/" + service + "/" + method
}
