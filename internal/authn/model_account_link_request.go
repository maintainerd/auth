package authn

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AccountLinkRequest is a short-lived pending state created when a social login
// presents an email that matches an existing local account. The user must
// confirm (re-authenticate as the existing account) before the external
// identity is attached, guarding against account takeover via provider email.
//
// Backed by migration 081_create_account_link_requests_table.go.
type AccountLinkRequest struct {
	AccountLinkRequestID   int64          `gorm:"column:account_link_request_id;primaryKey;autoIncrement"`
	AccountLinkRequestUUID uuid.UUID      `gorm:"column:account_link_request_uuid;type:uuid;uniqueIndex;not null"`
	TenantID               int64          `gorm:"column:tenant_id;not null"`
	ExistingUserID         int64          `gorm:"column:existing_user_id;not null"`
	ProviderName           string         `gorm:"column:provider_name;type:varchar(100);not null"`
	ProviderSubject        string         `gorm:"column:provider_subject;type:varchar(512);not null"`
	ProviderEmail          *string        `gorm:"column:provider_email;type:varchar(255)"`
	ProviderClaims         datatypes.JSON `gorm:"column:provider_claims;type:jsonb;not null;default:'{}'"`
	Status                 string         `gorm:"column:status;type:varchar(20);not null;default:'pending'"`
	ConfirmationToken      string         `gorm:"column:confirmation_token;type:varchar(255);not null;uniqueIndex"`
	IPAddress              *string        `gorm:"column:ip_address"`
	ExpiresAt              time.Time      `gorm:"column:expires_at;not null"`
	ConfirmedAt            *time.Time     `gorm:"column:confirmed_at"`
	RejectedAt             *time.Time     `gorm:"column:rejected_at"`
	CreatedAt              time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
}

// TableName returns the database table name for AccountLinkRequest.
func (AccountLinkRequest) TableName() string {
	return "account_link_requests"
}

// BeforeCreate assigns a UUID before insert when one has not been set.
func (a *AccountLinkRequest) BeforeCreate(tx *gorm.DB) error {
	if a.AccountLinkRequestUUID == uuid.Nil {
		a.AccountLinkRequestUUID = uuid.New()
	}
	return nil
}

// IsExpired reports whether the request's TTL has passed.
func (a *AccountLinkRequest) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}
