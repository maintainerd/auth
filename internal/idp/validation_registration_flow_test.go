package idp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationFlowCreateRequestDto_Validate(t *testing.T) {
	valid := RegistrationFlowCreateRequestDTO{
		Name:        "default-flow",
		Description: "The default registration flow",
		ClientUUID:  uuid.New().String(),
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := valid
		d.Name = string(make([]byte, 101))
		require.Error(t, d.Validate())
	})

	t.Run("missing description", func(t *testing.T) {
		d := valid
		d.Description = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		bad := "unknown"
		d := valid
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid active status", func(t *testing.T) {
		s := shared.StatusActive
		d := valid
		d.Status = &s
		assert.NoError(t, d.Validate())
	})

	t.Run("missing client_uuid", func(t *testing.T) {
		d := valid
		d.ClientUUID = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid client_uuid", func(t *testing.T) {
		d := valid
		d.ClientUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})

	t.Run("valid required fields", func(t *testing.T) {
		fields := `["email","fullname","phone"]`
		d := valid
		d.RequiredFields = &fields
		assert.NoError(t, d.Validate())
	})

	t.Run("required fields must be a string array", func(t *testing.T) {
		fields := `{"email":true}`
		d := valid
		d.RequiredFields = &fields
		require.Error(t, d.Validate())
	})

	t.Run("unknown required field is rejected", func(t *testing.T) {
		fields := `["address"]`
		d := valid
		d.RequiredFields = &fields
		require.Error(t, d.Validate())
	})
}

func TestRegistrationFlowUpdateRequestDto_Validate(t *testing.T) {
	d := RegistrationFlowUpdateRequestDTO{
		Name:        "updated-flow",
		Description: "Updated description",
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestRegistrationFlowUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, RegistrationFlowUpdateStatusRequestDTO{Status: shared.StatusActive}.Validate())
	assert.NoError(t, RegistrationFlowUpdateStatusRequestDTO{Status: shared.StatusInactive}.Validate())
	require.Error(t, RegistrationFlowUpdateStatusRequestDTO{Status: ""}.Validate())
	require.Error(t, RegistrationFlowUpdateStatusRequestDTO{Status: "bad"}.Validate())
}

func TestRegistrationFlowFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: validPagination(), Status: []string{"bad"}}
		require.Error(t, f.Validate())
	})

	t.Run("invalid client_uuid", func(t *testing.T) {
		s := "not-a-uuid"
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: validPagination(), ClientUUID: &s}
		require.Error(t, f.Validate())
	})
}

// ------ RegistrationFlowRoleDTO tests ------

func TestRegistrationFlowAssignRolesRequestDto_Validate(t *testing.T) {
	t.Run("valid single role", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: []string{uuid.New().String()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing role_uuids", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("empty role_uuids", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: []string{}}
		require.Error(t, d.Validate())
	})

	t.Run("invalid uuid in list", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: []string{"not-a-uuid"}}
		require.Error(t, d.Validate())
	})
}
