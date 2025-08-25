package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type InviteRoleRepository interface {
	BaseRepositoryMethods[model.InviteRole]
}

type inviteRoleRepository struct {
	*BaseRepository[model.InviteRole]
	db *gorm.DB
}

func NewInviteRoleRepository(db *gorm.DB) InviteRoleRepository {
	return &inviteRoleRepository{
		BaseRepository: NewBaseRepository[model.InviteRole](db, "invite_role_uuid", "invite_role_id"),
		db:             db,
	}
}
