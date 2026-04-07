package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func validServiceCreate() ServiceCreateOrUpdateRequestDTO {
	return ServiceCreateOrUpdateRequestDTO{
		Name:        "auth-svc",
		DisplayName: "Auth Service",
		Description: "Handles authentication",
		Version:     "v1.0.0",
		Status:      model.StatusActive,
	}
}

func TestServiceCreateOrUpdateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validServiceCreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validServiceCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := validServiceCreate()
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("missing display_name", func(t *testing.T) {
		d := validServiceCreate()
		d.DisplayName = ""
		require.Error(t, d.Validate())
	})

	t.Run("display_name too short", func(t *testing.T) {
		d := validServiceCreate()
		d.DisplayName = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("description too short", func(t *testing.T) {
		d := validServiceCreate()
		d.Description = "short"
		require.Error(t, d.Validate())
	})

	t.Run("missing version", func(t *testing.T) {
		d := validServiceCreate()
		d.Version = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validServiceCreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("maintenance status valid", func(t *testing.T) {
		d := validServiceCreate()
		d.Status = model.StatusMaintenance
		assert.NoError(t, d.Validate())
	})

	t.Run("deprecated status valid", func(t *testing.T) {
		d := validServiceCreate()
		d.Status = model.StatusDeprecated
		assert.NoError(t, d.Validate())
	})
}

func TestServiceFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := ServiceFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := ServiceFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})

	t.Run("valid status list", func(t *testing.T) {
		f := ServiceFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{model.StatusActive, model.StatusMaintenance},
		}
		assert.NoError(t, f.Validate())
	})
}

func TestServiceStatusUpdateRequestDto_Validate(t *testing.T) {
	assert.NoError(t, ServiceStatusUpdateRequestDTO{Status: model.StatusActive}.Validate())
	assert.NoError(t, ServiceStatusUpdateRequestDTO{Status: model.StatusDeprecated}.Validate())
	require.Error(t, ServiceStatusUpdateRequestDTO{Status: ""}.Validate())
	require.Error(t, ServiceStatusUpdateRequestDTO{Status: "bad"}.Validate())
}

