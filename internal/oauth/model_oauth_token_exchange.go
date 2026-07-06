package oauth

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type OAuthTokenExchange struct {
	OAuthTokenExchangeID   int64          `gorm:"column:oauth_token_exchange_id;primaryKey"`
	OAuthTokenExchangeUUID uuid.UUID      `gorm:"column:oauth_token_exchange_uuid;unique;not null"`
	TenantID               int64          `gorm:"column:tenant_id;not null"`
	ActorClientID          int64          `gorm:"column:actor_client_id;not null"`
	SubjectTokenType       string         `gorm:"column:subject_token_type;not null"`
	RequestedTokenType     string         `gorm:"column:requested_token_type;not null"`
	SubjectUserID          *int64         `gorm:"column:subject_user_id"`
	SubjectClientID        *int64         `gorm:"column:subject_client_id"`
	IssuedJTI              *string        `gorm:"column:issued_jti"`
	ExchangeType           string         `gorm:"column:exchange_type;not null"`
	Scope                  pq.StringArray `gorm:"column:scope;type:text[];not null;default:'{}'"`
	IPAddress              *string        `gorm:"column:ip_address"`
	CreatedAt              time.Time      `gorm:"column:created_at;not null;autoCreateTime"`
}

func (OAuthTokenExchange) TableName() string {
	return "oauth_token_exchanges"
}

func (e *OAuthTokenExchange) BeforeCreate(tx *gorm.DB) (err error) {
	if e.OAuthTokenExchangeUUID == uuid.Nil {
		e.OAuthTokenExchangeUUID = uuid.New()
	}
	return
}
