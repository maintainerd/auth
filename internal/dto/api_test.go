package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func validAPICreate() APICreateRequestDTO {
	return APICreateRequestDTO{
		Name:        "my-api",
		DisplayName: "My API",
		Description: "A test API description",
		APIType:     model.APITypeRest,
		Status:      model.StatusActive,
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

	t.Run("description too short", func(t *testing.T) {
		d := validAPICreate()
		d.Description = "short"
		require.Error(t, d.Validate())
	})

	t.Run("invalid api_type", func(t *testing.T) {
		d := validAPICreate()
		d.APIType = "ftp"
		require.Error(t, d.Validate())
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
		APIType:     model.APITypeGRPC,
		Status:      model.StatusInactive,
		ServiceUUID: uuid.New().String(),
	}
	assert.NoError(t, d.Validate())

	d.APIType = "bad"
	require.Error(t, d.Validate())
}

func TestAPIFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := APIFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid api_type filter", func(t *testing.T) {
		apiType := "bad"
		f := APIFilterDTO{PaginationRequestDTO: validPagination(), APIType: &apiType}
		require.Error(t, f.Validate())
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
	assert.NoError(t, APIStatusUpdateDTO{Status: model.StatusActive}.Validate())
	assert.NoError(t, APIStatusUpdateDTO{Status: model.StatusInactive}.Validate())
	require.Error(t, APIStatusUpdateDTO{Status: "unknown"}.Validate())
	require.Error(t, APIStatusUpdateDTO{Status: ""}.Validate())
}

