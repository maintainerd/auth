package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/model"
)

type ProfileRequest struct {
	FirstName  string  `json:"first_name"`
	MiddleName *string `json:"middle_name,omitempty"`
	LastName   *string `json:"last_name,omitempty"`
	Suffix     *string `json:"suffix,omitempty"`
	Birthdate  *string `json:"birthdate,omitempty"`
	Gender     *string `json:"gender,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Email      *string `json:"email,omitempty"`
	Address    *string `json:"address,omitempty"`
	AvatarURL  *string `json:"avatar_url,omitempty"`
	CoverURL   *string `json:"cover_url,omitempty"`
}

func (r ProfileRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.FirstName,
			validation.Required.Error("First name is required"),
			validation.RuneLength(0, 100).Error("First name must be at most 100 characters"),
		),
		validation.Field(&r.MiddleName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Middle name must be at most 100 characters"),
		),
		validation.Field(&r.LastName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Last name must be at most 100 characters"),
		),
		validation.Field(&r.Suffix,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Suffix must be at most 50 characters"),
		),
		validation.Field(&r.Birthdate,
			validation.NilOrNotEmpty,
			validation.By(validateDateFormat),
		),
		validation.Field(&r.Gender,
			validation.NilOrNotEmpty,
			validation.In("male", "female", "other").Error("Gender must be male, female, or other"),
		),
		validation.Field(&r.Phone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 20).Error("Phone must be at most 20 characters"),
		),
		validation.Field(&r.Email,
			validation.NilOrNotEmpty,
			is.Email.Error("Invalid email format"),
			validation.RuneLength(0, 255).Error("Email must be at most 255 characters"),
		),
		validation.Field(&r.Address,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 1000).Error("Address must be at most 1000 characters"),
		),
		validation.Field(&r.AvatarURL,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 1000).Error("Avatar URL must be at most 1000 characters"),
		),
		validation.Field(&r.CoverURL,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 1000).Error("Cover URL must be at most 1000 characters"),
		),
	)
}

// validateDateFormat ensures the date is in "2006-01-02" format.
func validateDateFormat(value interface{}) error {
	if str, ok := value.(*string); ok && str != nil {
		_, err := time.Parse("2006-01-02", *str)
		return err
	}
	return nil
}

type ProfileResponse struct {
	ProfileUUID string     `json:"profile_uuid"`
	FirstName   string     `json:"first_name"`
	MiddleName  *string    `json:"middle_name,omitempty"`
	LastName    *string    `json:"last_name,omitempty"`
	Suffix      *string    `json:"suffix,omitempty"`
	Birthdate   *time.Time `json:"birthdate,omitempty"`
	Gender      *string    `json:"gender,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Email       *string    `json:"email,omitempty"`
	Address     *string    `json:"address,omitempty"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	CoverURL    *string    `json:"cover_url,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func NewProfileResponse(p *model.Profile) *ProfileResponse {
	return &ProfileResponse{
		ProfileUUID: p.ProfileUUID.String(),
		FirstName:   p.FirstName,
		MiddleName:  p.MiddleName,
		LastName:    p.LastName,
		Suffix:      p.Suffix,
		Birthdate:   p.Birthdate,
		Gender:      p.Gender,
		Phone:       p.Phone,
		Email:       p.Email,
		Address:     p.Address,
		AvatarURL:   p.AvatarURL,
		CoverURL:    p.CoverURL,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
