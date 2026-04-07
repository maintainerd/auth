package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

func TestProfileRequestDto_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "John"}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid full", func(t *testing.T) {
		d := ProfileRequestDTO{
			FirstName:  "John",
			LastName:   strPtr("Doe"),
			Birthdate:  strPtr("1990-01-25"),
			Gender:     strPtr(model.GenderMale),
			Email:      strPtr("john@example.com"),
			Country:    strPtr("US"),
			ProfileURL: strPtr("https://cdn.example.com/avatar.png"),
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing first_name", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: ""}
		require.Error(t, d.Validate())
	})

	t.Run("invalid birthdate format", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "John", Birthdate: strPtr("25-01-1990")}
		require.Error(t, d.Validate())
	})

	t.Run("invalid gender", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "John", Gender: strPtr("unknown")}
		require.Error(t, d.Validate())
	})

	t.Run("invalid email", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "John", Email: strPtr("not-an-email")}
		require.Error(t, d.Validate())
	})

	t.Run("country not 2 chars", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "John", Country: strPtr("USA")}
		require.Error(t, d.Validate())
	})

	t.Run("invalid profile url", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "John", ProfileURL: strPtr("not-a-url")}
		require.Error(t, d.Validate())
	})

	t.Run("valid female gender", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "Jane", Gender: strPtr(model.GenderFemale)}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid prefer_not_to_say gender", func(t *testing.T) {
		d := ProfileRequestDTO{FirstName: "Alex", Gender: strPtr(model.GenderPreferNotToSay)}
		assert.NoError(t, d.Validate())
	})
}

func TestProfileFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := ProfileFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("missing page fails", func(t *testing.T) {
		f := ProfileFilterDTO{PaginationRequestDTO: PaginationRequestDTO{Limit: 10}}
		require.Error(t, f.Validate())
	})
}

func TestNewProfileResponseDTO(t *testing.T) {
	t.Run("empty metadata returns empty map", func(t *testing.T) {
		p := &model.Profile{ProfileUUID: uuid.New(), Metadata: datatypes.JSON(nil)}
		dto := NewProfileResponseDTO(p)
		assert.NotNil(t, dto)
		assert.Empty(t, dto.Metadata)
	})

	t.Run("valid metadata is converted", func(t *testing.T) {
		p := &model.Profile{ProfileUUID: uuid.New(), Metadata: datatypes.JSON(`{"key":"value"}`)}
		dto := NewProfileResponseDTO(p)
		assert.Equal(t, "value", dto.Metadata["key"])
	})

	t.Run("invalid metadata JSON returns empty map", func(t *testing.T) {
		p := &model.Profile{ProfileUUID: uuid.New(), Metadata: datatypes.JSON([]byte("not-json"))}
		dto := NewProfileResponseDTO(p)
		assert.Empty(t, dto.Metadata)
	})
}
