package oauth

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type OAuthAuthorizeRequest struct {
	OAuthAuthorizeRequestID   int64          `gorm:"column:oauth_authorize_request_id;primaryKey;autoIncrement"`
	OAuthAuthorizeRequestUUID uuid.UUID      `gorm:"column:oauth_authorize_request_uuid;type:uuid;uniqueIndex;not null"`
	ClientID                  int64          `gorm:"column:client_id;not null"`
	TenantID                  int64          `gorm:"column:tenant_id;not null"`
	RedirectURI               string         `gorm:"column:redirect_uri;not null"`
	Scope                     pq.StringArray `gorm:"column:scope;type:text[]"`
	State                     *string        `gorm:"column:state"`
	Nonce                     *string        `gorm:"column:nonce"`
	ResponseType              string         `gorm:"column:response_type;not null"`
	CodeChallenge             *string        `gorm:"column:code_challenge"`
	CodeChallengeMethod       *string        `gorm:"column:code_challenge_method"`
	ScreenHint                *string        `gorm:"column:screen_hint"`
	RegistrationFlowID        *int64         `gorm:"column:registration_flow_id"`
	Status                    string         `gorm:"column:status;not null;default:pending"`
	ExpiresAt                 time.Time      `gorm:"column:expires_at;not null"`
	ConsumedAt                *time.Time     `gorm:"column:consumed_at"`
	CreatedAt                 time.Time      `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt                 time.Time      `gorm:"column:updated_at;autoUpdateTime;not null"`
	DeletedAt                 gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (OAuthAuthorizeRequest) TableName() string {
	return "oauth_authorize_requests"
}

func (r *OAuthAuthorizeRequest) BeforeCreate(_ *gorm.DB) error {
	if r.OAuthAuthorizeRequestUUID == uuid.Nil {
		r.OAuthAuthorizeRequestUUID = uuid.New()
	}
	return nil
}

func (r *OAuthAuthorizeRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

func (r *OAuthAuthorizeRequest) IsConsumed() bool {
	return r.ConsumedAt != nil
}
