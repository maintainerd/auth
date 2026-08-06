package dashboard

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Service interface {
	GetSummary(ctx context.Context, tenantID int64) (*SummaryResponseDTO, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) GetSummary(ctx context.Context, tenantID int64) (*SummaryResponseDTO, error) {
	var summary SummaryResponseDTO

	if err := s.countUsers(tenantID, &summary.Users); err != nil {
		return nil, err
	}
	if err := s.countResource("services", tenantID, &summary.Services); err != nil {
		return nil, err
	}
	if err := s.countResource("clients", tenantID, &summary.Clients); err != nil {
		return nil, err
	}
	if err := s.countResource("identity_providers", tenantID, &summary.IdentityProviders); err != nil {
		return nil, err
	}
	if err := s.countResource("roles", tenantID, &summary.Roles); err != nil {
		return nil, err
	}
	if err := s.countAuthEvents(tenantID, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

func (s *service) countUsers(tenantID int64, out *UserCount) error {
	// DISTINCT: a user holding identities under several providers (the built-in
	// one plus a federated one, say) matches the join once per identity and
	// would otherwise be counted repeatedly.
	base := s.db.Table("users").
		Joins("JOIN user_identities ON users.user_id = user_identities.user_id").
		Where("user_identities.tenant_id = ?", tenantID).
		Where("users.deleted_at IS NULL").
		Distinct("users.user_id")

	if err := base.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return err
	}
	if err := base.Session(&gorm.Session{}).Where("users.status = ?", "active").Count(&out.Active).Error; err != nil {
		return err
	}
	if err := base.Session(&gorm.Session{}).Where("users.status = ?", "inactive").Count(&out.Inactive).Error; err != nil {
		return err
	}
	if err := base.Session(&gorm.Session{}).Where("users.status = ?", "suspended").Count(&out.Suspended).Error; err != nil {
		return err
	}
	if err := base.Session(&gorm.Session{}).Where("users.status = ?", "pending").Count(&out.Pending).Error; err != nil {
		return err
	}
	return nil
}

func (s *service) countResource(table string, tenantID int64, out *ResourceCount) error {
	base := s.db.Table(table).
		Where("tenant_id = ?", tenantID).
		Where("deleted_at IS NULL")

	if err := base.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return err
	}
	if err := base.Session(&gorm.Session{}).Where("status = ?", "active").Count(&out.Active).Error; err != nil {
		return err
	}
	out.Inactive = out.Total - out.Active
	return nil
}

func (s *service) countAuthEvents(tenantID int64, out *SummaryResponseDTO) error {
	since := time.Now().Add(-24 * time.Hour)

	if err := s.db.Table("auth_events").
		Where("tenant_id = ? AND event_type = ? AND created_at >= ?", tenantID, "authn_login_success", since).
		Count(&out.RecentLogins24h).Error; err != nil {
		return err
	}
	if err := s.db.Table("auth_events").
		Where("tenant_id = ? AND event_type = ? AND created_at >= ?", tenantID, "authn_login_fail", since).
		Count(&out.FailedLogins24h).Error; err != nil {
		return err
	}
	return nil
}
