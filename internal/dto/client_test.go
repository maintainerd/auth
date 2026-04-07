package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

func validClientCreate() ClientCreateRequestDTO {
	return ClientCreateRequestDTO{
		Name:                 "my-client",
		DisplayName:          "My Auth Client",
		ClientType:           model.ClientTypeSPA,
		Domain:               "example.com",
		Config:               datatypes.JSON(`{}`),
		Status:               model.StatusActive,
		IdentityProviderUUID: uuid.New().String(),
	}
}

func TestClientCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validClientCreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validClientCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := validClientCreate()
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("display_name too short", func(t *testing.T) {
		d := validClientCreate()
		d.DisplayName = "short"
		require.Error(t, d.Validate())
	})

	t.Run("invalid client_type", func(t *testing.T) {
		d := validClientCreate()
		d.ClientType = "desktop"
		require.Error(t, d.Validate())
	})

	t.Run("missing domain", func(t *testing.T) {
		d := validClientCreate()
		d.Domain = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing config", func(t *testing.T) {
		d := validClientCreate()
		d.Config = nil
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validClientCreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("invalid idp uuid", func(t *testing.T) {
		d := validClientCreate()
		d.IdentityProviderUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})
}

func TestClientUpdateRequestDto_Validate(t *testing.T) {
	d := ClientUpdateRequestDTO{
		Name:        "my-client",
		DisplayName: "My Auth Client",
		ClientType:  model.ClientTypeMobile,
		Domain:      "example.com",
		Config:      datatypes.JSON(`{}`),
		Status:      model.StatusInactive,
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestClientURICreateOrUpdateRequestDto_Validate(t *testing.T) {
	d := ClientURICreateOrUpdateRequestDTO{URI: "https://app.example.com/callback", Type: model.ClientURITypeRedirect}
	assert.NoError(t, d.Validate())

	d.URI = ""
	require.Error(t, d.Validate())

	d.URI = "https://app.example.com/callback"
	d.Type = "bad-type"
	require.Error(t, d.Validate())
}

func TestClientFilterDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		f := ClientFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid client_type", func(t *testing.T) {
		f := ClientFilterDTO{PaginationRequestDTO: validPagination(), ClientType: []string{"desktop"}}
		require.Error(t, f.Validate())
	})

	t.Run("invalid identity_provider_uuid", func(t *testing.T) {
		s := "not-a-uuid"
		f := ClientFilterDTO{PaginationRequestDTO: validPagination(), IdentityProviderUUID: &s}
		require.Error(t, f.Validate())
	})
}

func TestClientAddPermissionsRequestDto_Validate(t *testing.T) {
	assert.NoError(t, ClientAddPermissionsRequestDTO{Permissions: []uuid.UUID{uuid.New()}}.Validate())
	require.Error(t, ClientAddPermissionsRequestDTO{}.Validate())
}

func TestAddClientAPIsRequestDto_Validate(t *testing.T) {
	assert.NoError(t, AddClientAPIsRequestDTO{APIUUIDs: []uuid.UUID{uuid.New()}}.Validate())
	require.Error(t, AddClientAPIsRequestDTO{}.Validate())
}

func TestAddClientAPIPermissionsRequestDto_Validate(t *testing.T) {
	assert.NoError(t, AddClientAPIPermissionsRequestDTO{PermissionUUIDs: []uuid.UUID{uuid.New()}}.Validate())
	require.Error(t, AddClientAPIPermissionsRequestDTO{}.Validate())
}

