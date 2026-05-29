package authn

import (
	"context"

	"github.com/google/uuid"
)

// Consumer-defined interfaces: authn specifies the shape of data it needs
// from upstream domains. The upstream domains implement adapters that satisfy
// these interfaces. This prevents import cycles and keeps coupling explicit.

// UserReader defines the interface authn needs to read users from the user domain.
type UserReader interface {
	GetByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64) (interface{}, error)
	GetByUsername(ctx context.Context, username string, tenantID int64) (interface{}, error)
	GetByEmail(ctx context.Context, email string, tenantID int64) (interface{}, error)
}

// UserWriter defines the interface authn needs to write/update users.
type UserWriter interface {
	Create(ctx context.Context, username, fullname string, email *string, password string, status string, tenantID int64) (interface{}, error)
	Update(ctx context.Context, userUUID uuid.UUID, tenantID int64, updates map[string]interface{}) (interface{}, error)
	SetStatus(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string) error
}

// ClientReader defines the interface authn needs to read OAuth clients.
type ClientReader interface {
	GetByID(ctx context.Context, clientID uuid.UUID, tenantID int64) (interface{}, error)
	GetByClientID(ctx context.Context, clientID string, tenantID int64) (interface{}, error)
}

// IdentityProviderReader defines the interface authn needs to read identity providers.
type IdentityProviderReader interface {
	GetByID(ctx context.Context, providerID uuid.UUID, tenantID int64) (interface{}, error)
	GetByProviderKey(ctx context.Context, providerKey string, tenantID int64) (interface{}, error)
}

// TenantReader defines the interface authn needs to read tenant data.
type TenantReader interface {
	GetByID(ctx context.Context, tenantID int64) (interface{}, error)
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (interface{}, error)
}

// SecuritySettingReader defines the interface authn needs to read security settings.
type SecuritySettingReader interface {
	GetByTenantID(ctx context.Context, tenantID int64) (interface{}, error)
}

// AuthEventWriter defines the interface authn needs to write auth events.
type AuthEventWriter interface {
	Create(ctx context.Context, event interface{}) (interface{}, error)
}

// SessionManager defines the interface authn needs for session management.
type SessionManager interface {
	Create(ctx context.Context, userID int64, clientID string, metadata map[string]interface{}) (interface{}, error)
	Get(ctx context.Context, sessionID string) (interface{}, error)
	Invalidate(ctx context.Context, sessionID string) error
	RefreshToken(ctx context.Context, userID int64) (string, error)
}

// Adapter is a placeholder type for adapters that connect consumer interfaces
// to upstream domain implementations. Each consumer-interface has a corresponding
// adapter struct in the upstream domain that satisfies it.
type Adapter struct {
	// This type is not instantiated; it documents the adapter pattern used
	// for cross-domain communication.
}
