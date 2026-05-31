package user

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/shared"
)

func (r UserSettingRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Timezone, validation.NilOrNotEmpty, validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters")),
		validation.Field(&r.PreferredLanguage, validation.NilOrNotEmpty, validation.RuneLength(2, 10).Error("Preferred language must be 2-10 characters")),
		validation.Field(&r.Locale, validation.NilOrNotEmpty, validation.RuneLength(2, 10).Error("Locale must be 2-10 characters")),
		validation.Field(&r.PreferredContactMethod, validation.NilOrNotEmpty, validation.In(shared.ContactMethodEmail, shared.ContactMethodPhone, shared.ContactMethodSMS).Error("Preferred contact method must be email, phone, or sms")),
		validation.Field(&r.ProfileVisibility, validation.NilOrNotEmpty, validation.In(shared.VisibilityPublic, shared.VisibilityPrivate, shared.VisibilityFriends).Error("Profile visibility must be public, private, or friends")),
		validation.Field(&r.EmergencyContactName, validation.NilOrNotEmpty, validation.RuneLength(0, 200).Error("Emergency contact name must be at most 200 characters")),
		validation.Field(&r.EmergencyContactPhone, validation.NilOrNotEmpty, validation.RuneLength(0, 20).Error("Emergency contact phone must be at most 20 characters")),
		validation.Field(&r.EmergencyContactEmail, validation.NilOrNotEmpty, is.Email.Error("Invalid emergency contact email format"), validation.RuneLength(0, 255).Error("Emergency contact email must be at most 255 characters")),
		validation.Field(&r.EmergencyContactRelation, validation.NilOrNotEmpty, validation.RuneLength(0, 50).Error("Emergency contact relation must be at most 50 characters")),
	)
}
