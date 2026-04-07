package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func validTenantCreate() TenantCreateRequestDTO {
	return TenantCreateRequestDTO{
		Name:        "my-tenant",
		Description: "A test tenant description",
		Status:      model.StatusActive,
	}
}

func TestTenantCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validTenantCreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validTenantCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := validTenantCreate()
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("name with uppercase invalid", func(t *testing.T) {
		d := validTenantCreate()
		d.Name = "MyTenant"
		require.Error(t, d.Validate())
	})

	t.Run("name with underscore invalid", func(t *testing.T) {
		d := validTenantCreate()
		d.Name = "my_tenant"
		require.Error(t, d.Validate())
	})

	t.Run("valid name with hyphens and numbers", func(t *testing.T) {
		d := validTenantCreate()
		d.Name = "my-tenant-2"
		assert.NoError(t, d.Validate())
	})

	t.Run("description too short", func(t *testing.T) {
		d := validTenantCreate()
		d.Description = "short"
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validTenantCreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("suspended status valid", func(t *testing.T) {
		d := validTenantCreate()
		d.Status = model.StatusSuspended
		assert.NoError(t, d.Validate())
	})
}

func TestTenantUpdateRequestDto_Validate(t *testing.T) {
	d := TenantUpdateRequestDTO{
		Name:        "updated-tenant",
		Description: "An updated tenant description",
		Status:      model.StatusInactive,
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestTenantFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := TenantFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})
}

// ------ TenantMember tests ------

func TestTenantMemberAddMemberRequestDto_Validate(t *testing.T) {
	t.Run("valid owner", func(t *testing.T) {
		d := TenantMemberAddMemberRequestDTO{UserUUID: uuid.New(), Role: "owner"}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid member", func(t *testing.T) {
		d := TenantMemberAddMemberRequestDTO{UserUUID: uuid.New(), Role: "member"}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid role", func(t *testing.T) {
		d := TenantMemberAddMemberRequestDTO{UserUUID: uuid.New(), Role: "admin"}
		require.Error(t, d.Validate())
	})
}

func TestTenantMemberUpdateRoleRequestDto_Validate(t *testing.T) {
	assert.NoError(t, TenantMemberUpdateRoleRequestDTO{Role: "owner"}.Validate())
	assert.NoError(t, TenantMemberUpdateRoleRequestDTO{Role: "member"}.Validate())
	require.Error(t, TenantMemberUpdateRoleRequestDTO{Role: "admin"}.Validate())
	require.Error(t, TenantMemberUpdateRoleRequestDTO{Role: ""}.Validate())
}

func TestTenantMemberFilterDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		f := TenantMemberFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid role filter", func(t *testing.T) {
		bad := "admin"
		f := TenantMemberFilterDTO{PaginationRequestDTO: validPagination(), Role: &bad}
		require.Error(t, f.Validate())
	})
}
