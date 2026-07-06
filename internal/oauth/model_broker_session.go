package oauth

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// OAuthBrokerSession correlates the two OAuth2 legs of a brokered login: the
// downstream app's request to /oauth/authorize (OAuth #1) and maintainerd's own
// request to the upstream identity provider (OAuth #2). It stores the original
// app request so it can be resumed after the provider callback, plus the
// per-attempt state and PKCE verifier used against the provider. Rows are
// single-use (ConsumedAt) and short-lived (ExpiresAt).
type OAuthBrokerSession struct {
	OAuthBrokerSessionID       int64     `gorm:"column:oauth_broker_session_id;primaryKey;autoIncrement"`
	OAuthBrokerSessionUUID     uuid.UUID `gorm:"column:oauth_broker_session_uuid;type:uuid;uniqueIndex;not null"`
	TenantID                   int64     `gorm:"column:tenant_id;not null"`
	ClientID                   int64     `gorm:"column:client_id;not null"`
	IdentityProviderID         int64     `gorm:"column:identity_provider_id;not null"`
	IdentityProviderIdentifier string    `gorm:"column:identity_provider_identifier;not null"`

	// Original OAuth #1 (app → maintainerd) request, resumed after OAuth #2.
	AppRedirectURI         string         `gorm:"column:app_redirect_uri;not null"`
	AppState               *string        `gorm:"column:app_state"`
	AppScope               pq.StringArray `gorm:"column:app_scope;type:text[]"`
	AppNonce               *string        `gorm:"column:app_nonce"`
	AppCodeChallenge       *string        `gorm:"column:app_code_challenge"`
	AppCodeChallengeMethod *string        `gorm:"column:app_code_challenge_method"`

	// OAuth #2 (maintainerd → provider) correlation.
	IdpState        string  `gorm:"column:idp_state;uniqueIndex;not null"`
	IdpPKCEVerifier string  `gorm:"column:idp_pkce_verifier;not null"`
	IdpNonce        *string `gorm:"column:idp_nonce"`

	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime;not null"`
}

func (OAuthBrokerSession) TableName() string {
	return "oauth_broker_sessions"
}

func (o *OAuthBrokerSession) BeforeCreate(_ *gorm.DB) error {
	if o.OAuthBrokerSessionUUID == uuid.Nil {
		o.OAuthBrokerSessionUUID = uuid.New()
	}
	return nil
}

// IsExpired reports whether the broker session has passed its expiry.
func (o *OAuthBrokerSession) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// IsConsumed reports whether the broker session has already been used.
func (o *OAuthBrokerSession) IsConsumed() bool {
	return o.ConsumedAt != nil
}
