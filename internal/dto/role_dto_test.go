package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func TestRoleCreateOrUpdateRequestDto_Validate(t *testing.T) {
	valid := RoleCreateOrUpdateRequestDto{
		Name:        "admin",
		Description: "Administrator role",
		Status:      model.StatusActive,
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := valid
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := valid
		d.Name = string(make([]byte, 21))
		require.Error(t, d.Validate())
	})

	t.Run("description too short", func(t *testing.T) {
		d := valid
		d.Description = "short"
		require.Error(t, d.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		d := valid
		d.Description = string(make([]byte, 101))
		require.Error(t, d.Validate())
	})

	t.Run("missing status", func(t *testing.T) {
		d := valid
		d.Status = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := valid
		d.Status = "pending"
		require.Error(t, d.Validate())
	})

	t.Run("inactive status valid", func(t *testing.T) {
		d := valid
		d.Status = model.StatusInactive
		assert.NoError(t, d.Validate())
	})
}

func TestRoleAddPermissionsRequestDto_Validate(t *testing.T) {
	t.Run("valid single permission", func(t *testing.T) {
		d := RoleAddPermissionsRequestDto{Permissions: []uuid.UUID{uuid.New()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid multiple permissions", func(t *testing.T) {
		d := RoleAddPermissionsRequestDto{Permissions: []uuid.UUID{uuid.New(), uuid.New()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing permissions", func(t *testing.T) {
		d := RoleAddPermissionsRequestDto{}
		require.Error(t, d.Validate())
	})
}

func TestRoleFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := RoleFilterDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status filter", func(t *testing.T) {
		bad := "pending"
		f := RoleFilterDto{PaginationRequestDto: validPagination(), Status: &bad}
		require.Error(t, f.Validate())
	})

	t.Run("valid status filter", func(t *testing.T) {
		s := model.StatusActive
		f := RoleFilterDto{PaginationRequestDto: validPagination(), Status: &s}
		assert.NoError(t, f.Validate())
	})
}

