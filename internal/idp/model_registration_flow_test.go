package idp

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationFlow_TableName(t *testing.T) {
	assert.Equal(t, "registration_flows", RegistrationFlow{}.TableName())
}

func TestRegistrationFlow_BeforeCreate(t *testing.T) {
	t.Run("assigns uuid and default status when empty", func(t *testing.T) {
		flow := &RegistrationFlow{}
		require.NoError(t, flow.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, flow.RegistrationFlowUUID)
		assert.Equal(t, shared.StatusActive, flow.Status)
	})

	t.Run("keeps existing uuid and status", func(t *testing.T) {
		existing := uuid.New()
		flow := &RegistrationFlow{
			RegistrationFlowUUID: existing,
			Status:               shared.StatusInactive,
		}
		require.NoError(t, flow.BeforeCreate(nil))
		assert.Equal(t, existing, flow.RegistrationFlowUUID)
		assert.Equal(t, shared.StatusInactive, flow.Status)
	})
}

func TestRegistrationFlow_BeforeCreate_LeavesTheRestAlone(t *testing.T) {
	// BeforeCreate only fills the UUID and the default status. In particular it
	// must not invent an identifier: that is derived by the service from the name
	flow := &RegistrationFlow{
		TenantID:             7,
		ClientID:             3,
		Name:                 "partner-signup",
		RequiredFields:       []byte(`["email"]`),
		VerificationRequired: true,
	}
	require.NoError(t, flow.BeforeCreate(nil))
	assert.Equal(t, int64(7), flow.TenantID)
	assert.Equal(t, int64(3), flow.ClientID)
	assert.JSONEq(t, `["email"]`, string(flow.RequiredFields))
	assert.True(t, flow.VerificationRequired)
	// A flow is never system-managed unless the seeder says so explicitly.
	assert.False(t, flow.IsSystem)
	assert.Nil(t, flow.CreatedBy)
	assert.Nil(t, flow.UpdatedBy)
}

// The model is persistence-only: the HTTP contract is served by the response
// DTOs, so a stray json tag here would let a schema change leak into the API.
func TestRegistrationFlow_HasNoJSONTags(t *testing.T) {
	typ := reflect.TypeOf(RegistrationFlow{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		assert.Empty(t, field.Tag.Get("json"), "field %s must not carry a json tag", field.Name)
	}
}

// Both relations are needed by the service: Client for the detail projection,
// Tenant for the actor's tenant-access check.
func TestRegistrationFlow_Relations(t *testing.T) {
	typ := reflect.TypeOf(RegistrationFlow{})

	client, ok := typ.FieldByName("Client")
	require.True(t, ok)
	assert.Equal(t, reflect.TypeOf(&Client{}), client.Type)
	assert.Contains(t, client.Tag.Get("gorm"), "foreignKey:ClientID")

	tenant, ok := typ.FieldByName("Tenant")
	require.True(t, ok)
	assert.Equal(t, reflect.TypeOf(&Tenant{}), tenant.Type)
	assert.Contains(t, tenant.Tag.Get("gorm"), "foreignKey:TenantID")
}
