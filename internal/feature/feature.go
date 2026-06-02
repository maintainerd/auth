package feature

import (
	"encoding/json"
	"strconv"
	"strings"
)

type Setting struct {
	FeatureFlags []byte
}

type Reader interface {
	FindByTenantID(tenantID int64) (*Setting, error)
}

func Enabled(reader Reader, tenantID int64, key string, defaultValue bool) bool {
	if reader == nil || tenantID <= 0 || strings.TrimSpace(key) == "" {
		return defaultValue
	}
	setting, err := reader.FindByTenantID(tenantID)
	if err != nil || setting == nil || len(setting.FeatureFlags) == 0 {
		return defaultValue
	}
	var flags map[string]any
	if err := json.Unmarshal(setting.FeatureFlags, &flags); err != nil {
		return defaultValue
	}
	raw, ok := flags[key]
	if !ok {
		return defaultValue
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}
