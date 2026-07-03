package idp

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func validIDPCreate() IdentityProviderCreateRequestDTO {
	return IdentityProviderCreateRequestDTO{
		Name:             "my-idp",
		DisplayName:      "My Identity Provider",
		Provider:         shared.IDPProviderGoogle,
		ProviderType:     shared.IDPTypeIdentity,
		Issuer:           "https://accounts.google.com",
		ProviderClientID: "test-client",
		Config:           validOAuth2ConfigJSON(),
		Status:           shared.StatusActive,
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

	t.Run("enterprise provider_type is valid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeEnterprise
		require.NoError(t, d.Validate())
	})

	t.Run("config is optional", func(t *testing.T) {
		d := validIDPCreate()
		d.Config = nil
		require.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validIDPCreate()
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("active social missing client_id is invalid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.ProviderClientID = ""
		require.Error(t, d.Validate())
	})

	t.Run("active social missing issuer is invalid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = ""
		require.Error(t, d.Validate())
	})

	t.Run("active social with non-url issuer is invalid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Issuer = "not-a-url"
		require.Error(t, d.Validate())
	})

	t.Run("enterprise with issuer and explicit endpoints is valid", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeEnterprise
		d.Issuer = "https://idp.example.com"
		d.ProviderClientID = "abc"
		d.Config = datatypes.JSON(`{"authorization_endpoint":"https://idp.example.com/authorize","token_endpoint":"https://idp.example.com/token"}`)
		require.NoError(t, d.Validate())
	})

	t.Run("invalid email domain is rejected", func(t *testing.T) {
		d := validIDPCreate()
		d.EmailDomains = []string{"bad domain!"}
		require.Error(t, d.Validate())
	})

	t.Run("valid email domains pass", func(t *testing.T) {
		d := validIDPCreate()
		d.EmailDomains = []string{"example.com", "sub.example.org"}
		require.NoError(t, d.Validate())
	})

	t.Run("inactive social without creds is allowed (draft)", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSocial
		d.Status = shared.StatusInactive
		d.Issuer = ""
		d.ProviderClientID = ""
		require.NoError(t, d.Validate())
	})

	t.Run("system provider type skips external creds rule", func(t *testing.T) {
		d := validIDPCreate()
		d.ProviderType = shared.IDPTypeSystem
		d.Issuer = ""
		d.ProviderClientID = ""
		d.Config = datatypes.JSON(`{}`)
		require.NoError(t, d.Validate())
	})
}

func TestIdentityProviderUpdateRequestDto_Validate(t *testing.T) {
	d := IdentityProviderUpdateRequestDTO{
		Name:         "my-idp",
		DisplayName:  "My Identity Provider",
		Provider:     shared.IDPProviderMaintainerd,
		ProviderType: shared.IDPTypeSocial,
		Config:       validOAuth2ConfigJSON(),
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

	t.Run("enterprise provider_type is valid", func(t *testing.T) {
		pt := "enterprise"
		f := IdentityProviderFilterDTO{PaginationRequestDTO: validPagination(), ProviderType: &pt}
		require.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := IdentityProviderFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})
}

func TestIdentityProviderCreateValidation_TokenFederation(t *testing.T) {
	t.Run("token federation enabled with issuer and audiences passes", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = []string{"app-1"}
		assert.NoError(t, d.Validate())
	})

	t.Run("token federation enabled requires audiences", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = nil
		require.Error(t, d.Validate())
	})

	t.Run("token federation enabled requires issuer", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = []string{"app-1"}
		d.Issuer = ""
		require.Error(t, d.Validate())
	})

	t.Run("token federation off with no audiences is ok", func(t *testing.T) {
		d := validIDPCreate()
		d.AllowTokenFederation = false
		d.AllowedAudiences = nil
		assert.NoError(t, d.Validate())
	})
}

func TestIdentityProviderUpdateValidation_TokenFederation(t *testing.T) {
	t.Run("token federation enabled with audiences passes", func(t *testing.T) {
		d := validIDPUpdate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = []string{"app-1"}
		assert.NoError(t, d.Validate())
	})

	t.Run("token federation enabled requires audiences", func(t *testing.T) {
		d := validIDPUpdate()
		d.AllowTokenFederation = true
		d.AllowedAudiences = nil
		require.Error(t, d.Validate())
	})
}

func validIDPUpdate() IdentityProviderUpdateRequestDTO {
	return IdentityProviderUpdateRequestDTO{
		Name:             "my-idp",
		DisplayName:      "My Identity Provider",
		Provider:         shared.IDPProviderGoogle,
		ProviderType:     shared.IDPTypeSocial,
		Issuer:           "https://accounts.google.com",
		ProviderClientID: "test-client",
		Config:           validOAuth2ConfigJSON(),
		Status:           shared.StatusActive,
	}
}
