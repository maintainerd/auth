package app

import (
	"github.com/maintainerd/auth/internal/feature"
	"github.com/maintainerd/auth/internal/tenant"
)

type featureSettingReader struct {
	repo tenant.TenantSettingRepository
}

func (r featureSettingReader) FindByTenantID(tenantID int64) (*feature.Setting, error) {
	setting, err := r.repo.FindByTenantID(tenantID)
	if err != nil || setting == nil {
		return nil, err
	}
	return &feature.Setting{FeatureFlags: setting.FeatureFlags}, nil
}
