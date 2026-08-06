package setup

import (
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantMetadataDto_Validate(t *testing.T) {
	t.Run("valid empty (all optional)", func(t *testing.T) {
		d := TenantMetadataDTO{}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid with logo url", func(t *testing.T) {
		d := TenantMetadataDTO{ApplicationLogoURL: strPtr("https://cdn.example.com/logo.png")}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid logo url", func(t *testing.T) {
		d := TenantMetadataDTO{ApplicationLogoURL: strPtr("not-a-url")}
		require.Error(t, d.Validate())
	})

	t.Run("invalid language format", func(t *testing.T) {
		d := TenantMetadataDTO{Language: strPtr("english")}
		require.Error(t, d.Validate())
	})

	t.Run("valid language 2-char", func(t *testing.T) {
		d := TenantMetadataDTO{Language: strPtr("en")}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid language with region", func(t *testing.T) {
		d := TenantMetadataDTO{Language: strPtr("en-US")}
		assert.NoError(t, d.Validate())
	})
}

func TestCreateTenantRequestDto_Validate(t *testing.T) {
	valid := CreateTenantRequestDTO{
		Name:        "my-tenant",
		DisplayName: "My Display Name",
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("valid with metadata", func(t *testing.T) {
		d := valid
		d.Metadata = &TenantMetadataDTO{Language: strPtr("en")}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid metadata propagates error", func(t *testing.T) {
		d := valid
		d.Metadata = &TenantMetadataDTO{ApplicationLogoURL: strPtr("not-a-url")}
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

	// The bootstrap name becomes a DNS label, so setup must reject everything the
	// tenant package would reject on a later update.
	for name, candidate := range map[string]string{
		"uppercase":       "MyTenant",
		"whitespace":      "My Tenant",
		"underscore":      "my_tenant",
		"dot":             "my.tenant",
		"leading hyphen":  "-mytenant",
		"trailing hyphen": "mytenant-",
		"too long":        strings.Repeat("a", 64),
	} {
		t.Run("name rejects "+name, func(t *testing.T) {
			d := valid
			d.Name = candidate
			require.Error(t, d.Validate())
		})
	}

	t.Run("description longer than the tenant update cap", func(t *testing.T) {
		d := valid
		d.Description = strPtr(strings.Repeat("a", 201))
		require.Error(t, d.Validate())
	})

	t.Run("missing display_name", func(t *testing.T) {
		d := valid
		d.DisplayName = ""
		require.Error(t, d.Validate())
	})
}

func TestRegisterControlServiceRequestDTO_Validate(t *testing.T) {
	valid := RegisterControlServiceRequestDTO{Name: "core", DisplayName: "Core"}

	t.Run("valid minimal", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("valid with optional fields", func(t *testing.T) {
		d := valid
		d.Description = strPtr("Maintainerd Core")
		d.Version = "v1.2.3"
		assert.NoError(t, d.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid name characters", func(t *testing.T) {
		d := valid
		d.Name = "core service"
		require.Error(t, d.Validate())
	})

	t.Run("missing display name", func(t *testing.T) {
		d := valid
		d.DisplayName = ""
		require.Error(t, d.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		d := valid
		long := make([]byte, 501)
		for i := range long {
			long[i] = 'a'
		}
		d.Description = strPtr(string(long))
		require.Error(t, d.Validate())
	})

	t.Run("invalid version", func(t *testing.T) {
		d := valid
		d.Version = "v1 beta"
		require.Error(t, d.Validate())
	})
}

func TestCreateAdminRequestDto_Validate(t *testing.T) {
	valid := CreateAdminRequestDTO{
		Username: "adminuser",
		Fullname: ptr.Ptr("Admin User"),
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
		d := CreateProfileRequestDTO{FirstName: "John"}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing first_name", func(t *testing.T) {
		d := CreateProfileRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("invalid email", func(t *testing.T) {
		d := CreateProfileRequestDTO{FirstName: "John", Email: strPtr("bad-email")}
		require.Error(t, d.Validate())
	})

	t.Run("country not 2 chars", func(t *testing.T) {
		d := CreateProfileRequestDTO{FirstName: "John", Country: strPtr("USA")}
		require.Error(t, d.Validate())
	})

	t.Run("invalid birthdate format", func(t *testing.T) {
		d := CreateProfileRequestDTO{FirstName: "John", Birthdate: strPtr("01-25-1990")}
		require.Error(t, d.Validate())
	})

	t.Run("valid birthdate format", func(t *testing.T) {
		d := CreateProfileRequestDTO{FirstName: "John", Birthdate: strPtr("1990-01-25")}
		assert.NoError(t, d.Validate())
	})
}
