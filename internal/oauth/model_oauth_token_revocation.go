package oauth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OAuthTokenRevocation struct {
	OAuthTokenRevocationID   int64     `gorm:"column:oauth_token_revocation_id;primaryKey"`
	OAuthTokenRevocationUUID uuid.UUID `gorm:"column:oauth_token_revocation_uuid;unique;not null"`
	TenantID                 int64     `gorm:"column:tenant_id;not null"`
	JTI                      string    `gorm:"column:jti;not null;unique"`
	TokenType                string    `gorm:"column:token_type;not null;default:'access_token'"`
	RevokedByUserID          *int64    `gorm:"column:revoked_by_user_id"`
	RevokedByClientID        *int64    `gorm:"column:revoked_by_client_id"`
	Reason                   string    `gorm:"column:reason;not null;default:'logout'"`
	ExpiresAt                time.Time `gorm:"column:expires_at;not null"`
	CreatedAt                time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (OAuthTokenRevocation) TableName() string {
	return "oauth_token_revocations"
}

func (r *OAuthTokenRevocation) BeforeCreate(tx *gorm.DB) (err error) {
	if r.OAuthTokenRevocationUUID == uuid.Nil {
		r.OAuthTokenRevocationUUID = uuid.New()
	}
	return
}
