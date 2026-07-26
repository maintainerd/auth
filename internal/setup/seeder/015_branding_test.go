package seeder

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemBrandingThemes(t *testing.T) {
	themes := systemBrandingThemes()
	require.Len(t, themes, 3)

	names := make([]string, 0, len(themes))
	activeCount := 0

	for _, theme := range themes {
		names = append(names, theme.name)
		if theme.active {
			activeCount++
		}

		var metadata map[string]any
		require.NoError(t, json.Unmarshal([]byte(theme.metadata), &metadata))
	}

	assert.Equal(t, []string{"default", "light", "dark"}, names)
	assert.Equal(t, 1, activeCount)
	assert.True(t, themes[0].active)
}

func TestLegacySystemBrandingThemes(t *testing.T) {
	themes := legacySystemBrandingThemes()
	require.Len(t, themes, 2)

	assert.Equal(t, "maintainerd-light", themes[0].name)
	assert.Equal(t, "light", themes[0].replacementName)
	assert.Equal(t, "maintainerd-dark", themes[1].name)
	assert.Equal(t, "dark", themes[1].replacementName)
}
