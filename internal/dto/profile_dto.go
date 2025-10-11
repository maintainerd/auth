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

	// Address Information
	Address    *string `json:"address,omitempty"`
	City       *string `json:"city,omitempty"`
	State      *string `json:"state,omitempty"`
	Country    *string `json:"country,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`

	// Professional Information
	Company    *string `json:"company,omitempty"`
	JobTitle   *string `json:"job_title,omitempty"`
	Department *string `json:"department,omitempty"`
	Industry   *string `json:"industry,omitempty"`
	WebsiteURL *string `json:"website_url,omitempty"`

	// Media & Assets
	AvatarURL *string `json:"avatar_url,omitempty"`
	CoverURL  *string `json:"cover_url,omitempty"`
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

		// Address Information
		validation.Field(&r.Address,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 1000).Error("Address must be at most 1000 characters"),
		),
		validation.Field(&r.City,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("City must be at most 100 characters"),
		),
		validation.Field(&r.State,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("State must be at most 100 characters"),
		),
		validation.Field(&r.Country,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Country must be at most 100 characters"),
		),
		validation.Field(&r.PostalCode,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 20).Error("Postal code must be at most 20 characters"),
		),

		// Professional Information
		validation.Field(&r.Company,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 200).Error("Company must be at most 200 characters"),
		),
		validation.Field(&r.JobTitle,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 200).Error("Job title must be at most 200 characters"),
		),
		validation.Field(&r.Department,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 200).Error("Department must be at most 200 characters"),
		),
		validation.Field(&r.Industry,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 100).Error("Industry must be at most 100 characters"),
		),
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
		validation.Field(&r.CoverURL,
			validation.NilOrNotEmpty,
			is.URL.Error("Invalid cover URL format"),
			validation.RuneLength(0, 1000).Error("Cover URL must be at most 1000 characters"),
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

	// Address Information
	Address    *string `json:"address,omitempty"`
	City       *string `json:"city,omitempty"`
	State      *string `json:"state,omitempty"`
	Country    *string `json:"country,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`

	// Professional Information
	Company    *string `json:"company,omitempty"`
	JobTitle   *string `json:"job_title,omitempty"`
	Department *string `json:"department,omitempty"`
	Industry   *string `json:"industry,omitempty"`
	WebsiteURL *string `json:"website_url,omitempty"`

	// Media & Assets
	AvatarURL *string `json:"avatar_url,omitempty"`
	CoverURL  *string `json:"cover_url,omitempty"`

	// System Fields
	LastProfileUpdate *time.Time `json:"last_profile_update,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
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

		// Address Information
		Address:    p.Address,
		City:       p.City,
		State:      p.State,
		Country:    p.Country,
		PostalCode: p.PostalCode,

		// Professional Information
		Company:    p.Company,
		JobTitle:   p.JobTitle,
		Department: p.Department,
		Industry:   p.Industry,
		WebsiteURL: p.WebsiteURL,

		// Media & Assets
		AvatarURL: p.AvatarURL,
		CoverURL:  p.CoverURL,

		// System Fields
		LastProfileUpdate: p.LastProfileUpdate,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}
