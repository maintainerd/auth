package server

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	_ "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
)

const authProtoPackage = "maintainerd.auth.v1"

// grpcContractServiceNames returns every service the shipped maintainerd.auth.v1
// proto contract declares, read from the protobuf registry the generated code
// populates at init.
func grpcContractServiceNames(t *testing.T) []string {
	t.Helper()
	var names []string
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if string(fd.Package()) != authProtoPackage {
			return true
		}
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			names = append(names, string(services.Get(i).FullName()))
		}
		return true
	})
	sort.Strings(names)
	return names
}

func grpcRegisteredServiceNames(t *testing.T) []string {
	t.Helper()
	s := grpc.NewServer()
	defer s.Stop()
	for _, svc := range grpcServices(&Application{}) {
		svc.register(s)
	}
	var names []string
	for name := range s.GetServiceInfo() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// grpcServices must actually bind what it claims: a typo in the name field, or
// a registration silently dropped during an edit, turns a live RPC into
// UNIMPLEMENTED while health still reports the service SERVING.
func TestGRPCServicesRegisterWhatTheyName(t *testing.T) {
	declared := make([]string, 0)
	for _, svc := range grpcServices(&Application{}) {
		declared = append(declared, svc.name)
	}
	sort.Strings(declared)

	assert.Equal(t, declared, grpcRegisteredServiceNames(t))
}

// Every service the contract declares must be served. This assertion is the
// inverse of the one it replaces: that test enumerated twelve declared-but-
// unregistered services (AuthEvent, Branding, EmailConfig, EmailTemplate,
// IPRestrictionRule, IdentityProvider, Invite, RegistrationFlow, SMSConfig,
// SMSTemplate, SecuritySetting, WebhookEndpoint) and asserted they stayed
// missing, which pinned the defect instead of fixing it — a contract advertising
// RPCs the server answers UNIMPLEMENTED is indistinguishable from an outage, and
// the pin would have let a thirteenth be added the same way. Those service
// blocks are gone from the protos (their surfaces are REST control-plane only),
// so declared and registered are now the same set and must stay that way.
func TestGRPCContractIsFullyServed(t *testing.T) {
	contract := grpcContractServiceNames(t)
	require.NotEmpty(t, contract, "the generated auth protos must be linked into this test binary")

	assert.Equal(t, contract, grpcRegisteredServiceNames(t),
		"every maintainerd.auth.v1 service must have a handler registered in grpcServices; declare an RPC only once the server answers it")
}
