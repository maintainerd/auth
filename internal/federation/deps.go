package federation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Upstream-domain projections
//
// These structs are local projections of types owned by other domains. They
// carry no json tags because they never touch the wire; the federation package
// declares them so it does not import those domains directly.
// ---------------------------------------------------------------------------

// Client projects the columns of the clients table needed to mint a workload
// token for the platform client a federation maps to.
type Client struct {
	ClientID       int64     `gorm:"column:client_id;primaryKey"`
	ClientUUID     uuid.UUID `gorm:"column:client_uuid"`
	TenantID       int64     `gorm:"column:tenant_id"`
	Name           string    `gorm:"column:name"`
	Identifier     *string   `gorm:"column:identifier"`
	Status         string    `gorm:"column:status"`
	AccessTokenTTL *int      `gorm:"column:access_token_ttl"`
	// AllowedScopes is the client's own scope allow-list, the one /oauth/token
	// enforces. The exchange intersects it with the federation's so a keyless WIF
	// token cannot carry scopes the same client would be refused directly.
	AllowedScopes pq.StringArray `gorm:"column:allowed_scopes;type:text[]"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the clients table name.
func (Client) TableName() string { return "clients" }

// ---------------------------------------------------------------------------
// Consumer interfaces
// ---------------------------------------------------------------------------

// ExchangeAuditEntry is the audit record written for every successful workload
// token exchange. The composition root adapts this to oauth.OAuthTokenExchange
// so the federation package never imports the oauth domain.
type ExchangeAuditEntry struct {
	TenantID           int64
	ActorClientID      int64
	SubjectTokenType   string
	RequestedTokenType string
	ExchangeType       string
	Scopes             []string
	IssuedJTI          *string
	IPAddress          *string
	CreatedAt          time.Time
}

// ExchangeAuditor records workload token exchange events for audit
// (oauth_token_exchanges, section 3.20). Best-effort — a recording failure
// must never block a valid exchange.
type ExchangeAuditor interface {
	RecordExchange(ctx context.Context, entry ExchangeAuditEntry) error
}
