package repository

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type InviteRepository interface {
	BaseRepositoryMethods[model.Invite]
	WithTx(tx *gorm.DB) InviteRepository
	FindByToken(token string) (*model.Invite, error)
	FindAllByAuthClientID(authClientID int64) ([]model.Invite, error)
	MarkAsUsed(inviteUUID uuid.UUID) error
	RevokeByUUID(inviteUUID uuid.UUID) error
}

type inviteRepository struct {
	*BaseRepository[model.Invite]
	db *gorm.DB
}

func NewInviteRepository(db *gorm.DB) InviteRepository {
	return &inviteRepository{
		BaseRepository: NewBaseRepository[model.Invite](db, "invite_uuid", "invite_id"),
		db:             db,
	}
}

func (r *inviteRepository) WithTx(tx *gorm.DB) InviteRepository {
	return &inviteRepository{
		BaseRepository: NewBaseRepository[model.Invite](tx, "invite_uuid", "invite_id"),
		db:             tx,
	}
}

func (r *inviteRepository) FindByToken(token string) (*model.Invite, error) {
	var invite model.Invite
	err := r.db.
		Preload("Roles").
		Where("invite_token = ?", token).
		First(&invite).Error
	return &invite, err
}

func (r *inviteRepository) FindAllByAuthClientID(authClientID int64) ([]model.Invite, error) {
	var invites []model.Invite
	err := r.db.
		Where("auth_client_id = ?", authClientID).
		Find(&invites).Error
	return invites, err
}

func (r *inviteRepository) MarkAsUsed(inviteUUID uuid.UUID) error {
	return r.db.Model(&model.Invite{}).
		Where("invite_uuid = ?", inviteUUID).
		Updates(map[string]interface{}{
			"status":  "accepted",
			"used_at": gorm.Expr("now()"),
		}).Error
}

func (r *inviteRepository) RevokeByUUID(inviteUUID uuid.UUID) error {
	return r.db.Model(&model.Invite{}).
		Where("invite_uuid = ?", inviteUUID).
		Update("status", "revoked").Error
}
