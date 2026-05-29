package app

import (
	"github.com/maintainerd/auth/internal/invite"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type inviteClientRepo struct {
	*database.BaseRepository[invite.Client]
}

func newInviteClientRepo(db *gorm.DB) invite.ClientRepository {
	return &inviteClientRepo{database.NewBaseRepository[invite.Client](db, "client_uuid", "client_id")}
}

func (r *inviteClientRepo) WithTx(tx *gorm.DB) invite.ClientRepository {
	return &inviteClientRepo{r.BaseRepository.WithTx(tx)}
}

func (r *inviteClientRepo) FindSystem() (*invite.Client, error) {
	var c invite.Client
	err := r.DB().Preload("IdentityProvider").Where("is_system = ?", true).First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

type inviteRoleRepo struct {
	*database.BaseRepository[invite.Role]
}

func newInviteRoleRepo(db *gorm.DB) invite.RoleRepository {
	return &inviteRoleRepo{database.NewBaseRepository[invite.Role](db, "role_uuid", "role_id")}
}

func (r *inviteRoleRepo) WithTx(tx *gorm.DB) invite.RoleRepository {
	return &inviteRoleRepo{r.BaseRepository.WithTx(tx)}
}
