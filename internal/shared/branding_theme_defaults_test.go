package shared

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemBrandingThemeDefaultsIncludeConsoleThemingComponents(t *testing.T) {
	required := []string{
		"topPanelCreateButton",
		"outlineButton",
		"ghostButton",
		"tableContainer",
		"tableHeader",
		"tableRow",
		"tableCell",
		"iconContainer",
		"listingItem",
		"listingItemIcon",
		"listingItemMeta",
		"listingSubContainer",
		"optionCard",
		"switchSubContainer",
		"checkboxSubContainer",
		"textarea",
		"datePicker",
		"alert",
	}

	for _, theme := range SystemBrandingThemeDefaults() {
		t.Run(theme.Name, func(t *testing.T) {
			var metadata map[string]any
			require.NoError(t, json.Unmarshal([]byte(theme.Metadata), &metadata))
			components, ok := metadata["components"].(map[string]any)
			require.True(t, ok)

			for _, component := range required {
				raw, ok := components[component].(map[string]any)
				require.Truef(t, ok, "missing %s", component)
				for _, key := range []string{"background", "hoverColor", "borderColor", "borderThickness", "borderRadius", "textColor", "size"} {
					require.NotEmptyf(t, raw[key], "%s.%s", component, key)
				}
			}
		})
	}
}
