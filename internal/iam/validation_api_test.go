package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validAPICreate() APICreateRequestDTO {
	return APICreateRequestDTO{
		Name:        "my-api",
		DisplayName: "My API",
		Description: "A test API description",
		Status:      shared.StatusActive,
		ServiceUUID: uuid.New().String(),
	}
}

func TestAPICreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validAPICreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validAPICreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := validAPICreate()
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("missing display_name", func(t *testing.T) {
		d := validAPICreate()
		d.DisplayName = ""
		require.Error(t, d.Validate())
	})

	t.Run("description too short is now valid", func(t *testing.T) {
		d := validAPICreate()
		d.Description = "short"
		require.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validAPICreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("invalid service uuid", func(t *testing.T) {
		d := validAPICreate()
		d.ServiceUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})
}

func TestAPIUpdateRequestDto_Validate(t *testing.T) {
	d := APIUpdateRequestDTO{
		Name:        "my-api",
		DisplayName: "My API",
		Description: "A valid description",
		Status:      shared.StatusInactive,
		ServiceUUID: uuid.New().String(),
	}
	assert.NoError(t, d.Validate())
}

func TestAPIFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := APIFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := APIFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{"bad-status"},
		}
		require.Error(t, f.Validate())
	})
}

func TestAPIStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, APIStatusUpdateDTO{Status: shared.StatusActive}.Validate())
	assert.NoError(t, APIStatusUpdateDTO{Status: shared.StatusInactive}.Validate())
	require.Error(t, APIStatusUpdateDTO{Status: "unknown"}.Validate())
	require.Error(t, APIStatusUpdateDTO{Status: ""}.Validate())
}
