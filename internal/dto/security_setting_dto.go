package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Security setting config response - returns config directly
type SecuritySettingConfigResponseDto map[string]interface{}

// Update config request - accepts config directly
type SecuritySettingUpdateConfigRequestDto map[string]interface{}

func (r SecuritySettingUpdateConfigRequestDto) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}
