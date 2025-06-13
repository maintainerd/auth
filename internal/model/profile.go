package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Profile struct {
	ProfileID   int64      `gorm:"column:profile_id;primaryKey"`
	ProfileUUID uuid.UUID  `gorm:"column:profile_uuid;type:uuid;not null;unique;index:idx_profiles_profile_uuid"`
	UserID      int64      `gorm:"column:user_id;type:integer;not null;index:idx_profiles_user_id"`
	FirstName   string     `gorm:"column:first_name;type:varchar(100);not null;index:idx_profiles_first_name"`
	MiddleName  *string    `gorm:"column:middle_name;type:varchar(100)"`
	LastName    *string    `gorm:"column:last_name;type:varchar(100);index:idx_profiles_last_name"`
	Suffix      *string    `gorm:"column:suffix;type:varchar(50)"`
	Birthdate   *time.Time `gorm:"column:birthdate;type:date"`
	Gender      *string    `gorm:"column:gender;type:varchar(10)"`
	Phone       *string    `gorm:"column:phone;type:varchar(20)"`
	Email       *string    `gorm:"column:email;type:varchar(255)"`
	Address     *string    `gorm:"column:address;type:text"`
	AvatarURL   *string    `gorm:"column:avatar_url;type:text"`
	AvatarS3Key *string    `gorm:"column:avatar_s3_key;type:text"`
	CoverURL    *string    `gorm:"column:cover_url;type:text"`
	CoverS3Key  *string    `gorm:"column:cover_s3_key;type:text"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
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
