package branding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The theming read must not carry the logo bytes: the response only needs
// logo_url, and the browser fetches the image from the cached logo endpoint. A
// SELECT * here costs up to 256 KB on every login page render, read and thrown
// away.
func TestBrandingPublicColumnsExcludeLogoData(t *testing.T) {
	assert.NotContains(t, brandingPublicColumns, "logo_data",
		"logo_data on the theming read is 256 KB fetched and discarded per page load")

	// The columns the payload actually renders must all be present, or the
	// exclusion has quietly broken theming.
	for _, required := range []string{"branding_id", "tenant_id", "logo_url", "favicon_url", "settings"} {
		assert.Contains(t, brandingPublicColumns, required)
	}
}
