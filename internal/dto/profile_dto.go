package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/maintainerd/auth/internal/model"
)

type ProfileRequest struct {
	// Basic Identity Information
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	Suffix      *string `json:"suffix,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`

	// Personal Information
	Birthdate *string `json:"birthdate,omitempty"`
	Gender    *string `json:"gender,omitempty"`
	Bio       *string `json:"bio,omitempty"`

	// Contact Information
	Phone *string `json:"phone,omitempty"`
	Email *string `json:"email,omitempty"`

	// Location Information (minimal)
	City    *string `json:"city,omitempty"`    // Current city
	Country *string `json:"country,omitempty"` // ISO 3166-1 alpha-2 code

	// Social/Web Presence
	WebsiteURL *string `json:"website_url,omitempty"` // Personal website/portfolio

	// Media & Assets (auth-centric)
	AvatarURL *string `json:"avatar_url,omitempty"` // User profile picture
}

func (r ProfileRequest) Validate() error {
	return validation.ValidateStruct(&r,
		// Basic Identity Information
		validation.Field(&r.FirstName,
			validation.Required.Error("First name is required"),
			validation.RuneLength(1, 100).Error("First name must be 1-100 characters"),
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
		validation.Field(&r.DisplayName,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Display name must be at most 100 characters"),
		),

		// Personal Information
		validation.Field(&r.Birthdate,
			validation.NilOrNotEmpty,
			validation.By(validateDateFormat),
		),
		validation.Field(&r.Gender,
			validation.NilOrNotEmpty,
			validation.In("male", "female", "other", "prefer_not_to_say").Error("Gender must be male, female, other, or prefer_not_to_say"),
		),
		validation.Field(&r.Bio,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 1000).Error("Bio must be at most 1000 characters"),
		),

		// Contact Information
		validation.Field(&r.Phone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 20).Error("Phone must be at most 20 characters"),
		),
		validation.Field(&r.Email,
			validation.NilOrNotEmpty,
			is.Email.Error("Invalid email format"),
			validation.RuneLength(0, 255).Error("Email must be at most 255 characters"),
		),

		// Location Information (minimal)
		validation.Field(&r.City,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("City must be at most 100 characters"),
		),
		validation.Field(&r.Country,
			validation.NilOrNotEmpty,
			validation.RuneLength(2, 2).Error("Country must be a 2-character ISO code (e.g., US, PH, CA)"),
		),

		// Social/Web Presence
		validation.Field(&r.WebsiteURL,
			validation.NilOrNotEmpty,
			is.URL.Error("Invalid website URL format"),
			validation.RuneLength(0, 1000).Error("Website URL must be at most 1000 characters"),
		),

		// Media & Assets
		validation.Field(&r.AvatarURL,
			validation.NilOrNotEmpty,
			is.URL.Error("Invalid avatar URL format"),
			validation.RuneLength(0, 1000).Error("Avatar URL must be at most 1000 characters"),
		),
	)
}

// validateDateFormat ensures the date is in "2006-01-02" format.
func validateDateFormat(value any) error {
	if str, ok := value.(*string); ok && str != nil {
		_, err := time.Parse("2006-01-02", *str)
		return err
	}
	return nil
}

type ProfileResponse struct {
	ProfileUUID string `json:"profile_uuid"`

	// Basic Identity Information
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	Suffix      *string `json:"suffix,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`

	// Personal Information
	Birthdate *time.Time `json:"birthdate,omitempty"`
	Gender    *string    `json:"gender,omitempty"`
	Bio       *string    `json:"bio,omitempty"`

	// Contact Information
	Phone *string `json:"phone,omitempty"`
	Email *string `json:"email,omitempty"`

	// Location Information (minimal)
	City    *string `json:"city,omitempty"`    // Current city
	Country *string `json:"country,omitempty"` // ISO 3166-1 alpha-2 code

	// Social/Web Presence
	WebsiteURL *string `json:"website_url,omitempty"` // Personal website/portfolio

	// Media & Assets (auth-centric)
	AvatarURL *string `json:"avatar_url,omitempty"` // User profile picture

	// System Fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewProfileResponse(p *model.Profile) *ProfileResponse {
	return &ProfileResponse{
		ProfileUUID: p.ProfileUUID.String(),

		// Basic Identity Information
		FirstName:   p.FirstName,
		MiddleName:  p.MiddleName,
		LastName:    p.LastName,
		Suffix:      p.Suffix,
		DisplayName: p.DisplayName,

		// Personal Information
		Birthdate: p.Birthdate,
		Gender:    p.Gender,
		Bio:       p.Bio,

		// Contact Information
		Phone: p.Phone,
		Email: p.Email,

		// Location Information (minimal)
		City:    p.City,
		Country: p.Country,

		// Social/Web Presence
		WebsiteURL: p.WebsiteURL,

		// Media & Assets (auth-centric)
		AvatarURL: p.AvatarURL,

		// System Fields
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
