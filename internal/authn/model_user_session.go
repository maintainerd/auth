package authn

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UserSession struct {
	UserSessionID      int64          `gorm:"column:user_session_id;primaryKey"`
	UserSessionUUID    uuid.UUID      `gorm:"column:user_session_uuid;unique;not null"`
	UserID             int64          `gorm:"column:user_id;not null"`
	TenantID           int64          `gorm:"column:tenant_id;not null"`
	ClientID           *int64         `gorm:"column:client_id"`
	IdentityProviderID *int64         `gorm:"column:identity_provider_id"`
	AuthTime           time.Time      `gorm:"column:auth_time;not null"`
	IPAddress          *string        `gorm:"column:ip_address"`
	UserAgent          *string        `gorm:"column:user_agent"`
	AMR                pq.StringArray `gorm:"column:amr;type:text[];not null;default:'{}'"`
	ACR                string         `gorm:"column:acr;not null;default:'1'"`
	IDPSessionID       *string        `gorm:"column:idp_session_id"`
	IdleTimeoutSeconds int            `gorm:"column:idle_timeout_seconds;not null;default:1800"`
	LastActiveAt       time.Time      `gorm:"column:last_active_at;not null"`
	ExpiresAt          time.Time      `gorm:"column:expires_at;not null"`
	RevokedAt          *time.Time     `gorm:"column:revoked_at"`
	RevokedReason      *string        `gorm:"column:revoked_reason"`
	CreatedAt          time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
}

func (UserSession) TableName() string {
	return "user_sessions"
}

func (s *UserSession) BeforeCreate(tx *gorm.DB) (err error) {
	if s.UserSessionUUID == uuid.Nil {
		s.UserSessionUUID = uuid.New()
	}
	return
}

func (s *UserSession) IsRevoked() bool {
	return s.RevokedAt != nil
}

func (s *UserSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
