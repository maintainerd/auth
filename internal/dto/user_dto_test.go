package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func validUserCreate() UserCreateRequestDto {
	return UserCreateRequestDto{
		Username:   "testuser",
		Fullname:   "Test User",
		Password:   "SecurePass1!",
		Status:     model.StatusActive,
		TenantUUID: uuid.New().String(),
	}
}

func TestUserCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		assert.NoError(t, validUserCreate().Validate())
	})

	t.Run("username too short", func(t *testing.T) {
		d := validUserCreate()
		d.Username = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("missing fullname", func(t *testing.T) {
		d := validUserCreate()
		d.Fullname = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid email", func(t *testing.T) {
		d := validUserCreate()
		d.Email = strPtr("not-an-email")
		require.Error(t, d.Validate())
	})

	t.Run("valid email", func(t *testing.T) {
		d := validUserCreate()
		d.Email = strPtr("user@example.com")
		assert.NoError(t, d.Validate())
	})

	t.Run("phone too short", func(t *testing.T) {
		d := validUserCreate()
		d.Phone = strPtr("123")
		require.Error(t, d.Validate())
	})

	t.Run("password too short", func(t *testing.T) {
		d := validUserCreate()
		d.Password = "short"
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validUserCreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("pending status valid", func(t *testing.T) {
		d := validUserCreate()
		d.Status = model.StatusPending
		assert.NoError(t, d.Validate())
	})

	t.Run("missing tenant_id", func(t *testing.T) {
		d := validUserCreate()
		d.TenantUUID = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid tenant_id uuid", func(t *testing.T) {
		d := validUserCreate()
		d.TenantUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})
}

func TestUserUpdateRequestDto_Validate(t *testing.T) {
	d := UserUpdateRequestDto{
		Username: "testuser",
		Fullname: "Test User",
		Status:   model.StatusActive,
	}
	assert.NoError(t, d.Validate())

	d.Username = ""
	require.Error(t, d.Validate())
}

func TestUserSetStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, UserSetStatusRequestDto{Status: model.StatusActive}.Validate())
	assert.NoError(t, UserSetStatusRequestDto{Status: model.StatusSuspended}.Validate())
	require.Error(t, UserSetStatusRequestDto{Status: ""}.Validate())
	require.Error(t, UserSetStatusRequestDto{Status: "unknown"}.Validate())
}

func TestUserAssignRolesRequestDto_Validate(t *testing.T) {
	t.Run("valid single role", func(t *testing.T) {
		d := UserAssignRolesRequestDto{RoleUUIDs: []uuid.UUID{uuid.New()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing roles", func(t *testing.T) {
		d := UserAssignRolesRequestDto{}
		require.Error(t, d.Validate())
	})
}

func TestUserFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := UserFilterDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		f := UserFilterDto{PaginationRequestDto: validPagination(), Status: []string{"unknown"}}
		require.Error(t, f.Validate())
	})

	t.Run("invalid tenant uuid", func(t *testing.T) {
		s := "not-a-uuid"
		f := UserFilterDto{PaginationRequestDto: validPagination(), TenantUUID: &s}
		require.Error(t, f.Validate())
	})

	t.Run("invalid role uuid", func(t *testing.T) {
		s := "not-a-uuid"
		f := UserFilterDto{PaginationRequestDto: validPagination(), RoleUUID: &s}
		require.Error(t, f.Validate())
	})
}

func TestUserRoleFilterDto_Validate(t *testing.T) {
	f := UserRoleFilterDto{PaginationRequestDto: validPagination()}
	assert.NoError(t, f.Validate())
}

func TestUserIdentityFilterDto_Validate(t *testing.T) {
	f := UserIdentityFilterDto{PaginationRequestDto: validPagination()}
	assert.NoError(t, f.Validate())
}

