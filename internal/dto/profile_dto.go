package dto

import (
	"time"

	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/validator"
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
	return validator.ValidateStruct(&r,
		validator.Field(&r.FirstName,
			validator.Required().Error("First name is required"),
			validator.MaxLength(100).Error("First name must be at most 100 characters"),
		),
		validator.Field(&r.MiddleName,
			validator.Optional(),
			validator.MaxLength(100).Error("Middle name must be at most 100 characters"),
		),
		validator.Field(&r.LastName,
			validator.Optional(),
			validator.MaxLength(100).Error("Last name must be at most 100 characters"),
		),
		validator.Field(&r.Suffix,
			validator.Optional(),
			validator.MaxLength(50).Error("Suffix must be at most 50 characters"),
		),
		validator.Field(&r.Birthdate,
			validator.Optional(),
			validator.Date("2006-01-02").Error("Birthdate must be in YYYY-MM-DD format"),
		),
		validator.Field(&r.Gender,
			validator.Optional(),
			validator.In("male", "female", "other").Error("Gender must be male, female, or other"),
		),
		validator.Field(&r.Phone,
			validator.Optional(),
			validator.MaxLength(20).Error("Phone must be at most 20 characters"),
		),
		validator.Field(&r.Email,
			validator.Optional(),
			validator.Email().Error("Invalid email format"),
			validator.MaxLength(255).Error("Email must be at most 255 characters"),
		),
		validator.Field(&r.Address,
			validator.Optional(),
			validator.MaxLength(1000).Error("Address must be at most 1000 characters"),
		),
		validator.Field(&r.AvatarURL,
			validator.Optional(),
			validator.MaxLength(1000).Error("Avatar URL must be at most 1000 characters"),
		),
		validator.Field(&r.CoverURL,
			validator.Optional(),
			validator.MaxLength(1000).Error("Cover URL must be at most 1000 characters"),
		),
	)
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
