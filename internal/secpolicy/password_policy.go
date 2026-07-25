package secpolicy

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

// PasswordHistoryRecorder is the minimal password-history writer needed by
// domains that hash a new user password.
type PasswordHistoryRecorder interface {
	AddEntry(userID int64, passwordHash string) error
	PruneExcess(userID int64, keep int) error
}

// LoadPasswordPolicy returns the effective password policy for a tenant.
//
// Every failure here falls back to the shipped defaults, which is the right
// availability call — a settings-table hiccup must not lock every user out of
// changing their password. But falling back SILENTLY was not: a tenant that had
// raised min_length to 16 and enabled the breach check would quietly drop to the
// defaults, and the only symptom would be passwords being accepted that the
// tenant's own policy forbids. Each fallback now says so.
func LoadPasswordPolicy(repo SecuritySettingRepository, tenantID int64) security.PasswordPolicy {
	if repo == nil {
		// Not an anomaly: several call sites are constructed without a settings
		// repository and are documented to use defaults.
		return defaultPasswordPolicy()
	}
	ss, err := repo.FindByTenantID(tenantID)
	if err != nil {
		return passwordPolicyFallback(tenantID, "security settings lookup failed", err)
	}
	if ss == nil {
		// A tenant with no row yet is expected during provisioning.
		return defaultPasswordPolicy()
	}
	cfg, err := NormalizeSecuritySettingConfig("password", canonicalPasswordConfigAliases(mapFromJSON(ss.PasswordConfig)), nil)
	if err != nil {
		return passwordPolicyFallback(tenantID, "stored password config is not valid", err)
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return passwordPolicyFallback(tenantID, "normalized password config could not be encoded", err)
	}
	return security.MergePasswordPolicy(b)
}

// passwordPolicyFallback logs the degradation and returns the shipped defaults.
func passwordPolicyFallback(tenantID int64, reason string, err error) security.PasswordPolicy {
	slog.Warn("password policy falling back to defaults; the tenant's configured policy is NOT being enforced",
		"tenant_id", tenantID,
		"reason", reason,
		"error", err,
	)
	return defaultPasswordPolicy()
}

// RecordPasswordHistory stores the latest password hash and prunes older entries
// according to the configured history count.
//
// The error is returned rather than discarded. A dropped history write is
// invisible and permanent: the entry that was supposed to stop the user cycling
// straight back to this password simply is not there, and nothing will ever
// notice. Callers decide whether that is fatal for their flow — see
// RecordPasswordHistoryBestEffort for the ones that have already committed.
func RecordPasswordHistory(repo PasswordHistoryRecorder, userID int64, historyCount int, newHash string) error {
	if repo == nil || historyCount <= 0 {
		return nil
	}
	if err := repo.AddEntry(userID, newHash); err != nil {
		return fmt.Errorf("failed to record password history: %w", err)
	}
	if err := repo.PruneExcess(userID, historyCount); err != nil {
		return fmt.Errorf("failed to prune password history: %w", err)
	}
	return nil
}

// RecordPasswordHistoryBestEffort records history and logs — rather than
// returns — a failure. For callers whose password write has already been
// committed, where failing the request would report a change that in fact
// succeeded.
func RecordPasswordHistoryBestEffort(repo PasswordHistoryRecorder, userID int64, historyCount int, newHash string) {
	if err := RecordPasswordHistory(repo, userID, historyCount, newHash); err != nil {
		slog.Error("password history not recorded; reuse of this password will not be detected",
			"user_id", userID,
			"error", err,
		)
	}
}

// DefaultPasswordPolicy is the shipped tenant baseline (12 chars, no composition
// rules, breach check on, no forced rotation) — the NIST 800-63B-aligned posture.
// Exported so bootstrap paths that run before a tenant's settings exist can still
// be held to the same standard.
func DefaultPasswordPolicy() security.PasswordPolicy {
	return defaultPasswordPolicy()
}

func defaultPasswordPolicy() security.PasswordPolicy {
	cfg := MustDefaultSecuritySettingConfig("password")
	b, err := json.Marshal(cfg)
	if err != nil {
		return security.DefaultPasswordPolicy()
	}
	return security.MergePasswordPolicy(b)
}

func mapFromJSON(raw []byte) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

// passwordConfigAliases maps each legacy/short key to the canonical key used by
// the settings schema.
//
// These aliases exist in FOUR places — the defaults map, the DTO json tags, this
// function, and security.MergePasswordPolicy — and two of them used to disagree
// about precedence. Keeping the pairs in one table here at least makes this
// file's behaviour auditable at a glance: canonical wins, always.
var passwordConfigAliases = map[string]string{
	"require_upper":     "require_uppercase",
	"require_lower":     "require_lowercase",
	"require_digit":     "require_number",
	"require_special":   "require_symbol",
	"blocklist_enabled": "reject_common_passwords",
	"history_count":     "password_history_count",
	"expiry_days":       "max_age_days",
}

// canonicalPasswordConfigAliases rewrites short-form keys onto their canonical
// names. When BOTH spellings are present the canonical one wins — the previous
// implementation let the alias overwrite it, so a config carrying both
// require_uppercase:true and require_upper:false silently enforced neither.
func canonicalPasswordConfigAliases(cfg map[string]any) map[string]any {
	if cfg == nil {
		return nil
	}
	for alias, canonical := range passwordConfigAliases {
		aliasValue, hasAlias := cfg[alias]
		if !hasAlias {
			continue
		}
		if _, hasCanonical := cfg[canonical]; hasCanonical {
			continue
		}
		cfg[canonical] = aliasValue
	}
	return cfg
}
