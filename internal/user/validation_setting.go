package user

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func (r UserSettingRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Timezone, validation.NilOrNotEmpty, validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters")),
		validation.Field(&r.PreferredLanguage, validation.NilOrNotEmpty, validation.RuneLength(2, 10).Error("Preferred language must be 2-10 characters")),
		validation.Field(&r.Locale, validation.NilOrNotEmpty, validation.RuneLength(2, 10).Error("Locale must be 2-10 characters")),
	)
}
