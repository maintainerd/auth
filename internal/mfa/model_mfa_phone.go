package mfa

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserMFAPhone struct {
	MfaPhoneID   int64      `gorm:"column:mfa_phone_id;primaryKey"`
	MfaPhoneUUID uuid.UUID  `gorm:"column:mfa_phone_uuid;unique"`
	UserID       int64      `gorm:"column:user_id;not null"`
	Phone        string     `gorm:"column:phone;not null"`
	IsVerified   bool       `gorm:"column:is_verified;default:false"`
	VerifiedAt   *time.Time `gorm:"column:verified_at"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserMFAPhone) TableName() string { return "user_mfa_phones" }

func (p *UserMFAPhone) BeforeCreate(tx *gorm.DB) error {
	if p.MfaPhoneUUID == uuid.Nil {
		p.MfaPhoneUUID = uuid.New()
	}
	return nil
}
