package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CIBA request status constants.
const (
	CIBAStatusPending  = "pending"
	CIBAStatusApproved = "approved"
	CIBAStatusDenied   = "denied"
	CIBAStatusExpired  = "expired"
)

// OAuthCIBARequest represents a Client-Initiated Backchannel Authentication
// request. The client initiates auth for a user identified by a hint; the
// server notifies the user out-of-band (email); the client polls /oauth/token
// until the user approves or the request expires (poll mode).
type OAuthCIBARequest struct {
	OAuthCIBARRequestID   int64     `gorm:"column:oauth_ciba_request_id;primaryKey;autoIncrement"`
	OAuthCIBARequestUUID  uuid.UUID `gorm:"column:oauth_ciba_request_uuid;type:uuid;uniqueIndex;not null"`
	AuthReqIDHash         string    `gorm:"column:auth_req_id_hash;uniqueIndex;not null"`
	ClientID              int64     `gorm:"column:client_id;not null"`
	TenantID              int64     `gorm:"column:tenant_id;not null"`
	UserID                *int64    `gorm:"column:user_id"`
	Scope                 string    `gorm:"column:scope;not null;default:''"`
	BindingMessage        *string   `gorm:"column:binding_message"`
	Status                string    `gorm:"column:status;not null;default:'pending'"`
	Interval              int       `gorm:"column:interval;not null;default:5"`
	LastPollAt            *time.Time `gorm:"column:last_poll_at"`
	NotificationSentAt    *time.Time `gorm:"column:notification_sent_at"`
	ExpiresAt             time.Time  `gorm:"column:expires_at;not null"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime;not null"`

	// Relationships
	Client *Client `gorm:"foreignKey:ClientID;references:ClientID"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID"`
	User   *User   `gorm:"foreignKey:UserID;references:UserID"`
}

func (OAuthCIBARequest) TableName() string {
	return "oauth_ciba_requests"
}

func (o *OAuthCIBARequest) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthCIBARequestUUID == uuid.Nil {
		o.OAuthCIBARequestUUID = uuid.New()
	}
	return nil
}

func (o *OAuthCIBARequest) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}
