package oauth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthPARRequest stores a Pushed Authorization Request (RFC 9126). The client
// POSTs its full authorization request here first and receives back a
// request_uri. That URI is then passed to the /oauth/authorize endpoint instead
// of the individual parameters, reducing URI length and enabling confidential
// transmission of request details.
type OAuthPARRequest struct {
	OAuthPARRequestID   int64     `gorm:"column:oauth_par_request_id;primaryKey;autoIncrement"`
	OAuthPARRequestUUID uuid.UUID `gorm:"column:oauth_par_request_uuid;type:uuid;uniqueIndex;not null"`
	RequestURIHash      string    `gorm:"column:request_uri_hash;uniqueIndex;not null"`
	ClientID            int64     `gorm:"column:client_id;not null"`
	TenantID            int64     `gorm:"column:tenant_id;not null"`
	ResponseType        string    `gorm:"column:response_type;not null;default:'code'"`
	RedirectURI         string    `gorm:"column:redirect_uri;not null"`
	Scope               string    `gorm:"column:scope;not null;default:''"`
	State               *string   `gorm:"column:state"`
	Nonce               *string   `gorm:"column:nonce"`
	CodeChallenge       string    `gorm:"column:code_challenge;not null"`
	CodeChallengeMethod string    `gorm:"column:code_challenge_method;not null;default:'S256'"`
	IsUsed              bool      `gorm:"column:is_used;not null;default:false"`
	ExpiresAt           time.Time `gorm:"column:expires_at;not null"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
}

func (OAuthPARRequest) TableName() string {
	return "oauth_par_requests"
}

func (o *OAuthPARRequest) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthPARRequestUUID == uuid.Nil {
		o.OAuthPARRequestUUID = uuid.New()
	}
	return nil
}

func (o *OAuthPARRequest) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}
