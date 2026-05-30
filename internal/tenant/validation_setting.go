package tenant

import validation "github.com/go-ozzo/ozzo-validation/v4"

// Validate ensures the request body is not empty.
func (r TenantSettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}
