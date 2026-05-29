package tenant

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// unmarshalJSON decodes a datatypes.JSON column into a map, returning an empty
// map on nil/invalid input so callers can safely read keys.
func unmarshalJSON(data datatypes.JSON) map[string]any {
	result := make(map[string]any)
	if len(data) > 0 {
		_ = json.Unmarshal(data, &result)
	}
	return result
}
