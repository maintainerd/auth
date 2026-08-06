package secpolicy

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// A nil policy means lockout is OFF: internal/authn's login path early-returns
// before counting a failed attempt when LoadLockoutPolicy returns nil. So every
// assertion here is really "brute-force protection is still switched on".
func TestLoadLockoutPolicy_NeverDisablesLockoutOnFailure(t *testing.T) {
	t.Run("nil repo falls back to defaults", func(t *testing.T) {
		// INVERTED: this used to return nil, i.e. no lockout at all for every
		// call site constructed without a settings repository.
		cfg := LoadLockoutPolicy(nil, 1)
		require.NotNil(t, cfg)
		assert.True(t, cfg.Enabled)
		assert.Equal(t, 5, cfg.MaxFailedAttempts)
	})

	t.Run("repo error falls back to defaults", func(t *testing.T) {
		// INVERTED: a transient database error used to disable lockout for the
		// tenant entirely — unlimited online guessing, while the admin UI still
		// showed max_failed_attempts: 5.
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(int64) (*SecuritySetting, error) {
				return nil, errors.New("connection refused")
			},
		}
		cfg := LoadLockoutPolicy(repo, 1)
		require.NotNil(t, cfg)
		assert.True(t, cfg.Enabled)
		assert.Equal(t, 5, cfg.MaxFailedAttempts)
		assert.Equal(t, 30*time.Minute, cfg.LockoutDuration)
	})

	t.Run("unnormalizable stored config falls back to defaults", func(t *testing.T) {
		// INVERTED: a single stale config row used to switch lockout off.
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(int64) (*SecuritySetting, error) {
				return &SecuritySetting{
					LockoutConfig: datatypes.JSON(`{"max_failed_attempts":"not-a-number"}`),
				}, nil
			},
		}
		cfg := LoadLockoutPolicy(repo, 1)
		require.NotNil(t, cfg)
		assert.True(t, cfg.Enabled)
		assert.Equal(t, 5, cfg.MaxFailedAttempts)
	})

	t.Run("no settings row yet falls back to defaults", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(int64) (*SecuritySetting, error) { return nil, nil },
		}
		cfg := LoadLockoutPolicy(repo, 1)
		require.NotNil(t, cfg)
		assert.True(t, cfg.Enabled)
	})

	t.Run("a valid stored config is still honoured over the defaults", func(t *testing.T) {
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(int64) (*SecuritySetting, error) {
				return &SecuritySetting{
					LockoutConfig: datatypes.JSON(`{"max_failed_attempts":3,"lockout_duration_minutes":10}`),
				}, nil
			},
		}
		cfg := LoadLockoutPolicy(repo, 1)
		require.NotNil(t, cfg)
		assert.Equal(t, 3, cfg.MaxFailedAttempts)
		assert.Equal(t, 10*time.Minute, cfg.LockoutDuration)
	})

	t.Run("an explicitly disabled policy is still honoured", func(t *testing.T) {
		// Falling back to defaults must not override a deliberate opt-out.
		repo := &mockSecuritySettingRepo{
			findByTenantIDFn: func(int64) (*SecuritySetting, error) {
				return &SecuritySetting{LockoutConfig: datatypes.JSON(`{"enabled":false}`)}, nil
			},
		}
		cfg := LoadLockoutPolicy(repo, 1)
		require.NotNil(t, cfg)
		assert.False(t, cfg.Enabled)
	})
}

func TestLoadMFAPolicy_DegradesToDefaultsOnLookupError(t *testing.T) {
	// The default mode is "optional", so this is a downgrade from an "enforced"
	// tenant — availability wins, but it is logged rather than silent, and the
	// discarded-error path (`err == nil && ss != nil`) is now explicit.
	repo := &mockSecuritySettingRepo{
		findByTenantIDFn: func(int64) (*SecuritySetting, error) {
			return nil, errors.New("connection refused")
		},
	}
	policy := LoadMFAPolicy(repo, 1)
	require.NotNil(t, policy)
	assert.Equal(t, "optional", policy.Mode)

	// repo == nil keeps its documented contract: nil means "no settings
	// available, treat as permissive".
	assert.Nil(t, LoadMFAPolicy(nil, 1))
}

func TestLoadMFAPolicy_HonoursStoredEnforcedMode(t *testing.T) {
	repo := &mockSecuritySettingRepo{
		findByTenantIDFn: func(int64) (*SecuritySetting, error) {
			return &SecuritySetting{MFAConfig: datatypes.JSON(`{"mode":"enforced"}`)}, nil
		},
	}
	policy := LoadMFAPolicy(repo, 1)
	require.NotNil(t, policy)
	assert.Equal(t, "enforced", policy.Mode)
}

// A public client cannot hold a credential, so PKCE is the only thing between an
// observed authorization code and the victim's tokens. The resolver only ever
// ESCALATES, which meant a client row carrying require_pkce=false could not be
// raised by anything once a tenant turned the default off. RFC 9700 §2.1.1.
func TestResolveEffectiveTokenPolicy_PublicClientForcesPKCE(t *testing.T) {
	tenantPKCEOff := map[string]any{"require_pkce": false}

	t.Run("a tenant can turn PKCE off for a confidential client", func(t *testing.T) {
		policy, err := ResolveEffectiveTokenPolicy(tenantPKCEOff, SecuritySettingClientOverrides{})
		require.NoError(t, err)
		assert.False(t, policy.RequirePKCE)
	})

	t.Run("a public client is forced on regardless", func(t *testing.T) {
		policy, err := ResolveEffectiveTokenPolicy(tenantPKCEOff, SecuritySettingClientOverrides{PublicClient: true})
		require.NoError(t, err)
		assert.True(t, policy.RequirePKCE)
	})

	t.Run("a public client's own require_pkce=false cannot override it", func(t *testing.T) {
		off := false
		policy, err := ResolveEffectiveTokenPolicy(tenantPKCEOff, SecuritySettingClientOverrides{
			PublicClient: true,
			RequirePKCE:  &off,
		})
		require.NoError(t, err)
		assert.True(t, policy.RequirePKCE)
	})
}
