package jsonutil

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// JSONToMap decodes a JSON/JSONB value into a map. Empty or invalid input
// returns an empty map so callers can safely read keys without nil checks.
func JSONToMap(data datatypes.JSON) map[string]any {
	result := make(map[string]any)
	if len(data) == 0 {
		return result
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return make(map[string]any)
	}
	if result == nil {
		return make(map[string]any)
	}
	return result
}
