package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoginAttempt struct {
	LoginAttemptID   int64     `gorm:"column:login_attempt_id;primaryKey"`
	LoginAttemptUUID uuid.UUID `gorm:"column:login_attempt_uuid;type:uuid;not null;unique"`
	UserID           *int64    `gorm:"column:user_id;type:integer;index:idx_login_attempts_user_id"`
	Email            *string   `gorm:"column:email;type:varchar(255);index:idx_login_attempts_email"`
	IPAddress        *string   `gorm:"column:ip_address;type:varchar(100)"`
	UserAgent        *string   `gorm:"column:user_agent;type:text"`
	IsSuccess        bool      `gorm:"column:is_success;type:boolean;default:false"`
	AttemptedAt      time.Time `gorm:"column:attempted_at;type:timestamptz;default:now()"`
	AuthContainerID  int64     `gorm:"column:auth_container_id;type:integer;not null;index:idx_login_attempts_auth_container_id"`
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime"`

	// Relationships
	User          *User          `gorm:"foreignKey:UserID;references:UserID"`
	AuthContainer *AuthContainer `gorm:"foreignKey:AuthContainerID;references:AuthContainerID;constraint:OnDelete:CASCADE"`
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
