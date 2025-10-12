package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Profile struct {
	ProfileID   int64     `gorm:"column:profile_id;primaryKey"`
	ProfileUUID uuid.UUID `gorm:"column:profile_uuid;unique;not null"`
	UserID      int64     `gorm:"column:user_id;not null"`

	// Basic Identity Information
	FirstName   string  `gorm:"column:first_name;not null"`
	MiddleName  *string `gorm:"column:middle_name"`
	LastName    *string `gorm:"column:last_name"`
	Suffix      *string `gorm:"column:suffix"`
	DisplayName *string `gorm:"column:display_name"`

	// Personal Information
	Birthdate *time.Time `gorm:"column:birthdate"`
	Gender    *string    `gorm:"column:gender"` // 'male', 'female', 'other', 'prefer_not_to_say'
	Bio       *string    `gorm:"column:bio"`

	// Contact Information
	Phone *string `gorm:"column:phone"`
	Email *string `gorm:"column:email"`

	// Address Information
	AddressLine1  *string `gorm:"column:address_line_1"`
	AddressLine2  *string `gorm:"column:address_line_2"`
	City          *string `gorm:"column:city"`
	StateProvince *string `gorm:"column:state_province"` // State/Province/Region
	PostalCode    *string `gorm:"column:postal_code"`    // ZIP/Postal Code
	Country       *string `gorm:"column:country"`        // ISO 3166-1 alpha-2 code (US, CA, GB, etc.)
	CountryName   *string `gorm:"column:country_name"`   // Full country name

	// Professional Information
	Company    *string `gorm:"column:company"`
	JobTitle   *string `gorm:"column:job_title"`
	Department *string `gorm:"column:department"`
	Industry   *string `gorm:"column:industry"`
	WebsiteURL *string `gorm:"column:website_url"`

	// Media & Assets
	AvatarURL   *string `gorm:"column:avatar_url"`
	AvatarS3Key *string `gorm:"column:avatar_s3_key"`
	CoverURL    *string `gorm:"column:cover_url"`
	CoverS3Key  *string `gorm:"column:cover_s3_key"`

	// System Fields
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID"`
}

func (Profile) TableName() string {
	return "profiles"
}

func (p *Profile) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ProfileUUID == uuid.Nil {
		p.ProfileUUID = uuid.New()
	}
	return
}
