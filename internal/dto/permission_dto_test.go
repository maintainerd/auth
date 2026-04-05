package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func TestPermissionCreateRequestDto_Validate(t *testing.T) {
	valid := PermissionCreateRequestDto{
		Name:        "perm:read",
		Description: "Read permission for all users",
		Status:      model.StatusActive,
		APIUUID:     uuid.New().String(),
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

	t.Run("description too short", func(t *testing.T) {
		d := valid
		d.Description = "short"
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := valid
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("missing api_id", func(t *testing.T) {
		d := valid
		d.APIUUID = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid api_id uuid", func(t *testing.T) {
		d := valid
		d.APIUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})
}

func TestPermissionUpdateRequestDto_Validate(t *testing.T) {
	d := PermissionUpdateRequestDto{
		Name:        "perm:write",
		Description: "Write permission for all resources",
		Status:      model.StatusInactive,
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestPermissionStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, PermissionStatusUpdateDto{Status: model.StatusActive}.Validate())
	assert.NoError(t, PermissionStatusUpdateDto{Status: model.StatusInactive}.Validate())
	require.Error(t, PermissionStatusUpdateDto{Status: ""}.Validate())
	require.Error(t, PermissionStatusUpdateDto{Status: "bad"}.Validate())
}

func TestPermissionFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := PermissionFilterDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		bad := "unknown"
		f := PermissionFilterDto{PaginationRequestDto: validPagination(), Status: &bad}
		require.Error(t, f.Validate())
	})

	t.Run("valid status filter", func(t *testing.T) {
		s := model.StatusActive
		f := PermissionFilterDto{PaginationRequestDto: validPagination(), Status: &s}
		assert.NoError(t, f.Validate())
	})
}

