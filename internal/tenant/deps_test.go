package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type accessActorStub struct {
	identities []AccessIdentity
}

func (a accessActorStub) AccessIdentities() []AccessIdentity {
	return a.identities
}

func TestAccessActorContract(t *testing.T) {
	var actor AccessActor = accessActorStub{
		identities: []AccessIdentity{{TenantID: 10, TenantIsSystem: true}},
	}

	identities := actor.AccessIdentities()

	assert.Len(t, identities, 1)
	assert.Equal(t, int64(10), identities[0].TenantID)
	assert.True(t, identities[0].TenantIsSystem)
}
