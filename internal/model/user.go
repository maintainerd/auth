package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	UserID             int64     `gorm:"column:user_id;primaryKey"`
	UserUUID           uuid.UUID `gorm:"column:user_uuid;unique"`
	Username           string    `gorm:"column:username"`
	Email              string    `gorm:"column:email"`
	Phone              string    `gorm:"column:phone"`
	Password           *string   `gorm:"column:password"` // nullable for external users
	IsEmailVerified    bool      `gorm:"column:is_email_verified;default:false"`
	IsProfileCompleted bool      `gorm:"column:is_profile_completed;default:false"`
	IsAccountCompleted bool      `gorm:"column:is_account_completed;default:false"`
	IsActive           bool      `gorm:"column:is_active;default:true"`
	OrganizationID     int64     `gorm:"column:organization_id"`
	AuthContainerID    int64     `gorm:"column:auth_container_id"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Organization  *Organization  `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnDelete:CASCADE"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.UserUUID == uuid.Nil {
		u.UserUUID = uuid.New()
	}
	return
}
