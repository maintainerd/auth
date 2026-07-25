package secpolicy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPasswordHistoryRecorder struct {
	addEntryFn    func(userID int64, passwordHash string) error
	pruneExcessFn func(userID int64, keep int) error
}

func (m *mockPasswordHistoryRecorder) AddEntry(userID int64, passwordHash string) error {
	if m.addEntryFn != nil {
		return m.addEntryFn(userID, passwordHash)
	}
	return nil
}

func (m *mockPasswordHistoryRecorder) PruneExcess(userID int64, keep int) error {
	if m.pruneExcessFn != nil {
		return m.pruneExcessFn(userID, keep)
	}
	return nil
}

func TestLoadPasswordPolicy(t *testing.T) {
	t.Run("nil repo returns default", func(t *testing.T) {
		policy := LoadPasswordPolicy(nil, 1)
		assert.Equal(t, 12, policy.MinLength)
		assert.False(t, policy.RequireUpper)
		assert.Equal(t, 5, policy.HistoryCount)
	})

	t.Run("repo error returns default", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(tenantID int64) (*SecuritySetting, error) {
				return nil, errors.New("db error")
			},
		}
		policy := LoadPasswordPolicy(repo, 1)
		assert.Equal(t, 12, policy.MinLength)
	})

	t.Run("nil setting returns default", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(tenantID int64) (*SecuritySetting, error) {
				return nil, nil
			},
		}
		policy := LoadPasswordPolicy(repo, 1)
		assert.Equal(t, 12, policy.MinLength)
	})

	t.Run("success merged policy", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(tenantID int64) (*SecuritySetting, error) {
				return &SecuritySetting{
					PasswordConfig: []byte(`{"min_length":16}`),
				}, nil
			},
		}
		policy := LoadPasswordPolicy(repo, 1)
		assert.Equal(t, 16, policy.MinLength)
	})

	t.Run("empty config returns default", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(tenantID int64) (*SecuritySetting, error) {
				return &SecuritySetting{
					PasswordConfig: []byte(`{}`),
				}, nil
			},
		}
		policy := LoadPasswordPolicy(repo, 1)
		assert.Equal(t, 12, policy.MinLength)
	})
}

func TestRecordPasswordHistory(t *testing.T) {
	t.Run("nil repo is no-op", func(t *testing.T) {
		RecordPasswordHistory(nil, 1, 5, "hash")
	})

	t.Run("historyCount <= 0 is no-op", func(t *testing.T) {
		recorder := &mockPasswordHistoryRecorder{
			addEntryFn: func(userID int64, passwordHash string) error {
				t.Error("should not be called")
				return nil
			},
		}
		RecordPasswordHistory(recorder, 1, 0, "hash")
	})

	t.Run("negative historyCount is no-op", func(t *testing.T) {
		recorder := &mockPasswordHistoryRecorder{
			addEntryFn: func(userID int64, passwordHash string) error {
				t.Error("should not be called")
				return nil
			},
		}
		RecordPasswordHistory(recorder, 1, -1, "hash")
	})

	t.Run("success calls add and prune", func(t *testing.T) {
		addCalled := false
		pruneCalled := false
		recorder := &mockPasswordHistoryRecorder{
			addEntryFn: func(userID int64, passwordHash string) error {
				addCalled = true
				assert.Equal(t, int64(1), userID)
				assert.Equal(t, "hash", passwordHash)
				return nil
			},
			pruneExcessFn: func(userID int64, keep int) error {
				pruneCalled = true
				assert.Equal(t, int64(1), userID)
				assert.Equal(t, 5, keep)
				return nil
			},
		}
		RecordPasswordHistory(recorder, 1, 5, "hash")
		assert.True(t, addCalled)
		assert.True(t, pruneCalled)
	})

	// A failed AddEntry used to be swallowed and pruning carried on regardless.
	// The entry that was supposed to stop password reuse simply would not exist
	// and nothing would report it, so the error is now surfaced and pruning is
	// skipped — there is nothing new to prune down to.
	t.Run("addEntry error is returned and pruning is skipped", func(t *testing.T) {
		pruneCalled := false
		recorder := &mockPasswordHistoryRecorder{
			addEntryFn: func(userID int64, passwordHash string) error {
				return errors.New("add error")
			},
			pruneExcessFn: func(userID int64, keep int) error {
				pruneCalled = true
				return nil
			},
		}
		err := RecordPasswordHistory(recorder, 1, 5, "hash")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to record password history")
		assert.False(t, pruneCalled)
	})

	t.Run("pruneExcess error is returned", func(t *testing.T) {
		recorder := &mockPasswordHistoryRecorder{
			addEntryFn: func(userID int64, passwordHash string) error {
				return nil
			},
			pruneExcessFn: func(userID int64, keep int) error {
				return errors.New("prune error")
			},
		}
		err := RecordPasswordHistory(recorder, 1, 5, "hash")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to prune password history")
	})

	t.Run("best-effort variant swallows the error for already-committed callers", func(t *testing.T) {
		recorder := &mockPasswordHistoryRecorder{
			addEntryFn: func(userID int64, passwordHash string) error {
				return errors.New("add error")
			},
		}
		assert.NotPanics(t, func() {
			RecordPasswordHistoryBestEffort(recorder, 1, 5, "hash")
		})
	})
}

// Password config is validated on READ as well as write: LoadPasswordPolicy →
// NormalizeSecuritySettingConfig → decodeSecuritySettingPatch →
// validatePasswordConfig. So tightening a bound retroactively invalidates every
// already-stored config that sits outside it, and those tenants silently fall
// back to the shipped defaults for ALL thirteen fields — a far bigger change
// than the one bound that was tightened.
//
// This test pins that read path. If it starts failing, a validation bound was
// tightened without accounting for stored data.
func TestNormalizeSecuritySettingConfig_StoredConfigOutsideRecommendedBoundsStillLoads(t *testing.T) {
	stored := map[string]any{
		"min_length":         float64(6),
		"min_strength_score": float64(0),
		"check_hibp":         false,
	}

	cfg, err := NormalizeSecuritySettingConfig("password", stored, nil)
	require.NoError(t, err, "a stored config below the recommended minimum must still load, not fall back to defaults")
	assert.Equal(t, float64(6), cfg["min_length"], "the tenant's own value must survive normalization")
	assert.Equal(t, false, cfg["check_hibp"])
}
