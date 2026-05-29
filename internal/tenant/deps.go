package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Consumer-defined interfaces and projection types for upstream domains.
// tenant declares the shape of data it needs from the user domain so it does
// not import the user package directly (avoiding an import cycle). The user
// domain provides adapters that satisfy these interfaces; wiring injects them.

// MemberUser is tenant's projection of a user, used when listing tenant members.
type MemberUser struct {
	UserID             int64
	UserUUID           uuid.UUID
	Username           string
	Fullname           string
	Email              string
	Phone              string
	IsEmailVerified    bool
	IsPhoneVerified    bool
	IsProfileCompleted bool
	IsAccountCompleted bool
	Status             string
	Metadata           datatypes.JSON
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// UserReader is the subset of the user domain that tenant needs to resolve
// member user details.
type UserReader interface {
	FindByUUID(userUUID uuid.UUID) (*MemberUser, error)
	FindByID(userID int64) (*MemberUser, error)
}

// AccessActor exposes the tenant-relevant identity data needed for access
// control checks (see access.go). The user domain implements this on its User.
type AccessActor interface {
	AccessIdentities() []AccessIdentity
}

// AccessIdentity is one of an actor's identities with the tenant facts needed
// for access control.
type AccessIdentity struct {
	TenantID       int64
	TenantIsSystem bool
}
