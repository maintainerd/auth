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
		d := APIKeyCreateRequestDto{Name: "my-key"}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := APIKeyCreateRequestDto{}
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := APIKeyCreateRequestDto{Name: string(make([]byte, 101))}
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		status := "unknown"
		d := APIKeyCreateRequestDto{Name: "key", Status: status}
		require.Error(t, d.Validate())
	})

	t.Run("rate_limit negative", func(t *testing.T) {
		// ozzo-validation treats 0 as "empty" for int, so use a negative value to trigger Min(1)
		d := APIKeyCreateRequestDto{Name: "key", RateLimit: intPtr(-1)}
		require.Error(t, d.Validate())
	})
}

func TestAPIKeyUpdateRequestDto_Validate(t *testing.T) {
	t.Run("valid empty update", func(t *testing.T) {
		d := APIKeyUpdateRequestDto{}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		s := "bad"
		d := APIKeyUpdateRequestDto{Status: &s}
		require.Error(t, d.Validate())
	})

	t.Run("valid active status", func(t *testing.T) {
		s := model.StatusActive
		d := APIKeyUpdateRequestDto{Status: &s}
		assert.NoError(t, d.Validate())
	})
}

func TestAPIKeyGetRequestDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		d := APIKeyGetRequestDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing page", func(t *testing.T) {
		d := APIKeyGetRequestDto{PaginationRequestDto: PaginationRequestDto{Limit: 10}}
		require.Error(t, d.Validate())
	})

	t.Run("invalid status filter", func(t *testing.T) {
		s := "unknown"
		d := APIKeyGetRequestDto{PaginationRequestDto: validPagination(), Status: &s}
		require.Error(t, d.Validate())
	})
}

func TestAPIKeyStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, APIKeyStatusUpdateDto{Status: model.StatusActive}.Validate())
	assert.NoError(t, APIKeyStatusUpdateDto{Status: model.StatusInactive}.Validate())
	require.Error(t, APIKeyStatusUpdateDto{Status: ""}.Validate())
	require.Error(t, APIKeyStatusUpdateDto{Status: "bad"}.Validate())
}

func TestAddAPIKeyApisRequestDto_Validate(t *testing.T) {
	t.Run("valid uuids", func(t *testing.T) {
		d := AddAPIKeyApisRequestDto{APIUUIDs: []uuid.UUID{uuid.New()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing uuids", func(t *testing.T) {
		d := AddAPIKeyApisRequestDto{}
		require.Error(t, d.Validate())
	})
}

func TestAddAPIKeyPermissionsRequestDto_Validate(t *testing.T) {
	t.Run("valid uuids", func(t *testing.T) {
		d := AddAPIKeyPermissionsRequestDto{PermissionUUIDs: []uuid.UUID{uuid.New()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing uuids", func(t *testing.T) {
		d := AddAPIKeyPermissionsRequestDto{}
		require.Error(t, d.Validate())
	})
}
