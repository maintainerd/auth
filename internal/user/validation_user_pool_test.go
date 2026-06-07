package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserPoolCreateRequestDTO_Validate(t *testing.T) {
	valid := UserPoolCreateRequestDTO{Name: "Customers", DisplayName: "Customer Pool", Status: "active"}

	t.Run("valid (with status)", func(t *testing.T) {
		require.NoError(t, valid.Validate())
	})

	t.Run("valid (status omitted)", func(t *testing.T) {
		dto := valid
		dto.Status = ""
		require.NoError(t, dto.Validate())
	})

	t.Run("name required", func(t *testing.T) {
		dto := valid
		dto.Name = ""
		err := dto.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Name")
	})

	t.Run("name too short", func(t *testing.T) {
		dto := valid
		dto.Name = "a"
		err := dto.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "2-100")
	})

	t.Run("name too long", func(t *testing.T) {
		dto := valid
		dto.Name = strings.Repeat("a", 101)
		require.Error(t, dto.Validate())
	})

	t.Run("display name too long", func(t *testing.T) {
		dto := valid
		dto.DisplayName = strings.Repeat("a", 151)
		require.Error(t, dto.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		dto := valid
		dto.Status = "bogus"
		err := dto.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Status")
	})
}

func TestUserPoolSetStatusRequestDTO_Validate(t *testing.T) {
	t.Run("valid active", func(t *testing.T) {
		require.NoError(t, UserPoolSetStatusRequestDTO{Status: "active"}.Validate())
	})

	t.Run("valid inactive", func(t *testing.T) {
		require.NoError(t, UserPoolSetStatusRequestDTO{Status: "inactive"}.Validate())
	})

	t.Run("status required", func(t *testing.T) {
		err := UserPoolSetStatusRequestDTO{Status: ""}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Status")
	})

	t.Run("invalid status", func(t *testing.T) {
		err := UserPoolSetStatusRequestDTO{Status: "suspended"}.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Status")
	})
}

func TestUserPoolUpdateRequestDTO_Validate(t *testing.T) {
	valid := UserPoolUpdateRequestDTO{Name: "Customers", DisplayName: "Customer Pool", Status: "active"}

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, valid.Validate())
	})

	t.Run("name required", func(t *testing.T) {
		dto := valid
		dto.Name = ""
		require.Error(t, dto.Validate())
	})

	t.Run("status required", func(t *testing.T) {
		dto := valid
		dto.Status = ""
		err := dto.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Status")
	})

	t.Run("invalid status", func(t *testing.T) {
		dto := valid
		dto.Status = "deleted"
		require.Error(t, dto.Validate())
	})
}
