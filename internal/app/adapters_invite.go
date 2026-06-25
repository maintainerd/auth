package app

import (
	"github.com/google/uuid"
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
	err := r.DB().Where("is_system = ?", true).First(&c).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &c, nil
}

type inviteAuthFlowRepo struct {
	*database.BaseRepository[invite.AuthFlow]
}

func newInviteAuthFlowRepo(db *gorm.DB) invite.AuthFlowRepository {
	return &inviteAuthFlowRepo{database.NewBaseRepository[invite.AuthFlow](db, "auth_flow_uuid", "auth_flow_id")}
}

func (r *inviteAuthFlowRepo) WithTx(tx *gorm.DB) invite.AuthFlowRepository {
	return &inviteAuthFlowRepo{r.BaseRepository.WithTx(tx)}
}

func (r *inviteAuthFlowRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, preloads ...string) (*invite.AuthFlow, error) {
	var af invite.AuthFlow
	query := r.DB().Where("auth_flow_uuid = ? AND tenant_id = ?", id, tenantID)
	for _, p := range preloads {
		query = query.Preload(p)
	}
	err := query.First(&af).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &af, nil
}

func (r *inviteAuthFlowRepo) FindByNameAndTenantID(name string, tenantID int64) (*invite.AuthFlow, error) {
	var af invite.AuthFlow
	err := r.DB().Where("name = ? AND tenant_id = ?", name, tenantID).First(&af).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &af, nil
}
