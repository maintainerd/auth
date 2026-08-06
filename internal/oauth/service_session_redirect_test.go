package oauth

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The redirect used to be built by appending "?state=", which produced a second
// '?' on any registered logout URI that already carried a query string: the RP
// then saw a single parameter literally named "…?state" and lost both its own
// parameters and its CSRF state.
func TestAppendLogoutState(t *testing.T) {
	t.Run("no state leaves the URI untouched", func(t *testing.T) {
		assert.Equal(t, "https://app.example.com/bye", appendLogoutState("https://app.example.com/bye", ""))
	})

	t.Run("state is added to a URI with no query", func(t *testing.T) {
		assert.Equal(t, "https://app.example.com/bye?state=abc",
			appendLogoutState("https://app.example.com/bye", "abc"))
	})

	t.Run("an existing query string is preserved", func(t *testing.T) {
		got := appendLogoutState("https://app.example.com/bye?lang=en", "abc")
		u, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, "en", u.Query().Get("lang"))
		assert.Equal(t, "abc", u.Query().Get("state"))
	})

	t.Run("state is escaped", func(t *testing.T) {
		got := appendLogoutState("https://app.example.com/bye", "a b&c=d")
		u, err := url.Parse(got)
		require.NoError(t, err)
		assert.Equal(t, "a b&c=d", u.Query().Get("state"))
	})
}
