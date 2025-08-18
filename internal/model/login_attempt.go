package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoginAttempt struct {
	LoginAttemptID   int64     `gorm:"column:login_attempt_id;primaryKey"`
	LoginAttemptUUID uuid.UUID `gorm:"column:login_attempt_uuid"`
	UserID           *int64    `gorm:"column:user_id"`
	Email            *string   `gorm:"column:email"`
	IPAddress        *string   `gorm:"column:ip_address"`
	UserAgent        *string   `gorm:"column:user_agent"`
	IsSuccess        bool      `gorm:"column:is_success"`
	AttemptedAt      time.Time `gorm:"column:attempted_at"`
	AuthContainerID  int64     `gorm:"column:auth_container_id"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	User          *User          `gorm:"foreignKey:UserID;references:UserID"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID"`
}

func (LoginAttempt) TableName() string {
	return "login_attempts"
}

func (la *LoginAttempt) BeforeCreate(tx *gorm.DB) (err error) {
	if la.LoginAttemptUUID == uuid.Nil {
		la.LoginAttemptUUID = uuid.New()
	}
	return
}
