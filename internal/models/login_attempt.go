package models

import (
	"time"

	"github.com/google/uuid"
)

type LoginAttempt struct {
	LoginAttemptID   int64     `gorm:"primaryKey;column:login_attempt_id"`
	LoginAttemptUUID uuid.UUID `gorm:"type:uuid;not null;unique;column:login_attempt_uuid"`

	UserID    *int64  `gorm:"index:idx_login_attempts_user_id"` // nullable
	Email     *string `gorm:"type:varchar(255);index:idx_login_attempts_email"`
	IPAddress *string `gorm:"type:varchar(100)"`
	UserAgent *string `gorm:"type:text"`

	IsSuccess   bool      `gorm:"default:false"`
	AttemptedAt time.Time `gorm:"column:attempted_at;autoCreateTime"`

	// Relation (optional)
	User *User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:SET NULL"`
}

// TableName overrides the default table name
func (LoginAttempt) TableName() string {
	return "login_attempts"
}
