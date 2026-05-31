package secpolicy

import validation "github.com/go-ozzo/ozzo-validation/v4"

func (r SecuritySettingUpdateConfigRequestDTO) Validate() error {
	if len(r) == 0 {
		return validation.NewError("validation_error", "Config cannot be empty")
	}
	return nil
}
