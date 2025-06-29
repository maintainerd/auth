package model

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ProfileID   int64     `gorm:"primaryKey;column:profile_id"`
	ProfileUUID uuid.UUID `gorm:"type:uuid;not null;unique;column:profile_uuid;index:idx_profiles_profile_uuid"`

	UserID int64 `gorm:"not null;index:idx_profiles_user_id"`

	// Personal Information
	FirstName  *string    `gorm:"type:varchar(100);index:idx_profiles_first_name"`
	MiddleName *string    `gorm:"type:varchar(100)"`
	LastName   *string    `gorm:"type:varchar(100);index:idx_profiles_last_name"`
	Suffix     *string    `gorm:"type:varchar(50)"`
	Birthdate  *time.Time `gorm:"type:date"`
	Gender     *string    `gorm:"type:varchar(20)"`

	// Contact Information
	Phone   *string `gorm:"type:varchar(20)"`
	Email   *string `gorm:"type:varchar(255)"`
	Address *string `gorm:"type:text"`

	// Media
	AvatarURL   *string `gorm:"type:text"`
	AvatarS3Key *string `gorm:"type:text"`
	CoverURL    *string `gorm:"type:text"`
	CoverS3Key  *string `gorm:"type:text"`

	// Metadata
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime"`

	// Relation
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// TableName overrides the default table name
func (Profile) TableName() string {
	return "profiles"
}
