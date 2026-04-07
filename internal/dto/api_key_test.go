package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func TestAPIKeyCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		d := APIKeyCreateRequestDTO{Name: "my-key"}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := APIKeyCreateRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := APIKeyCreateRequestDTO{Name: string(make([]byte, 101))}
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		status := "unknown"
		d := APIKeyCreateRequestDTO{Name: "key", Status: status}
		require.Error(t, d.Validate())
	})

	t.Run("rate_limit negative", func(t *testing.T) {
		// ozzo-validation treats 0 as "empty" for int, so use a negative value to trigger Min(1)
		d := APIKeyCreateRequestDTO{Name: "key", RateLimit: intPtr(-1)}
		require.Error(t, d.Validate())
	})
}

func TestAPIKeyUpdateRequestDto_Validate(t *testing.T) {
	t.Run("valid empty update", func(t *testing.T) {
		d := APIKeyUpdateRequestDTO{}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		s := "bad"
		d := APIKeyUpdateRequestDTO{Status: &s}
		require.Error(t, d.Validate())
	})

	t.Run("valid active status", func(t *testing.T) {
		s := model.StatusActive
		d := APIKeyUpdateRequestDTO{Status: &s}
		assert.NoError(t, d.Validate())
	})
}

func TestAPIKeyGetRequestDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		d := APIKeyGetRequestDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing page", func(t *testing.T) {
		d := APIKeyGetRequestDTO{PaginationRequestDTO: PaginationRequestDTO{Limit: 10}}
		require.Error(t, d.Validate())
	})

	t.Run("invalid status filter", func(t *testing.T) {
		s := "unknown"
		d := APIKeyGetRequestDTO{PaginationRequestDTO: validPagination(), Status: &s}
		require.Error(t, d.Validate())
	})
}

func TestAPIKeyStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, APIKeyStatusUpdateDTO{Status: model.StatusActive}.Validate())
	assert.NoError(t, APIKeyStatusUpdateDTO{Status: model.StatusInactive}.Validate())
	require.Error(t, APIKeyStatusUpdateDTO{Status: ""}.Validate())
	require.Error(t, APIKeyStatusUpdateDTO{Status: "bad"}.Validate())
}

func TestAddAPIKeyAPIsRequestDto_Validate(t *testing.T) {
	t.Run("valid uuids", func(t *testing.T) {
		d := AddAPIKeyAPIsRequestDTO{APIUUIDs: []uuid.UUID{uuid.New()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing uuids", func(t *testing.T) {
		d := AddAPIKeyAPIsRequestDTO{}
		require.Error(t, d.Validate())
	})
}

func TestAddAPIKeyPermissionsRequestDto_Validate(t *testing.T) {
	t.Run("valid uuids", func(t *testing.T) {
		d := AddAPIKeyPermissionsRequestDTO{PermissionUUIDs: []uuid.UUID{uuid.New()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing uuids", func(t *testing.T) {
		d := AddAPIKeyPermissionsRequestDTO{}
		require.Error(t, d.Validate())
	})
}
