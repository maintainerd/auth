package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Profile struct {
	ProfileID   int64      `gorm:"column:profile_id;primaryKey"`
	ProfileUUID uuid.UUID  `gorm:"column:profile_uuid;unique;not null"`
	UserID      int64      `gorm:"column:user_id;not null"`
	FirstName   string     `gorm:"column:first_name;not null"`
	MiddleName  *string    `gorm:"column:middle_name"`
	LastName    *string    `gorm:"column:last_name"`
	Suffix      *string    `gorm:"column:suffix"`
	Birthdate   *time.Time `gorm:"column:birthdate"`
	Gender      *string    `gorm:"column:gender"` // 'male', 'female'
	Phone       *string    `gorm:"column:phone"`
	Email       *string    `gorm:"column:email"`
	Address     *string    `gorm:"column:address"`
	AvatarURL   *string    `gorm:"column:avatar_url"`
	AvatarS3Key *string    `gorm:"column:avatar_s3_key"`
	CoverURL    *string    `gorm:"column:cover_url"`
	CoverS3Key  *string    `gorm:"column:cover_s3_key"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`

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
