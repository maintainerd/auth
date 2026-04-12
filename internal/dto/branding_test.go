package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validBrandingUpdate() BrandingUpdateRequestDTO {
	return BrandingUpdateRequestDTO{
		CompanyName:  "Acme Corp",
		PrimaryColor: "#111111",
	}
}

func TestBrandingUpdateRequestDTO_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		assert.NoError(t, BrandingUpdateRequestDTO{}.Validate())
	})

	t.Run("valid full", func(t *testing.T) {
		assert.NoError(t, validBrandingUpdate().Validate())
	})

	t.Run("company_name too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.CompanyName = strings.Repeat("a", 256)
		require.Error(t, d.Validate())
	})

	t.Run("primary_color too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.PrimaryColor = strings.Repeat("a", 21)
		require.Error(t, d.Validate())
	})

	t.Run("secondary_color too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.SecondaryColor = strings.Repeat("a", 21)
		require.Error(t, d.Validate())
	})

	t.Run("accent_color too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.AccentColor = strings.Repeat("a", 21)
		require.Error(t, d.Validate())
	})

	t.Run("font_family too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.FontFamily = strings.Repeat("a", 101)
		require.Error(t, d.Validate())
	})

	t.Run("logo_url too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.LogoURL = "https://example.com/" + strings.Repeat("a", 2030)
		require.Error(t, d.Validate())
	})

	t.Run("logo_url invalid format", func(t *testing.T) {
		d := validBrandingUpdate()
		d.LogoURL = "not a url"
		require.Error(t, d.Validate())
	})

	t.Run("logo_url valid", func(t *testing.T) {
		d := validBrandingUpdate()
		d.LogoURL = "https://cdn.example.com/logo.png"
		assert.NoError(t, d.Validate())
	})

	t.Run("favicon_url invalid format", func(t *testing.T) {
		d := validBrandingUpdate()
		d.FaviconURL = "not-a-url"
		require.Error(t, d.Validate())
	})

	t.Run("support_url invalid format", func(t *testing.T) {
		d := validBrandingUpdate()
		d.SupportURL = "bad url"
		require.Error(t, d.Validate())
	})

	t.Run("privacy_policy_url invalid format", func(t *testing.T) {
		d := validBrandingUpdate()
		d.PrivacyPolicyURL = "bad"
		require.Error(t, d.Validate())
	})

	t.Run("terms_of_service_url invalid format", func(t *testing.T) {
		d := validBrandingUpdate()
		d.TermsOfServiceURL = "bad"
		require.Error(t, d.Validate())
	})

	t.Run("terms_of_service_url valid", func(t *testing.T) {
		d := validBrandingUpdate()
		d.TermsOfServiceURL = "https://example.com/tos"
		assert.NoError(t, d.Validate())
	})

	t.Run("custom_css too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.CustomCSS = strings.Repeat("a", 50001)
		require.Error(t, d.Validate())
	})

	t.Run("custom_css valid", func(t *testing.T) {
		d := validBrandingUpdate()
		d.CustomCSS = "body { color: red; }"
		assert.NoError(t, d.Validate())
	})
}
