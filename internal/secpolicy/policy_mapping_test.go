package secpolicy_test

// Black-box verification that LoadPasswordPolicy maps the stored password_config
// JSON onto the enforced security.PasswordPolicy struct — including fields whose
// DB key differs from the struct's json tag (require_uppercase→RequireUpper,
// reject_common_passwords→BlocklistEnabled, password_history_count→HistoryCount,
// max_age_days→ExpiryDays). No *password* source file is read; this only exercises
// the public API.

import (
	"testing"

	"gorm.io/datatypes"

	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
)

// stubSettingRepo satisfies SecuritySettingRepository by embedding the interface
// (nil) and overriding only FindByTenantID, which is all LoadPasswordPolicy calls.
type stubSettingRepo struct {
	secpolicy.SecuritySettingRepository
	setting *secpolicy.SecuritySetting
}

func (s stubSettingRepo) FindByTenantID(int64) (*secpolicy.SecuritySetting, error) {
	return s.setting, nil
}

func load(t *testing.T, passwordConfig string) (minLen, maxLen, strength, history, expiry int, upper, lower, digit, special, hibp, blocklist bool) {
	t.Helper()
	repo := stubSettingRepo{setting: &secpolicy.SecuritySetting{PasswordConfig: datatypes.JSON(passwordConfig)}}
	p := secpolicy.LoadPasswordPolicy(repo, 1)
	return p.MinLength, p.MaxLength, p.MinStrengthScore, p.HistoryCount, p.ExpiryDays,
		p.RequireUpper, p.RequireLower, p.RequireDigit, p.RequireSpecial, p.CheckHIBP, p.BlocklistEnabled
}

func TestLoadPasswordPolicy_MapsStoredConfig(t *testing.T) {
	t.Run("all-off config is wired through", func(t *testing.T) {
		cfg := `{"check_hibp":false,"max_length":128,"min_length":12,"max_age_days":0,
			"hash_algorithm":"argon2id","require_number":false,"require_symbol":false,
			"require_lowercase":false,"require_uppercase":false,"min_strength_score":0,
			"password_history_count":5,"reject_common_passwords":false,
			"temporary_password_validity_hours":72}`
		minLen, maxLen, strength, history, expiry, upper, lower, digit, special, hibp, blocklist := load(t, cfg)
		if minLen != 12 || maxLen != 128 {
			t.Fatalf("length not mapped: min=%d max=%d", minLen, maxLen)
		}
		if strength != 0 || hibp || blocklist {
			t.Fatalf("strength/hibp/blocklist should be off: strength=%d hibp=%v blocklist=%v", strength, hibp, blocklist)
		}
		if upper || lower || digit || special {
			t.Fatalf("complexity should be off: U=%v L=%v D=%v S=%v", upper, lower, digit, special)
		}
		if history != 5 {
			t.Fatalf("password_history_count should map to HistoryCount=5, got %d", history)
		}
		if expiry != 0 {
			t.Fatalf("max_age_days should map to ExpiryDays=0, got %d", expiry)
		}
	})

	t.Run("all-on config is wired through (name-mismatched keys included)", func(t *testing.T) {
		cfg := `{"check_hibp":true,"max_length":100,"min_length":16,"max_age_days":90,
			"hash_algorithm":"argon2id","require_number":true,"require_symbol":true,
			"require_lowercase":true,"require_uppercase":true,"min_strength_score":3,
			"password_history_count":7,"reject_common_passwords":true,
			"temporary_password_validity_hours":48}`
		minLen, maxLen, strength, history, expiry, upper, lower, digit, special, hibp, blocklist := load(t, cfg)
		if minLen != 16 || maxLen != 100 {
			t.Fatalf("length not mapped: min=%d max=%d", minLen, maxLen)
		}
		if strength != 3 {
			t.Fatalf("min_strength_score should map to 3, got %d", strength)
		}
		if !hibp {
			t.Fatal("check_hibp should map to CheckHIBP=true")
		}
		if !blocklist {
			t.Fatal("reject_common_passwords should map to BlocklistEnabled=true")
		}
		if !upper || !lower || !digit || !special {
			t.Fatalf("complexity should map on: U=%v L=%v D=%v S=%v", upper, lower, digit, special)
		}
		if history != 7 {
			t.Fatalf("password_history_count should map to HistoryCount=7, got %d", history)
		}
		if expiry != 90 {
			t.Fatalf("max_age_days should map to ExpiryDays=90, got %d", expiry)
		}
	})
}
