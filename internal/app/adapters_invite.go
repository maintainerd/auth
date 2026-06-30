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

type inviteRegistrationFlowRepo struct {
	*database.BaseRepository[invite.RegistrationFlow]
}

func newInviteRegistrationFlowRepo(db *gorm.DB) invite.RegistrationFlowRepository {
	return &inviteRegistrationFlowRepo{database.NewBaseRepository[invite.RegistrationFlow](db, "registration_flow_uuid", "registration_flow_id")}
}

func (r *inviteRegistrationFlowRepo) WithTx(tx *gorm.DB) invite.RegistrationFlowRepository {
	return &inviteRegistrationFlowRepo{r.BaseRepository.WithTx(tx)}
}

func (r *inviteRegistrationFlowRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, preloads ...string) (*invite.RegistrationFlow, error) {
	var flow invite.RegistrationFlow
	query := r.DB().Where("registration_flow_uuid = ? AND tenant_id = ?", id, tenantID)
	for _, p := range preloads {
		query = query.Preload(p)
	}
	err := query.First(&flow).Error
	if err != nil {
		return nil, firstOrNil(err)
	}
	return &flow, nil
}
