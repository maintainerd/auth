package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserConsent struct {
	UserConsentID   int64     `gorm:"column:user_consent_id;primaryKey"`
	UserConsentUUID uuid.UUID `gorm:"column:user_consent_uuid;unique;not null"`
	UserID          int64     `gorm:"column:user_id;not null"`
	TenantID        int64     `gorm:"column:tenant_id;not null"`
	ConsentType     string    `gorm:"column:consent_type;not null;size:50"`
	PolicyVersion   string    `gorm:"column:policy_version;not null;size:50"`
	Accepted        bool      `gorm:"column:accepted;not null"`
	IPAddress       *string   `gorm:"column:ip_address"`
	UserAgent       *string   `gorm:"column:user_agent"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (UserConsent) TableName() string {
	return "user_consents"
}

func (uc *UserConsent) BeforeCreate(tx *gorm.DB) (err error) {
	if uc.UserConsentUUID == uuid.Nil {
		uc.UserConsentUUID = uuid.New()
	}
	return
}
