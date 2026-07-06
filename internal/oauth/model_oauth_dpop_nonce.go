package oauth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthDPoPNonce is a server-issued, single-use DPoP nonce (RFC 9449 §8). It is
// issued to DPoP-required clients and consumed on the next token request to make
// replay of a DPoP proof impossible within its TTL. Ephemeral — rows are deleted
// after use or after expiry by the cleanup worker.
//
// Backed by migration 083_create_oauth_dpop_nonces_table.go.
type OAuthDPoPNonce struct {
	OAuthDPoPNonceID   int64      `gorm:"column:oauth_dpop_nonce_id;primaryKey;autoIncrement"`
	OAuthDPoPNonceUUID uuid.UUID  `gorm:"column:oauth_dpop_nonce_uuid;type:uuid;uniqueIndex;not null"`
	TenantID           int64      `gorm:"column:tenant_id;not null"`
	ClientID           int64      `gorm:"column:client_id;not null"`
	Nonce              string     `gorm:"column:nonce;type:varchar(512);not null;uniqueIndex"`
	UsedAt             *time.Time `gorm:"column:used_at"`
	ExpiresAt          time.Time  `gorm:"column:expires_at;not null"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName returns the database table name for OAuthDPoPNonce.
func (OAuthDPoPNonce) TableName() string {
	return "oauth_dpop_nonces"
}

// BeforeCreate assigns a UUID before insert when one has not been set.
func (n *OAuthDPoPNonce) BeforeCreate(tx *gorm.DB) error {
	if n.OAuthDPoPNonceUUID == uuid.Nil {
		n.OAuthDPoPNonceUUID = uuid.New()
	}
	return nil
}
