package notifier

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserOTP struct {
	UserOTPID      int64     `gorm:"column:user_otp_id;primaryKey"`
	UserOTPUUID    uuid.UUID `gorm:"column:user_otp_uuid;unique"`
	UserID         int64     `gorm:"column:user_id"`
	Channel        string    `gorm:"column:channel"`
	Recipient      string    `gorm:"column:recipient"`
	OTPHash        string    `gorm:"column:otp_hash"`
	ExpiresAt      time.Time `gorm:"column:expires_at"`
	Used           bool      `gorm:"column:used;default:false"`
	FailedAttempts int       `gorm:"column:failed_attempts;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (UserOTP) TableName() string { return "user_otps" }

func (u *UserOTP) BeforeCreate(tx *gorm.DB) error {
	if u.UserOTPUUID == uuid.Nil {
		u.UserOTPUUID = uuid.New()
	}
	return nil
}
