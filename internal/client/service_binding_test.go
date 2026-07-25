package client

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Binding a client to a service is what makes its tokens carry the `svc` claim, and
// `svc` is the principal the policy bundle and the gRPC authorizer resolve. It is a
// privilege grant, so the client type is a security rule, not a preference.
func TestResolveServiceBinding_OnlyM2MClientsMayActAsAService(t *testing.T) {
	serviceUUID := uuid.NewString()

	for _, clientType := range []string{
		shared.ClientTypeSPA,         // public: ships its credential in readable code
		shared.ClientTypeMobile,      // public: same
		shared.ClientTypeTraditional, // a user-facing app, not a machine principal
	} {
		_, err := resolveServiceBinding(nil, 1, clientType, &serviceUUID)
		require.Error(t, err, clientType)
		assert.Contains(t, err.Error(), "only an m2m client")
	}
}

func TestResolveServiceBinding_NoBindingRequested(t *testing.T) {
	// nil means "unchanged"; an empty string means "unbind". Neither touches the DB,
	// so a nil tx is safe here and proves no lookup happens.
	got, err := resolveServiceBinding(nil, 1, shared.ClientTypeSPA, nil)
	require.NoError(t, err)
	assert.Nil(t, got)

	empty := ""
	got, err = resolveServiceBinding(nil, 1, shared.ClientTypeM2M, &empty)
	require.NoError(t, err)
	assert.Nil(t, got, "an empty service_id unbinds rather than erroring")
}

func TestResolveServiceBinding_RejectsAMalformedUUID(t *testing.T) {
	bad := "not-a-uuid"
	_, err := resolveServiceBinding(nil, 1, shared.ClientTypeM2M, &bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid UUID")
}

func TestServiceUUIDForClient(t *testing.T) {
	assert.Nil(t, serviceUUIDForClient(nil))
	assert.Nil(t, serviceUUIDForClient(&Client{}), "an unbound client reports no service")

	id := int64(7)
	svcUUID := uuid.New()
	got := serviceUUIDForClient(&Client{
		ServiceID: &id,
		Service:   &boundService{ServiceID: id, ServiceUUID: svcUUID},
	})
	require.NotNil(t, got)
	assert.Equal(t, svcUUID.String(), *got)

	// Bound but not preloaded: report absent rather than leaking the internal id.
	assert.Nil(t, serviceUUIDForClient(&Client{ServiceID: &id}))
}
