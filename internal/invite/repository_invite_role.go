package invite

import (
	"gorm.io/gorm"
)

type InviteRoleRepository interface {
	BaseRepositoryMethods[InviteRole]
}

type inviteRoleRepository struct {
	*BaseRepository[InviteRole]
}

func NewInviteRoleRepository(db *gorm.DB) InviteRoleRepository {
	return &inviteRoleRepository{
		BaseRepository: NewBaseRepository[InviteRole](db, "invite_role_uuid", "invite_role_id"),
	}
}
