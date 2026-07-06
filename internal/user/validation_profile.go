package user

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

func (r ProfileRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.FirstName,
			validation.Required.Error("First name is required"),
			validation.RuneLength(1, 100).Error("First name must be 1-100 characters"),
		),
		validation.Field(&r.MiddleName, validation.NilOrNotEmpty, validation.RuneLength(0, 100).Error("Middle name must be at most 100 characters")),
		validation.Field(&r.LastName, validation.NilOrNotEmpty, validation.RuneLength(0, 100).Error("Last name must be at most 100 characters")),
		validation.Field(&r.DisplayName, validation.NilOrNotEmpty, validation.RuneLength(0, 100).Error("Display name must be at most 100 characters")),
		validation.Field(&r.Birthdate, validation.NilOrNotEmpty, validation.By(validateDateFormat)),
		validation.Field(&r.Gender, validation.NilOrNotEmpty, validation.In(shared.GenderMale, shared.GenderFemale, shared.GenderOther, shared.GenderPreferNotToSay).Error("Gender must be male, female, other, or prefer_not_to_say")),
		validation.Field(&r.Email, validation.NilOrNotEmpty, is.Email.Error("Invalid email format"), validation.RuneLength(0, 255).Error("Email must be at most 255 characters")),
		validation.Field(&r.Timezone, validation.NilOrNotEmpty, validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters")),
		validation.Field(&r.Language, validation.NilOrNotEmpty, validation.RuneLength(0, 10).Error("Language must be at most 10 characters")),
		validation.Field(&r.ProfileURL, validation.NilOrNotEmpty, is.URL.Error("Invalid profile URL format"), validation.RuneLength(0, 1000).Error("Profile URL must be at most 1000 characters")),
	)
}

func (f ProfileFilterDTO) Validate() error {
	return f.PaginationRequestDTO.Validate()
}

func validateDateFormat(value any) error {
	if str, ok := value.(*string); ok && str != nil {
		if _, err := time.Parse("2006-01-02", *str); err != nil {
			return validation.NewError("validation_invalid_date", "Birthdate must be in YYYY-MM-DD format (e.g., 1990-01-25)")
		}
	}
	return nil
}
