package event

import (
	"encoding/json"

	"gorm.io/datatypes"
)

func marshalStringSlice(s []string) (datatypes.JSON, error) {
	if len(s) == 0 {
		return datatypes.JSON("[]"), nil
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(raw), nil
}

func unmarshalStringSliceRaw(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var s []string
	if err := json.Unmarshal(raw, &s); err != nil {
		return []string{}
	}
	return s
}
