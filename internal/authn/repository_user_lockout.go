package authn

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type userLockoutRepository struct {
	db *gorm.DB
}

func NewUserLockoutRepository(db *gorm.DB) UserLockoutRepository {
	return &userLockoutRepository{db: db}
}

func (r *userLockoutRepository) WithTx(tx *gorm.DB) UserLockoutRepository {
	return &userLockoutRepository{db: tx}
}

func (r *userLockoutRepository) IsLocked(ctx context.Context, tenantID int64, identifier string, maxAttempts int, lockDuration time.Duration) (bool, error) {
	var lockout UserLockout
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND identifier = ?", tenantID, identifier).
		First(&lockout).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if lockout.LockedUntil != nil && lockout.LockedUntil.After(time.Now()) {
		return true, nil
	}

	return false, nil
}

func (r *userLockoutRepository) UpsertOnFailure(ctx context.Context, tenantID int64, identifier string, ip string, maxAttempts int, lockDuration time.Duration) (*UserLockout, error) {
	var lockout UserLockout
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND identifier = ?", tenantID, identifier).
		First(&lockout).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	applyLock := func(count int) *time.Time {
		if maxAttempts > 0 && lockDuration > 0 && count >= maxAttempts {
			t := now.Add(lockDuration)
			return &t
		}
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		lockout = UserLockout{
			TenantID:     tenantID,
			Identifier:   identifier,
			FailedCount:  1,
			LastFailedAt: &now,
			IPAddress:    &ip,
			LockedUntil:  applyLock(1),
		}
		if err := r.db.WithContext(ctx).Create(&lockout).Error; err != nil {
			return nil, err
		}
		return &lockout, nil
	}

	lockout.FailedCount++
	lockout.LastFailedAt = &now
	if ip != "" {
		lockout.IPAddress = &ip
	}
	if lu := applyLock(lockout.FailedCount); lu != nil {
		lockout.LockedUntil = lu
	}
	if err := r.db.WithContext(ctx).Save(&lockout).Error; err != nil {
		return nil, err
	}
	return &lockout, nil
}

func (r *userLockoutRepository) ClearLockout(ctx context.Context, tenantID int64, identifier string) error {
	return r.db.WithContext(ctx).
		Model(&UserLockout{}).
		Where("tenant_id = ? AND identifier = ?", tenantID, identifier).
		Updates(map[string]any{
			"failed_count":  0,
			"locked_until":  nil,
			"last_failed_at": nil,
		}).Error
}
