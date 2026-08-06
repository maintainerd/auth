package oauth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// SessionAuthContext is what a login actually established: when it happened and
// by which factors. It is written to user_sessions by the authentication layer
// (authn.SessionAttributes) and read back here so an OAuth token can assert the
// truth instead of a placeholder.
type SessionAuthContext struct {
	ACR      string
	AMR      []string
	AuthTime time.Time
}

// SessionAuthContextResolver reads the recorded authentication facts for a
// browser session.
//
// It is an interface with an explicit setter rather than a constructor argument
// so that call sites which have no session store — and the service's own unit
// tests — keep the documented fallback instead of gaining a database
// dependency.
type SessionAuthContextResolver interface {
	// ResolveSessionAuthContext returns the session's recorded acr/amr/auth_time.
	// A nil result with a nil error means "no such session"; callers fall back to
	// their defaults.
	ResolveSessionAuthContext(ctx context.Context, sessionUUID uuid.UUID) (*SessionAuthContext, error)
}

// userSessionAuthRepository resolves session auth context straight from
// user_sessions. It reads three columns and never writes: user_sessions is
// owned by the authentication layer.
type userSessionAuthRepository struct {
	db *gorm.DB
}

// NewUserSessionAuthContextResolver builds the gorm-backed resolver.
func NewUserSessionAuthContextResolver(db *gorm.DB) SessionAuthContextResolver {
	return &userSessionAuthRepository{db: db}
}

func (r *userSessionAuthRepository) ResolveSessionAuthContext(ctx context.Context, sessionUUID uuid.UUID) (*SessionAuthContext, error) {
	if r.db == nil || sessionUUID == uuid.Nil {
		return nil, nil
	}
	var row struct {
		ACR      string         `gorm:"column:acr"`
		AMR      pq.StringArray `gorm:"column:amr;type:text[]"`
		AuthTime time.Time      `gorm:"column:auth_time"`
	}
	err := r.db.WithContext(ctx).
		Table("user_sessions").
		Select("acr", "amr", "auth_time").
		Where("user_session_uuid = ? AND revoked_at IS NULL", sessionUUID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &SessionAuthContext{
		ACR:      row.ACR,
		AMR:      []string(row.AMR),
		AuthTime: row.AuthTime,
	}, nil
}
