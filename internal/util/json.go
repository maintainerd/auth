package util

import "encoding/json"

func ToJSON(data any) string {
	bytes, _ := json.Marshal(data)
	return string(bytes)
}
