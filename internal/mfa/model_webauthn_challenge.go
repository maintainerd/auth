package mfa

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WebAuthnChallenge struct {
	WebAuthnChallengeID   int64      `gorm:"column:webauthn_challenge_id;primaryKey"`
	WebAuthnChallengeUUID uuid.UUID  `gorm:"column:webauthn_challenge_uuid;unique;not null"`
	TenantID              int64      `gorm:"column:tenant_id;not null"`
	UserID                *int64     `gorm:"column:user_id"`
	Challenge             string     `gorm:"column:challenge;not null"`
	Operation             string     `gorm:"column:operation;not null"`
	RPID                  string     `gorm:"column:rp_id;not null"`
	ExpiresAt             time.Time  `gorm:"column:expires_at;not null"`
	UsedAt                *time.Time `gorm:"column:used_at"`
	CreatedAt             time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

func (WebAuthnChallenge) TableName() string {
	return "webauthn_challenges"
}

func (c *WebAuthnChallenge) BeforeCreate(tx *gorm.DB) (err error) {
	if c.WebAuthnChallengeUUID == uuid.Nil {
		c.WebAuthnChallengeUUID = uuid.New()
	}
	return
}
