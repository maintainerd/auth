package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UserID             int64      `gorm:"primaryKey;column:user_id"`
	UserUUID           uuid.UUID  `gorm:"type:uuid;not null;unique;column:user_uuid;index:idx_users_user_uuid"`
	Username           string     `gorm:"type:varchar(255);not null;unique;index:idx_users_username"`
	Email              string     `gorm:"type:varchar(255);not null;unique;index:idx_users_email"`
	Password           *string    `gorm:"type:text"`
	IsEmailVerified    bool       `gorm:"default:false"`
	IsProfileCompleted bool       `gorm:"default:false"`
	IsAccountCompleted bool       `gorm:"default:false"`
	IsActive           bool       `gorm:"default:true"`
	CreatedAt          time.Time  `gorm:"autoCreateTime"`
	UpdatedAt          *time.Time `gorm:"autoUpdateTime"`
}

// TableName overrides the default table name
func (User) TableName() string {
	return "users"
}
