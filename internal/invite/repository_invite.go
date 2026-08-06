package invite

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// hashInviteToken derives the value stored in invites.invite_token from the raw
// token that only ever exists in the invitee's email link.
//
// SHA-256 rather than bcrypt: the token is 32 bytes of CSPRNG output, so there
// is nothing to brute-force and no need for a slow KDF, and the digest has to be
// deterministic to remain a unique-indexed equality lookup. This mirrors how
// every other bearer token in the system (authorization codes, refresh tokens,
// OTPs, trusted-device tokens) is persisted.
//
// The input is trimmed because the raw token travels through a URL query
// parameter and the registration validator sanitises it before handing it over;
// hashing an untrimmed copy would silently miss the row.
func hashInviteToken(rawToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

var ErrInviteAlreadyUsed = errors.New("invite has already been used")

// ErrInviteNotPending is returned when a state-changing write finds the invite
// in a settled state (accepted, revoked, or expired) rather than pending.
var ErrInviteNotPending = errors.New("invite is not pending")

// ErrInviteNotResendable is returned when a resend targets an invite that has
// already been settled by the invitee or an admin.
var ErrInviteNotResendable = errors.New("invite is not resendable")

// statusExpired is the Invite.Status value written when a sweeper ages out a
// pending invite. shared has no constant for it because only invites use it.
const statusExpired = "expired"

// resendableStatuses are the Invite.Status values a resend may act on. An invite
// that is pending — including one whose expires_at has passed, which is the
// whole point of resending — and one already marked expired may be re-issued;
// accepted and revoked are terminal. Resurrecting either would mint a live
// token for an invite the invitee already consumed or an admin withdrew.
var resendableStatuses = []string{shared.StatusPending, statusExpired}

type InviteRepository interface {
	BaseRepositoryMethods[Invite]
	FindAll(preloads ...string) ([]Invite, error)
	FindByUUID(uuid any, preloads ...string) (*Invite, error)
	FindByUUIDs(uuids []string, preloads ...string) ([]Invite, error)
	FindByID(id any, preloads ...string) (*Invite, error)
	UpdateByUUID(uuid any, updatedData any) (*Invite, error)
	UpdateByID(id any, updatedData any) (*Invite, error)
	DeleteByUUID(uuid any) error
	DeleteByID(id any) error
	Paginate(conditions map[string]any, page int, limit int, preloads ...string) (*PaginationResult[Invite], error)
	WithTx(tx *gorm.DB) InviteRepository
	FindByUUIDAndTenantID(inviteUUID uuid.UUID, tenantID int64, preloads ...string) (*Invite, error)
	// FindByToken and FindByTokenForUpdate take the RAW token from the invite
	// link and hash it internally — the repository is the single boundary where a
	// raw invite token is converted to its stored digest, so no caller ever has to
	// know the storage form.
	FindByToken(token string) (*Invite, error)
	FindByTokenForUpdate(token string) (*Invite, error)
	FindAllByClientID(clientID int64) ([]Invite, error)
	FindAllByTenantID(tenantID int64) ([]Invite, error)
	MarkAsUsed(inviteUUID uuid.UUID) error
	RevokeByUUID(inviteUUID uuid.UUID) error
	ResetForResend(inviteUUID uuid.UUID, newToken string, newExpiry time.Time) error
	DeleteExpired(before time.Time) (int64, error)
}

type inviteRepository struct {
	*BaseRepository[Invite]
}

func NewInviteRepository(db *gorm.DB) InviteRepository {
	return &inviteRepository{
		BaseRepository: database.NewBaseRepository[Invite](db, "invite_uuid", "invite_id"),
	}
}

func (r *inviteRepository) WithTx(tx *gorm.DB) InviteRepository {
	return &inviteRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *inviteRepository) FindByUUIDAndTenantID(inviteUUID uuid.UUID, tenantID int64, preloads ...string) (*Invite, error) {
	var invite Invite
	query := r.DB().Where("invite_uuid = ? AND tenant_id = ?", inviteUUID, tenantID)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

func (r *inviteRepository) FindByToken(token string) (*Invite, error) {
	// An empty token must never reach the query: it would hash to a fixed digest
	// and turn a caller that simply forgot the parameter into a lookup that could
	// match a row. Fail closed as "not found".
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	var invite Invite
	err := r.DB().
		Where("invite_token = ?", hashInviteToken(token)).
		First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

func (r *inviteRepository) FindByTokenForUpdate(token string) (*Invite, error) {
	// Same fail-closed guard as FindByToken: registration consumes the invite
	// through this path, so a blank token must not be able to select a row.
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	var invite Invite
	err := r.DB().
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("invite_token = ?", hashInviteToken(token)).
		First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

func (r *inviteRepository) FindAllByClientID(clientID int64) ([]Invite, error) {
	var invites []Invite
	err := r.DB().
		Where("client_id = ?", clientID).
		Find(&invites).Error
	return invites, err
}

func (r *inviteRepository) FindAllByTenantID(tenantID int64) ([]Invite, error) {
	var invites []Invite
	err := r.DB().
		Where("tenant_id = ?", tenantID).
		Preload("RegistrationFlow").
		Find(&invites).Error
	return invites, err
}

func (r *inviteRepository) MarkAsUsed(inviteUUID uuid.UUID) error {
	result := r.DB().Model(&Invite{}).
		Where("invite_uuid = ? AND status = ?", inviteUUID, shared.StatusPending).
		Updates(map[string]any{
			"status":  shared.StatusAccepted,
			"used_at": gorm.Expr("now()"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInviteAlreadyUsed
	}
	return nil
}

// RevokeByUUID revokes an invite that is still pending. The status predicate is
// in the WHERE clause, not just in the caller: without it an already-accepted
// invite flipped to 'revoked' while keeping its used_at timestamp, leaving an
// audit trail that claimed the invite was both used and never honoured.
func (r *inviteRepository) RevokeByUUID(inviteUUID uuid.UUID) error {
	result := r.DB().Model(&Invite{}).
		Where("invite_uuid = ? AND status = ?", inviteUUID, shared.StatusPending).
		Update("status", shared.StatusRevoked)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInviteNotPending
	}
	return nil
}

// ResetForResend issues a fresh token and expiry for an invite that has not been
// settled. newToken is the RAW token that goes into the email; only its digest is
// written, matching the lookup path.
//
// The status predicate makes the check atomic with the write: a bare
// invite_uuid filter resurrected revoked and accepted invites — minting a new
// emailed token for an invite an admin had already withdrawn, and clearing the
// used_at that recorded the acceptance.
func (r *inviteRepository) ResetForResend(inviteUUID uuid.UUID, newToken string, newExpiry time.Time) error {
	result := r.DB().Model(&Invite{}).
		Where("invite_uuid = ? AND status IN ?", inviteUUID, resendableStatuses).
		Updates(map[string]any{
			"invite_token": hashInviteToken(newToken),
			"status":       shared.StatusPending,
			"expires_at":   newExpiry,
			"used_at":      nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInviteNotResendable
	}
	return nil
}

func (r *inviteRepository) DeleteExpired(before time.Time) (int64, error) {
	var total int64
	for {
		result := r.DB().
			Where("status = 'expired' OR expires_at < ?", before).
			Limit(10000).
			Delete(&Invite{})
		if result.Error != nil {
			return total, result.Error
		}
		if result.RowsAffected == 0 {
			break
		}
		total += result.RowsAffected
	}
	return total, nil
}
