package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Profile holds biographical/PII data for a user. Multiple profiles per user
// are supported (is_default marks the active one). Removed columns:
//   - email     → use users.email
//   - timezone  → use user_settings.timezone
//   - language  → use user_settings.locale
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
	Gender    *string    `gorm:"column:gender"`

	// Contact Information (profile-level contact, distinct from users.phone which is the login phone)
	Phone   *string `gorm:"column:phone"`
	Address *string `gorm:"column:address"`

	// Email is NOT persisted on profiles — it lives in users.email.
	// Kept as a transient field for API compatibility.
	Email *string `gorm:"-"`

	// Location Information
	City    *string `gorm:"column:city"`
	Country *string `gorm:"column:country"`

	// Timezone/Language are NOT persisted on profiles — they live in user_settings.
	// Kept as transient fields for API compatibility (Language maps to user_settings.locale).
	Timezone *string `gorm:"-"`
	Language *string `gorm:"-"`

	// Media & Assets (auth-centric)
	ProfileURL *string `gorm:"column:profile_url"`

	// Extended data
	Metadata datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`

	// Audit
	CreatedBy *int64 `gorm:"column:created_by"`
	UpdatedBy *int64 `gorm:"column:updated_by"`

	// System Fields
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`

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
