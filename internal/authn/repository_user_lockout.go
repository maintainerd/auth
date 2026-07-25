package authn

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// IsLocked reports whether the identifier is currently locked. Two lock states
// count: a permanent lock (auto_unlock=false) and a timed lock whose expiry is
// still in the future.
func (r *userLockoutRepository) IsLocked(ctx context.Context, tenantID int64, identifier string) (bool, error) {
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
	return lockout.isCurrentlyLocked(time.Now()), nil
}

func (l *UserLockout) isCurrentlyLocked(now time.Time) bool {
	if l.Locked {
		return true
	}
	return l.LockedUntil != nil && l.LockedUntil.After(now)
}

// RecordFailure records one failed attempt and applies the lockout policy.
//
// The whole read-modify-write runs inside a transaction with the row locked
// FOR UPDATE, so concurrent failed logins for the same identifier cannot lose
// increments and slip past max_failed_attempts (the previous read-then-Save had
// exactly that race). An INSERT ... ON CONFLICT DO NOTHING guarantees the row
// exists before it is locked, without failing when two requests race to create.
//
// It honors every lockout-policy field the old UpsertOnFailure ignored:
//   - observation_window: failures older than the window start a fresh count.
//   - progressive_lockout: each lock escalates the level; duration = level*base
//     capped at max_lockout_duration; the level resets after progression_reset.
//   - auto_unlock=false: a permanent lock (no expiry) instead of a timed one.
func (r *userLockoutRepository) RecordFailure(ctx context.Context, tenantID int64, identifier, ip string, rules LockoutRules) (LockoutResult, error) {
	var result LockoutResult
	txErr := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Ensure the row exists so the FOR UPDATE lock below has something to
		// grab, race-free.
		seed := UserLockout{TenantID: tenantID, Identifier: identifier}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
			return err
		}

		var lockout UserLockout
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND identifier = ?", tenantID, identifier).
			First(&lockout).Error; err != nil {
			return err
		}

		now := time.Now()
		wasLocked := lockout.isCurrentlyLocked(now)

		lockout.FailedCount = nextFailedCount(lockout.FailedCount, lockout.LastFailedAt, now, rules.ObservationWindow)
		lockout.LastFailedAt = &now
		if ip != "" {
			lockout.IPAddress = &ip
		}

		if rules.MaxAttempts > 0 && lockout.FailedCount >= rules.MaxAttempts {
			applyLockout(&lockout, now, rules)
			result.JustLocked = !wasLocked
		}

		if err := tx.Save(&lockout).Error; err != nil {
			return err
		}
		result.Locked = lockout.isCurrentlyLocked(now)
		return nil
	})
	return result, txErr
}

// nextFailedCount returns the failure count after this attempt, honoring the
// observation window: a failure arriving after the window has elapsed since the
// previous one begins a fresh count rather than adding to a stale one. Without
// this the count only ever grew, so once a user was locked every later failure
// re-locked them forever (the fix for the observation_window dead-config bug).
func nextFailedCount(current int, lastFailedAt *time.Time, now time.Time, window time.Duration) int {
	if window > 0 && lastFailedAt != nil && now.Sub(*lastFailedAt) > window {
		return 1
	}
	return current + 1
}

// applyLockout mutates the row to reflect a fresh lock, computing the escalated
// duration for progressive policies and choosing a permanent vs timed lock.
func applyLockout(lockout *UserLockout, now time.Time, rules LockoutRules) {
	level := 1
	if rules.Progressive {
		// The tier resets once progression_reset has elapsed since the last
		// lock; ProgressionReset<=0 means it never resets (always escalates),
		// mirroring the original Redis behavior which fell back to the
		// observation window and then to "no expiry".
		resetAfter := rules.ProgressionReset
		if resetAfter <= 0 {
			resetAfter = rules.ObservationWindow
		}
		escalate := lockout.LastLockedAt != nil &&
			(resetAfter <= 0 || now.Sub(*lockout.LastLockedAt) <= resetAfter)
		if escalate {
			level = lockout.LockoutLevel + 1
		}
	}
	lockout.LockoutLevel = level
	lockout.LastLockedAt = &now

	if !rules.AutoUnlock {
		// Permanent lock: only an admin ClearLockout releases it.
		lockout.Locked = true
		lockout.LockedUntil = nil
		return
	}

	duration := rules.BaseDuration
	if rules.Progressive {
		duration = time.Duration(level) * rules.BaseDuration
		if rules.MaxDuration > 0 && duration > rules.MaxDuration {
			duration = rules.MaxDuration
		}
	}
	until := now.Add(duration)
	lockout.Locked = false
	lockout.LockedUntil = &until
}

// ClearLockout fully releases an identifier: the failure count, both lock
// states, and the escalation level. Used on successful login (when the policy
// resets on success) and by an admin unlock.
func (r *userLockoutRepository) ClearLockout(ctx context.Context, tenantID int64, identifier string) error {
	return r.db.WithContext(ctx).
		Model(&UserLockout{}).
		Where("tenant_id = ? AND identifier = ?", tenantID, identifier).
		Updates(map[string]any{
			"failed_count":   0,
			"locked_until":   nil,
			"locked":         false,
			"lockout_level":  0,
			"last_locked_at": nil,
			"last_failed_at": nil,
		}).Error
}

// ResetExpiredLockouts clears timed locks whose expiry has passed. Permanent
// locks (auto_unlock=false) are deliberately left for an admin to release.
func (r *userLockoutRepository) ResetExpiredLockouts() (int64, error) {
	result := r.db.
		Model(&UserLockout{}).
		Where("locked = false AND locked_until IS NOT NULL AND locked_until < ?", time.Now()).
		Updates(map[string]any{
			"failed_count":   0,
			"locked_until":   nil,
			"last_failed_at": nil,
		})
	return result.RowsAffected, result.Error
}
