package mfa

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserMFAEmail struct {
	MFAEmailID   int64      `gorm:"column:mfa_email_id;primaryKey"`
	MFAEmailUUID uuid.UUID  `gorm:"column:mfa_email_uuid;unique"`
	UserID       int64      `gorm:"column:user_id;not null"`
	Email        string     `gorm:"column:email;not null"`
	IsVerified   bool       `gorm:"column:is_verified;default:false"`
	VerifiedAt   *time.Time `gorm:"column:verified_at"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserMFAEmail) TableName() string { return "user_mfa_emails" }

func (e *UserMFAEmail) BeforeCreate(tx *gorm.DB) error {
	if e.MFAEmailUUID == uuid.Nil {
		e.MFAEmailUUID = uuid.New()
	}
	return nil
}
