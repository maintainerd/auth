package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	UserID             int64     `gorm:"column:user_id;primaryKey"`
	UserUUID           uuid.UUID `gorm:"column:user_uuid;type:uuid;not null;unique;index:idx_users_user_uuid"`
	Username           string    `gorm:"column:username;type:varchar(255);not null;index:idx_users_username"`
	Email              string    `gorm:"column:email;type:varchar(255);index:idx_users_email"`
	Phone              string    `gorm:"column:phone;type:varchar(20);index:idx_users_phone"`
	Password           *string   `gorm:"column:password;type:text"` // nullable for external users
	IsEmailVerified    bool      `gorm:"column:is_email_verified;type:boolean;default:false"`
	IsProfileCompleted bool      `gorm:"column:is_profile_completed;type:boolean;default:false"`
	IsAccountCompleted bool      `gorm:"column:is_account_completed;type:boolean;default:false"`
	IsActive           bool      `gorm:"column:is_active;type:boolean;default:true"`
	OrganizationID     int64     `gorm:"column:organization_id;type:integer;not null;index:idx_users_organization_id"`
	AuthContainerID    int64     `gorm:"column:auth_container_id;type:integer;not null;index:idx_users_auth_container_id"`
	CreatedAt          time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

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
