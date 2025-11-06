package dto

import (
	"encoding/json"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

type ProfileRequest struct {
	// Basic Identity Information
	FirstName   string  `json:"first_name"`
	MiddleName  *string `json:"middle_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	Suffix      *string `json:"suffix,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`

	// Personal Information
	Birthdate *string `json:"birthdate,omitempty"`
	Gender    *string `json:"gender,omitempty"`

	// Contact Information
	Phone   *string `json:"phone,omitempty"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`

	// Location Information
	City    *string `json:"city,omitempty"`    // Current city
	Country *string `json:"country,omitempty"` // ISO 3166-1 alpha-2 code

	// Preference
	Timezone *string `json:"timezone,omitempty"` // User timezone
	Language *string `json:"language,omitempty"` // ISO 639-1 language code

	// Media & Assets (auth-centric)
	ProfileURL *string `json:"profile_url,omitempty"` // User profile picture

	// Extended data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
		validation.Field(&r.Address,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 500).Error("Address must be at most 500 characters"),
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

		// Preference
		validation.Field(&r.Timezone,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 50).Error("Timezone must be at most 50 characters"),
		),
		validation.Field(&r.Language,
			validation.NilOrNotEmpty,
			validation.RuneLength(0, 10).Error("Language must be at most 10 characters"),
		),

		// Media & Assets
		validation.Field(&r.ProfileURL,
			validation.NilOrNotEmpty,
			is.URL.Error("Invalid profile URL format"),
			validation.RuneLength(0, 1000).Error("Profile URL must be at most 1000 characters"),
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
	Bio         *string `json:"bio,omitempty"`

	// Personal Information
	Birthdate *time.Time `json:"birthdate,omitempty"`
	Gender    *string    `json:"gender,omitempty"`

	// Contact Information
	Phone   *string `json:"phone,omitempty"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`

	// Location Information
	City    *string `json:"city,omitempty"`    // Current city
	Country *string `json:"country,omitempty"` // ISO 3166-1 alpha-2 code

	// Preference
	Timezone *string `json:"timezone,omitempty"` // User timezone
	Language *string `json:"language,omitempty"` // ISO 639-1 language code

	// Media & Assets (auth-centric)
	ProfileURL *string `json:"profile_url,omitempty"` // User profile picture

	// Extended data
	Metadata map[string]interface{} `json:"metadata,omitempty"`

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
		Bio:         p.Bio,

		// Personal Information
		Birthdate: p.Birthdate,
		Gender:    p.Gender,

		// Contact Information
		Phone:   p.Phone,
		Email:   p.Email,
		Address: p.Address,

		// Location Information
		City:    p.City,
		Country: p.Country,

		// Preference
		Timezone: p.Timezone,
		Language: p.Language,

		// Media & Assets (auth-centric)
		ProfileURL: p.ProfileURL,

		// Extended data
		Metadata: convertJSONBToMap(p.Metadata),

		// System Fields
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// Helper function to convert JSONB to map
func convertJSONBToMap(jsonb datatypes.JSON) map[string]interface{} {
	if len(jsonb) == 0 {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonb, &result); err != nil {
		return nil
	}
	return result
}
