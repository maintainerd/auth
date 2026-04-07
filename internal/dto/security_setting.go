package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Security setting config response - returns config directly
type SecuritySettingConfigResponseDTO map[string]any

// Update config request - accepts config directly
type SecuritySettingUpdateConfigRequestDTO map[string]any

func (r SecuritySettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}
