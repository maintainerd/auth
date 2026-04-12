package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// TenantSettingConfigResponseDTO returns a single JSONB config as a map.
type TenantSettingConfigResponseDTO map[string]any

// TenantSettingUpdateConfigRequestDTO is the request body for updating a
// tenant setting config section.
type TenantSettingUpdateConfigRequestDTO map[string]any

// Validate ensures the request body is not empty.
func (r TenantSettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}
