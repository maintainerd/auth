package idp

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthFlowCallbackURI links an auth flow to one of its client's registered
// redirect URIs (client_uris). An auth flow may have many callback URIs; each
// references a client_uris row, so the allowlist stays owned by the client.
type AuthFlowCallbackURI struct {
	AuthFlowCallbackURIID   int64      `gorm:"column:auth_flow_callback_uri_id;primaryKey;autoIncrement" json:"auth_flow_callback_uri_id"`
	AuthFlowCallbackURIUUID uuid.UUID  `gorm:"column:auth_flow_callback_uri_uuid;type:uuid;uniqueIndex;not null" json:"auth_flow_callback_uri_uuid"`
	AuthFlowID              int64      `gorm:"column:auth_flow_id;not null" json:"auth_flow_id"`
	ClientURIID             int64      `gorm:"column:client_uri_id;not null" json:"client_uri_id"`
	AuthFlow                *AuthFlow  `gorm:"foreignKey:AuthFlowID;references:AuthFlowID" json:"auth_flow,omitempty"`
	ClientURI               *ClientURI `gorm:"foreignKey:ClientURIID;references:ClientURIID" json:"client_uri,omitempty"`
	CreatedAt               time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AuthFlowCallbackURI) TableName() string {
	return "auth_flow_callback_uris"
}

func (c *AuthFlowCallbackURI) BeforeCreate(tx *gorm.DB) error {
	if c.AuthFlowCallbackURIUUID == uuid.Nil {
		c.AuthFlowCallbackURIUUID = uuid.New()
	}
	return nil
}
