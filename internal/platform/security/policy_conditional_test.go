package security_test

// Black-box verification that ValidatePasswordPolicy enforces each check ONLY
// when the corresponding config field is enabled (i.e. it is driven by the
// policy, not blindly applied). Offline checks only — CheckHIBP is intentionally
// left off here since it performs a network lookup.

import (
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

func TestValidatePasswordPolicy_IsConfigConditional(t *testing.T) {
	const pw = "111111111111" // 12 chars, digits only, common/weak

	// Length-only policy with every optional check OFF.
	base := security.PasswordPolicy{
		MinLength:        12,
		MaxLength:        128,
		HashAlgorithm:    "argon2id",
		MinStrengthScore: 0,     // disabled
		CheckHIBP:        false, // disabled
		BlocklistEnabled: false, // disabled
	}

	t.Run("all optional checks off → length-only password passes", func(t *testing.T) {
		if err := security.ValidatePasswordPolicy(pw, base); err != nil {
			t.Fatalf("expected pass (only length enforced), got error: %v", err)
		}
	})

	t.Run("min_strength_score on → rejected", func(t *testing.T) {
		p := base
		p.MinStrengthScore = 2
		if err := security.ValidatePasswordPolicy(pw, p); err == nil {
			t.Fatal("expected rejection when min_strength_score is enabled")
		}
	})

	t.Run("require_digit only → passes (digits satisfy it)", func(t *testing.T) {
		p := base
		p.RequireDigit = true
		if err := security.ValidatePasswordPolicy(pw, p); err != nil {
			t.Fatalf("expected pass (digit requirement met), got: %v", err)
		}
	})

	t.Run("require_upper on → rejected (no uppercase present)", func(t *testing.T) {
		p := base
		p.RequireUpper = true
		if err := security.ValidatePasswordPolicy(pw, p); err == nil {
			t.Fatal("expected rejection when uppercase is required")
		}
	})

	t.Run("min_length not met → rejected", func(t *testing.T) {
		p := base
		p.MinLength = 20
		if err := security.ValidatePasswordPolicy(pw, p); err == nil {
			t.Fatal("expected rejection when below min length")
		}
	})
}
