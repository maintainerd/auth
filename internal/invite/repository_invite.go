package invite

import (
	"errors"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/shared"
	"gorm.io/gorm"
)

type InviteRepository interface {
	BaseRepositoryMethods[Invite]
	WithTx(tx *gorm.DB) InviteRepository
	FindByUUIDAndTenantID(inviteUUID uuid.UUID, tenantID int64, preloads ...string) (*Invite, error)
	FindByToken(token string) (*Invite, error)
	FindAllByClientID(clientID int64) ([]Invite, error)
	FindAllByTenantID(tenantID int64) ([]Invite, error)
	MarkAsUsed(inviteUUID uuid.UUID) error
	RevokeByUUID(inviteUUID uuid.UUID) error
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
	var invite Invite
	err := r.DB().
		Preload("Roles").
		Where("invite_token = ?", token).
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
		Find(&invites).Error
	return invites, err
}

func (r *inviteRepository) MarkAsUsed(inviteUUID uuid.UUID) error {
	return r.DB().Model(&Invite{}).
		Where("invite_uuid = ?", inviteUUID).
		Updates(map[string]any{
			"status":  shared.StatusAccepted,
			"used_at": gorm.Expr("now()"),
		}).Error
}

func (r *inviteRepository) RevokeByUUID(inviteUUID uuid.UUID) error {
	return r.DB().Model(&Invite{}).
		Where("invite_uuid = ?", inviteUUID).
		Update("status", shared.StatusRevoked).Error
}
