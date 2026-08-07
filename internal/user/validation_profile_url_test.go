package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The URL is rendered as an <img> src for whoever views the profile.
func TestValidateProfileURL(t *testing.T) {
	t.Run("accepts https", func(t *testing.T) {
		got, err := ValidateProfileURL("  https://cdn.example.com/a.png  ")
		require.NoError(t, err)
		assert.Equal(t, "https://cdn.example.com/a.png", got)
	})

	t.Run("empty is allowed and means no picture", func(t *testing.T) {
		got, err := ValidateProfileURL("   ")
		require.NoError(t, err)
		assert.Equal(t, "", got)
	})

	// http on a page served over https is mixed content — the browser blocks it
	// and the avatar silently fails to load.
	t.Run("rejects http", func(t *testing.T) {
		_, err := ValidateProfileURL("http://cdn.example.com/a.png")
		require.Error(t, err)
	})

	// A javascript: or data: URL in an <img> src is an injection primitive.
	t.Run("rejects non-http schemes", func(t *testing.T) {
		for _, raw := range []string{
			"javascript:alert(1)",
			"data:image/svg+xml;base64,PHN2Zz48c2NyaXB0PmFsZXJ0KDEpPC9zY3JpcHQ+PC9zdmc+",
			"file:///etc/passwd",
		} {
			_, err := ValidateProfileURL(raw)
			assert.Error(t, err, raw)
		}
	})

	// Credentials in a URL leak into markup and referrer headers.
	t.Run("rejects embedded credentials", func(t *testing.T) {
		_, err := ValidateProfileURL("https://user:pass@cdn.example.com/a.png")
		require.Error(t, err)
	})

	t.Run("rejects a relative path", func(t *testing.T) {
		_, err := ValidateProfileURL("/avatars/me.png")
		require.Error(t, err)
	})

	t.Run("rejects an over-long URL", func(t *testing.T) {
		_, err := ValidateProfileURL("https://example.com/" + strings.Repeat("a", 2100))
		require.Error(t, err)
	})
}
