package idp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func validIDPCreate() IdentityProviderCreateRequestDTO {
	return IdentityProviderCreateRequestDTO{
		Name:         "my-idp",
		DisplayName:  "My Identity Provider",
		Provider:     shared.IDPProviderGoogle,
		ProviderType: shared.IDPTypeIdentity,
		Config:       datatypes.JSON(`{}`),
		Status:       shared.StatusActive,
		TenantUUID:   uuid.New().String(),
	}
}

func TestIdentityProviderCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validIDPCreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := validIDPCreate()
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("display_name too short", func(t *testing.T) {
		d := validIDPCreate()
		d.DisplayName = "short"
		require.Error(t, d.Validate())
	})

	t.Run("invalid provider", func(t *testing.T) {
		d := validIDPCreate()
		d.Provider = "yahoo"
		require.Error(t, d.Validate())
	})

	t.Run("invalid provider_type", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = "enterprise"
		require.Error(t, d.Validate())
	})

	t.Run("missing config", func(t *testing.T) {
		d := validIDPCreate()
		d.Config = nil
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validIDPCreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("invalid tenant_uuid", func(t *testing.T) {
		d := validIDPCreate()
		d.TenantUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})
}

func TestIdentityProviderUpdateRequestDto_Validate(t *testing.T) {
	d := IdentityProviderUpdateRequestDTO{
		Name:         "my-idp",
		DisplayName:  "My Identity Provider",
		Provider:     shared.IDPProviderInternal,
		ProviderType: shared.IDPTypeSocial,
		Config:       datatypes.JSON(`{}`),
		Status:       shared.StatusInactive,
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestIdentityProviderStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, IdentityProviderStatusUpdateDTO{Status: shared.StatusActive}.Validate())
	require.Error(t, IdentityProviderStatusUpdateDTO{Status: "bad"}.Validate())
	require.Error(t, IdentityProviderStatusUpdateDTO{Status: ""}.Validate())
}

func TestIdentityProviderFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := IdentityProviderFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid provider in list", func(t *testing.T) {
		f := IdentityProviderFilterDTO{
			PaginationRequestDTO: validPagination(),
			Provider:             []string{"yahoo"},
		}
		require.Error(t, f.Validate())
	})

	t.Run("invalid provider_type", func(t *testing.T) {
		pt := "enterprise"
		f := IdentityProviderFilterDTO{PaginationRequestDTO: validPagination(), ProviderType: &pt}
		require.Error(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := IdentityProviderFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})
}
