package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
	Bio         *string `gorm:"column:bio"`

	// Profile Flags
	IsDefault bool `gorm:"column:is_default;default:false"`

	// Personal Information
	Birthdate *time.Time `gorm:"column:birthdate"`
	Gender    *string    `gorm:"column:gender"` // 'male', 'female', 'other', 'prefer_not_to_say'

	// Contact Information
	Phone   *string `gorm:"column:phone"`
	Email   *string `gorm:"column:email"`
	Address *string `gorm:"column:address"`

	// Location Information
	City    *string `gorm:"column:city"`    // Current city
	Country *string `gorm:"column:country"` // ISO 3166-1 alpha-2 code (US, PH, etc.)

	// Preference
	Timezone *string `gorm:"column:timezone"` // User timezone (e.g., America/New_York, Europe/London)
	Language *string `gorm:"column:language"` // ISO 639-1 language code (e.g., en, es, fr)

	// Media & Assets (auth-centric)
	ProfileURL *string `gorm:"column:profile_url"` // User profile picture

	// Extended data
	Metadata datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`

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
