package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func TestSignupFlowCreateRequestDto_Validate(t *testing.T) {
	valid := SignupFlowCreateRequestDto{
		Name:        "default-flow",
		Description: "The default signup flow",
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
		s := model.StatusActive
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
}

func TestSignupFlowUpdateRequestDto_Validate(t *testing.T) {
	d := SignupFlowUpdateRequestDto{
		Name:        "updated-flow",
		Description: "Updated description",
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestSignupFlowUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, SignupFlowUpdateStatusRequestDto{Status: model.StatusActive}.Validate())
	assert.NoError(t, SignupFlowUpdateStatusRequestDto{Status: model.StatusInactive}.Validate())
	require.Error(t, SignupFlowUpdateStatusRequestDto{Status: ""}.Validate())
	require.Error(t, SignupFlowUpdateStatusRequestDto{Status: "bad"}.Validate())
}

func TestSignupFlowFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := SignupFlowFilterDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := SignupFlowFilterDto{PaginationRequestDto: validPagination(), Status: []string{"bad"}}
		require.Error(t, f.Validate())
	})

	t.Run("invalid client_uuid", func(t *testing.T) {
		s := "not-a-uuid"
		f := SignupFlowFilterDto{PaginationRequestDto: validPagination(), ClientUUID: &s}
		require.Error(t, f.Validate())
	})
}

// ------ SignupFlowRoleDto tests ------

func TestSignupFlowAssignRolesRequestDto_Validate(t *testing.T) {
	t.Run("valid single role", func(t *testing.T) {
		d := SignupFlowAssignRolesRequestDto{RoleUUIDs: []string{uuid.New().String()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing role_uuids", func(t *testing.T) {
		d := SignupFlowAssignRolesRequestDto{}
		require.Error(t, d.Validate())
	})

	t.Run("empty role_uuids", func(t *testing.T) {
		d := SignupFlowAssignRolesRequestDto{RoleUUIDs: []string{}}
		require.Error(t, d.Validate())
	})

	t.Run("invalid uuid in list", func(t *testing.T) {
		d := SignupFlowAssignRolesRequestDto{RoleUUIDs: []string{"not-a-uuid"}}
		require.Error(t, d.Validate())
	})
}

