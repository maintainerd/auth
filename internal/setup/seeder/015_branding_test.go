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
		assert.Contains(t, metadata, "colors")
		assert.Contains(t, metadata, "font")
		assert.Contains(t, metadata, "components")
		assertThemeColorsComplete(t, metadata)
		assertThemeComponentsComplete(t, metadata)
	}

	assert.Equal(t, []string{"default", "light", "dark"}, names)
	assert.Equal(t, 1, activeCount)
	assert.True(t, themes[0].active)
}

func assertThemeColorsComplete(t *testing.T, metadata map[string]any) {
	t.Helper()

	colors, ok := metadata["colors"].(map[string]any)
	require.True(t, ok)
	for _, key := range []string{
		"primary",
		"secondary",
		"accent",
		"appBackground",
		"topPanelBackground",
		"topPanelBorder",
		"topPanelText",
		"authPageBackground",
		"authFormPanelBackground",
		"authFormPanelBorder",
		"authFormPanelText",
		"authVisualPanelBackground",
		"authVisualPanelText",
		"authVisualPanelOverlay",
		"authDecorativeLight",
		"authDecorativeDark",
		"authProgressPanelBackground",
		"authSecurityPanelBackground",
		"sidePanelBackground",
		"sidePanelBorder",
		"sidePanelSectionText",
		"sidePanelItemIcon",
		"sidePanelItemIconHover",
		"sidePanelItemActiveIcon",
		"sidePanelItemHoverText",
		"sidePanelChevron",
		"cardBackground",
		"textPrimary",
		"textMuted",
		"border",
	} {
		assert.NotEmptyf(t, colors[key], "missing colors.%s", key)
	}
}

func assertThemeComponentsComplete(t *testing.T, metadata map[string]any) {
	t.Helper()

	components, ok := metadata["components"].(map[string]any)
	require.True(t, ok)

	for _, name := range []string{
		"topPanelControl",
		"topPanelDropdownTrigger",
		"topPanelProfileTrigger",
		"sidePanelSectionLabel",
		"sidePanelItem",
		"sidePanelItemActive",
		"sidePanelSubItem",
		"sidePanelSubItemActive",
		"primaryButton",
		"secondaryButton",
		"destructiveButton",
		"outlineButton",
		"ghostButton",
		"iconContainer",
		"card",
		"alert",
		"input",
		"textarea",
		"switch",
		"switchSubContainer",
		"checkboxSubContainer",
		"optionCard",
	} {
		assertComponentThemeComplete(t, components, name)
	}

	switchTheme, ok := components["switch"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, switchTheme["uncheckedBackground"])
	assert.NotEmpty(t, switchTheme["thumbColor"])

	badges, ok := components["badges"].(map[string]any)
	require.True(t, ok)
	for _, group := range []string{
		"positive",
		"in-progress",
		"neutral",
		"negative",
	} {
		assertComponentThemeComplete(t, badges, group)
		badgeTheme, ok := badges[group].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, badgeTheme["dotColor"])
	}
}

func assertComponentThemeComplete(t *testing.T, parent map[string]any, name string) {
	t.Helper()

	component, ok := parent[name].(map[string]any)
	require.Truef(t, ok, "missing component theme %s", name)
	for _, key := range []string{
		"background",
		"hoverColor",
		"borderColor",
		"borderThickness",
		"borderRadius",
		"textColor",
		"size",
	} {
		assert.NotEmptyf(t, component[key], "missing %s.%s", name, key)
	}
}

func TestLegacySystemBrandingThemes(t *testing.T) {
	themes := legacySystemBrandingThemes()
	require.Len(t, themes, 2)

	assert.Equal(t, "maintainerd-light", themes[0].name)
	assert.Equal(t, "light", themes[0].replacementName)
	assert.Equal(t, "maintainerd-dark", themes[1].name)
	assert.Equal(t, "dark", themes[1].replacementName)
}
