package server

import (
	"testing"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
)

func TestGRPCServicePermissions_InviteIsPolicyGated(t *testing.T) {
	method := grpcMethod(authv1.InviteService_ServiceDesc.ServiceName, "SendInvite")

	permission, protected := grpcServicePermissions[method]

	assert.True(t, protected)
	assert.Equal(t, "user:invite", permission)
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
