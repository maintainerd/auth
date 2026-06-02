package notifier

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SMSOtp struct {
	SMSOtpID       int64     `gorm:"column:sms_otp_id;primaryKey"`
	SMSOtpUUID     uuid.UUID `gorm:"column:sms_otp_uuid;unique"`
	UserID         int64     `gorm:"column:user_id"`
	Phone          string    `gorm:"column:phone"`
	OTPHash        string    `gorm:"column:otp_hash"`
	ExpiresAt      time.Time `gorm:"column:expires_at"`
	Used           bool      `gorm:"column:used;default:false"`
	FailedAttempts int       `gorm:"column:failed_attempts;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (SMSOtp) TableName() string { return "sms_otps" }

func (s *SMSOtp) BeforeCreate(tx *gorm.DB) error {
	if s.SMSOtpUUID == uuid.Nil {
		s.SMSOtpUUID = uuid.New()
	}
	return nil
}
