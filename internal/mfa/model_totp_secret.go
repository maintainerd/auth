package mfa

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserTOTPSecret stores the TOTP secret for a user.
// Only one active secret per user is allowed (uq_user_totp_secrets_user_id).
type UserTOTPSecret struct {
	TOTPSecretID   int64      `gorm:"column:totp_secret_id;primaryKey"`
	TOTPSecretUUID uuid.UUID  `gorm:"column:totp_secret_uuid"`
	UserID         int64      `gorm:"column:user_id;not null"`
	Secret         string     `gorm:"column:secret;not null"`
	IsEnabled      bool       `gorm:"column:is_enabled;default:false"`
	EnrolledAt     *time.Time `gorm:"column:enrolled_at"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at"`
	LastUsedStep   *int64     `gorm:"column:last_used_step"`
	Digits         int        `gorm:"column:digits;default:6"`
	Period         int        `gorm:"column:period;default:30"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserTOTPSecret) TableName() string { return "user_totp_secrets" }

func (s *UserTOTPSecret) BeforeCreate(tx *gorm.DB) error {
	if s.TOTPSecretUUID == uuid.Nil {
		s.TOTPSecretUUID = uuid.New()
	}
	return nil
}
