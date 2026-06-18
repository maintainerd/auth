package tenant

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Validate ensures the request body is not empty.
func (r TenantSettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}

func (r TenantSettingUpdateConfigRequestDTO) ValidateRateLimitConfig() error {
	if err := r.Validate(); err != nil {
		return err
	}
	for key, value := range r {
		switch key {
		case "enabled", "per_ip", "per_api_key":
			if _, ok := value.(bool); !ok {
				return validation.NewError("validation_error", key+" must be a boolean")
			}
		case "requests_per_window":
			n, ok := auditConfigInt(value)
			if !ok {
				return validation.NewError("validation_error", "requests_per_window must be a number")
			}
			if n < 1 || n > 100000 {
				return validation.NewError("validation_error", "requests_per_window must be between 1 and 100000")
			}
		case "window_duration_seconds":
			n, ok := auditConfigInt(value)
			if !ok {
				return validation.NewError("validation_error", "window_duration_seconds must be a number")
			}
			if n < 1 || n > 3600 {
				return validation.NewError("validation_error", "window_duration_seconds must be between 1 and 3600")
			}
		case "exempt_ips":
			if err := validateStringList(value, "exempt_ips"); err != nil {
				return err
			}
		case "endpoint_overrides":
			if err := validateEndpointOverrides(value); err != nil {
				return err
			}
		default:
			return validation.NewError("validation_error", fmt.Sprintf("unknown rate_limit_config field: %s", key))
		}
	}
	return nil
}

func (r TenantSettingUpdateConfigRequestDTO) ValidateAuditConfig() error {
	if err := r.Validate(); err != nil {
		return err
	}
	for key, value := range r {
		switch key {
		case "enabled", "pii_masking":
			if _, ok := value.(bool); !ok {
				return validation.NewError("validation_error", key+" must be a boolean")
			}
		case "retention_days":
			days, ok := auditConfigInt(value)
			if !ok {
				return validation.NewError("validation_error", "retention_days must be a number")
			}
			if days < 1 || days > 3650 {
				return validation.NewError("validation_error", "retention_days must be between 1 and 3650")
			}
		case "log_level":
			level, ok := value.(string)
			if !ok {
				return validation.NewError("validation_error", "log_level must be a string")
			}
			switch strings.ToLower(level) {
			case "debug", "info", "warn", "critical":
			default:
				return validation.NewError("validation_error", "log_level must be one of: debug, info, warn, critical")
			}
		case "event_types":
			if err := validateAuditEventTypes(value); err != nil {
				return err
			}
		default:
			return validation.NewError("validation_error", fmt.Sprintf("unknown audit_config field: %s", key))
		}
	}
	return nil
}

func (r TenantSettingUpdateConfigRequestDTO) ValidateMaintenanceConfig() error {
	if err := r.Validate(); err != nil {
		return err
	}
	var start, end *time.Time
	for key, value := range r {
		switch key {
		case "enabled":
			if _, ok := value.(bool); !ok {
				return validation.NewError("validation_error", "enabled must be a boolean")
			}
		case "message":
			message, ok := value.(string)
			if !ok {
				return validation.NewError("validation_error", "message must be a string")
			}
			if strings.TrimSpace(message) == "" {
				return validation.NewError("validation_error", "message cannot be empty")
			}
		case "scheduled_start":
			parsed, err := optionalRFC3339Time(value, "scheduled_start")
			if err != nil {
				return err
			}
			start = parsed
		case "scheduled_end":
			parsed, err := optionalRFC3339Time(value, "scheduled_end")
			if err != nil {
				return err
			}
			end = parsed
		default:
			return validation.NewError("validation_error", fmt.Sprintf("unknown maintenance_config field: %s", key))
		}
	}
	if start != nil && end != nil && !start.Before(*end) {
		return validation.NewError("validation_error", "scheduled_start must be before scheduled_end")
	}
	return nil
}

func auditConfigInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		if v != float64(int(v)) {
			return 0, false
		}
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(v)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func optionalRFC3339Time(value any, field string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	raw, ok := value.(string)
	if !ok {
		return nil, validation.NewError("validation_error", field+" must be an RFC3339 timestamp or null")
	}
	if strings.TrimSpace(raw) == "" {
		return nil, validation.NewError("validation_error", field+" must be an RFC3339 timestamp or null")
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, validation.NewError("validation_error", field+" must be an RFC3339 timestamp or null")
	}
	return &parsed, nil
}

func validateAuditEventTypes(value any) error {
	switch values := value.(type) {
	case []string:
		for _, eventType := range values {
			if eventType == "" {
				return validation.NewError("validation_error", "event_types cannot contain empty values")
			}
		}
	case []any:
		for _, item := range values {
			eventType, ok := item.(string)
			if !ok || eventType == "" {
				return validation.NewError("validation_error", "event_types must be a list of non-empty strings")
			}
		}
	default:
		return validation.NewError("validation_error", "event_types must be a list of strings")
	}
	return nil
}

func validateStringList(value any, field string) error {
	switch values := value.(type) {
	case []string:
		for _, v := range values {
			if v == "" {
				return validation.NewError("validation_error", field+" cannot contain empty values")
			}
		}
	case []any:
		for _, item := range values {
			s, ok := item.(string)
			if !ok || s == "" {
				return validation.NewError("validation_error", field+" must be a list of non-empty strings")
			}
		}
	default:
		return validation.NewError("validation_error", field+" must be a list of strings")
	}
	return nil
}

func validateEndpointOverrides(value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return validation.NewError("validation_error", "endpoint_overrides must be an object")
	}
	for key, v := range m {
		if key == "" {
			return validation.NewError("validation_error", "endpoint_overrides keys cannot be empty")
		}
		n, ok := auditConfigInt(v)
		if !ok || n < 1 {
			return validation.NewError("validation_error", "endpoint_overrides values must be positive integers")
		}
	}
	return nil
}
