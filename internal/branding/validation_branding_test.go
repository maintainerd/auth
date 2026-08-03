package branding

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validBrandingUpdate() BrandingUpdateRequestDTO {
	return BrandingUpdateRequestDTO{
		CompanyName: "Acme Corp",
		Layout:      "centered",
	}
}

func TestHttpURL(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		assert.NoError(t, httpURL.Validate(""))
	})

	t.Run("invalid url", func(t *testing.T) {
		assert.Error(t, httpURL.Validate("not a url"))
	})

	t.Run("non http scheme", func(t *testing.T) {
		assert.Error(t, httpURL.Validate("ftp://example.com"))
	})

	t.Run("valid https", func(t *testing.T) {
		assert.NoError(t, httpURL.Validate("https://example.com"))
	})
}

func TestBrandingUpdateRequestDTO_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		assert.NoError(t, BrandingUpdateRequestDTO{}.Validate())
	})

	t.Run("valid full", func(t *testing.T) {
		assert.NoError(t, validBrandingUpdate().Validate())
	})

	for _, layout := range []string{"centered", "full_page", "split"} {
		t.Run("valid layout "+layout, func(t *testing.T) {
			d := validBrandingUpdate()
			d.Layout = layout
			assert.NoError(t, d.Validate())
		})
	}

	t.Run("invalid layout", func(t *testing.T) {
		d := validBrandingUpdate()
		d.Layout = "sidebar"
		require.Error(t, d.Validate())
	})

	t.Run("company_name too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.CompanyName = strings.Repeat("a", 256)
		require.Error(t, d.Validate())
	})

	t.Run("logo_label too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.LogoLabel = strings.Repeat("a", 256)
		require.Error(t, d.Validate())
	})

	t.Run("show_logo_label defaults to visible when omitted", func(t *testing.T) {
		d := validBrandingUpdate()
		assert.True(t, d.ShowLogoLabelOrDefault())
	})

	t.Run("show_logo_label preserves explicit false", func(t *testing.T) {
		hide := false
		d := validBrandingUpdate()
		d.ShowLogoLabel = &hide
		assert.False(t, d.ShowLogoLabelOrDefault())
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

	t.Run("logo_url rejects script scheme", func(t *testing.T) {
		d := validBrandingUpdate()
		d.LogoURL = "javascript:alert(1)"
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

	t.Run("name too long", func(t *testing.T) {
		d := validBrandingUpdate()
		d.Name = strings.Repeat("a", 101)
		require.Error(t, d.Validate())
	})

	t.Run("name valid", func(t *testing.T) {
		d := validBrandingUpdate()
		d.Name = "My Theme"
		assert.NoError(t, d.Validate())
	})
}
