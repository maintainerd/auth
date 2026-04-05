package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantMetadataDto_Validate(t *testing.T) {
	t.Run("valid empty (all optional)", func(t *testing.T) {
		d := TenantMetadataDto{}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid with logo url", func(t *testing.T) {
		d := TenantMetadataDto{ApplicationLogoURL: strPtr("https://cdn.example.com/logo.png")}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid logo url", func(t *testing.T) {
		d := TenantMetadataDto{ApplicationLogoURL: strPtr("not-a-url")}
		require.Error(t, d.Validate())
	})

	t.Run("invalid language format", func(t *testing.T) {
		d := TenantMetadataDto{Language: strPtr("english")}
		require.Error(t, d.Validate())
	})

	t.Run("valid language 2-char", func(t *testing.T) {
		d := TenantMetadataDto{Language: strPtr("en")}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid language with region", func(t *testing.T) {
		d := TenantMetadataDto{Language: strPtr("en-US")}
		assert.NoError(t, d.Validate())
	})
}

func TestCreateTenantRequestDto_Validate(t *testing.T) {
	valid := CreateTenantRequestDto{
		Name:        "My Tenant",
		DisplayName: "My Display Name",
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("valid with metadata", func(t *testing.T) {
		d := valid
		d.Metadata = &TenantMetadataDto{Language: strPtr("en")}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid metadata propagates error", func(t *testing.T) {
		d := valid
		d.Metadata = &TenantMetadataDto{ApplicationLogoURL: strPtr("not-a-url")}
		require.Error(t, d.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := valid
		d.Name = "A"
		require.Error(t, d.Validate())
	})

	t.Run("name invalid characters", func(t *testing.T) {
		d := valid
		d.Name = "Name@Invalid!"
		require.Error(t, d.Validate())
	})

	t.Run("missing display_name", func(t *testing.T) {
		d := valid
		d.DisplayName = ""
		require.Error(t, d.Validate())
	})
}

func TestCreateAdminRequestDto_Validate(t *testing.T) {
	valid := CreateAdminRequestDto{
		Username: "adminuser",
		Fullname: "Admin User",
		Password: "SecurePass1!",
		Email:    "admin@example.com",
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("username too short", func(t *testing.T) {
		d := valid
		d.Username = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("invalid username characters", func(t *testing.T) {
		d := valid
		d.Username = "user name!"
		require.Error(t, d.Validate())
	})

	t.Run("password too short", func(t *testing.T) {
		d := valid
		d.Password = "short"
		require.Error(t, d.Validate())
	})

	t.Run("invalid email", func(t *testing.T) {
		d := valid
		d.Email = "not-an-email"
		require.Error(t, d.Validate())
	})
}

func TestCreateProfileRequestDto_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		d := CreateProfileRequestDto{FirstName: "John"}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing first_name", func(t *testing.T) {
		d := CreateProfileRequestDto{}
		require.Error(t, d.Validate())
	})

	t.Run("invalid email", func(t *testing.T) {
		d := CreateProfileRequestDto{FirstName: "John", Email: strPtr("bad-email")}
		require.Error(t, d.Validate())
	})

	t.Run("country not 2 chars", func(t *testing.T) {
		d := CreateProfileRequestDto{FirstName: "John", Country: strPtr("USA")}
		require.Error(t, d.Validate())
	})
}
