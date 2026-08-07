package server

import (
	"context"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gRPC carries two unrelated surfaces over one socket: the authorization PDP,
// token introspection and peer reads (runtime, wanted by anyone with more than
// one service), and the provisioning RPCs (orchestration, dangerous).
//
// They are gated separately so an organisation can run a PDP without exposing
// tenant, client and policy creation. Collapsing them would mean taking the
// dangerous surface to obtain the safe one.
func TestGRPCSurfaceSplit(t *testing.T) {
	names := func(services []grpcService) map[string]bool {
		out := make(map[string]bool, len(services))
		for _, svc := range services {
			out[svc.name] = true
		}
		return out
	}

	t.Run("runtime services are served without the control plane", func(t *testing.T) {
		withControlPlaneConfig(t, false, config.InstanceRoleSystem)
		served := names(grpcServices(&Application{}))

		for _, runtime := range []string{
			authv1.AuthorizationService_ServiceDesc.ServiceName,
			authv1.OAuthIntrospectionService_ServiceDesc.ServiceName,
			authv1.UserService_ServiceDesc.ServiceName,
			authv1.UserProfileService_ServiceDesc.ServiceName,
		} {
			assert.True(t, served[runtime], "%s is a data-plane call, not orchestration", runtime)
		}
	})

	// Not merely refused by the interceptor — NOT REGISTERED. A registered service
	// still shows up in reflection and the health surface, telling a caller this
	// instance has an orchestrator API it does not have.
	t.Run("administrative services are absent without the control plane", func(t *testing.T) {
		withControlPlaneConfig(t, false, config.InstanceRoleSystem)
		served := names(grpcServices(&Application{}))

		for _, admin := range []string{
			authv1.TenantService_ServiceDesc.ServiceName,
			authv1.ClientService_ServiceDesc.ServiceName,
			authv1.PolicyService_ServiceDesc.ServiceName,
			authv1.SetupService_ServiceDesc.ServiceName,
			authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName,
		} {
			assert.False(t, served[admin], "%s must not be advertised on a runtime-only instance", admin)
		}
	})

	t.Run("the control plane serves everything", func(t *testing.T) {
		withControlPlaneConfig(t, true, config.InstanceRoleSystem)
		assert.Len(t, grpcServices(&Application{}), len(allGRPCServices(&Application{})))
	})

	// UserService and UserProfileService are MIXED: a peer service needs their
	// reads, but they also create, delete and re-role users. gRPC registers whole
	// services, so the write methods are refused rather than withheld.
	t.Run("write methods on mixed services are refused without the control plane", func(t *testing.T) {
		withControlPlaneConfig(t, false, config.InstanceRoleSystem)

		for _, method := range []string{
			grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "CreateUser"),
			grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "DeleteUser"),
			grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "AssignUserRoles"),
			grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "ForceUserPasswordChange"),
			grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "DeleteUserProfile"),
		} {
			_, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(10, time.Minute), method)
			require.Error(t, err, method)
			assert.Equal(t, codes.FailedPrecondition, status.Code(err), method)
		}
	})

	// The reads must still work, or the split has taken away the thing it exists
	// to provide.
	t.Run("reads on mixed services stay available", func(t *testing.T) {
		withControlPlaneConfig(t, false, config.InstanceRoleSystem)

		for _, method := range []string{
			grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "GetUser"),
			grpcMethod(authv1.UserService_ServiceDesc.ServiceName, "ListUsers"),
			grpcMethod(authv1.UserProfileService_ServiceDesc.ServiceName, "GetUserProfile"),
		} {
			_, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(10, time.Minute), method)
			// Unauthenticated (no token) is the expected stop — NOT
			// FailedPrecondition, which would mean the method was withheld.
			assert.NotEqual(t, codes.FailedPrecondition, status.Code(err), method)
		}
	})

	// Every declared service must be classified one way or the other, so a newly
	// added one cannot land on the runtime surface by omission.
	t.Run("every service is classified", func(t *testing.T) {
		all := allGRPCServices(&Application{})
		require.NotEmpty(t, all)

		withControlPlaneConfig(t, false, config.InstanceRoleSystem)
		runtime := len(grpcServices(&Application{}))
		assert.Equal(t, len(all)-len(grpcAdministrativeServices), runtime,
			"a service added to neither group silently becomes runtime-visible")
	})
}
