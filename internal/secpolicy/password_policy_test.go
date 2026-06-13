package secpolicy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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

	t.Run("addEntry error does not panic", func(t *testing.T) {
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
		RecordPasswordHistory(recorder, 1, 5, "hash")
		assert.True(t, pruneCalled)
	})

	t.Run("pruneExcess error does not panic", func(t *testing.T) {
		recorder := &mockPasswordHistoryRecorder{
			addEntryFn: func(userID int64, passwordHash string) error {
				return nil
			},
			pruneExcessFn: func(userID int64, keep int) error {
				return errors.New("prune error")
			},
		}
		RecordPasswordHistory(recorder, 1, 5, "hash")
	})
}
