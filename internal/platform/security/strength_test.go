package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The whole point of replacing the old class-counting scorer is that character
// classes must stop buying score. These are the cases the old implementation got
// backwards: it rated "Passw0rd!" near the top (four classes) and a long
// passphrase near the bottom (one class).
func TestPasswordStrengthScore_RewardsUnguessabilityNotCharacterClasses(t *testing.T) {
	t.Parallel()

	classHeavyButGuessable := PasswordStrengthScore("Passw0rd!")
	lowercasePassphrase := PasswordStrengthScore("correct horse battery staple")

	assert.LessOrEqual(t, classHeavyButGuessable, 1,
		"a leet-disguised blocklist word must score weak no matter how many character classes it hits")
	assert.Equal(t, 4, lowercasePassphrase,
		"a long single-class passphrase must score strong")
	assert.Greater(t, lowercasePassphrase, classHeavyButGuessable,
		"the passphrase is orders of magnitude harder to guess and must outrank the class-heavy password")
}

// A tenant that disables composition rules must not have them reimposed by
// raising MinStrengthScore. Adding a symbol to an otherwise identical password
// must not, on its own, move the score.
func TestPasswordStrengthScore_AddingASymbolDoesNotBuyScore(t *testing.T) {
	t.Parallel()

	withoutSymbol := PasswordStrengthScore("kwjhtrnbdxpq")
	withSymbol := PasswordStrengthScore("kwjhtrnbdxp!")

	assert.Equal(t, withoutSymbol, withSymbol,
		"swapping a letter for a symbol changes no guessing cost and must not change the score")
}

func TestPasswordStrengthScore_PenalizesRecognizablePatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		maxScore int
	}{
		{"empty", "", 0},
		{"single repeated character", "aaaaaaaaaaaa", 1},
		{"alphabetical run", "abcdefghijkl", 1},
		{"keyboard walk", "qwertyuiop", 1},
		{"blocklist word", "letmein", 1},
		{"blocklist word with year", "welcome2026", 2},
		{"seasonal rotation password", "Summer2026!", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.LessOrEqual(t, PasswordStrengthScore(tc.password), tc.maxScore)
		})
	}
}

func TestPasswordStrengthScore_LengthDrivesTheScore(t *testing.T) {
	t.Parallel()

	// Same alphabet, increasing length: the score must be monotonic.
	previous := -1
	for _, password := range []string{
		"xk9m",
		"xk9mrq2",
		"xk9mrq2vtb",
		"xk9mrq2vtbzh4wn",
	} {
		score := PasswordStrengthScore(password)
		assert.GreaterOrEqual(t, score, previous, "score must not fall as length grows: %q", password)
		previous = score
	}
	assert.Equal(t, 4, previous, "a 15-character unpatterned password must reach the top band")
}

func TestPasswordStrengthScore_StaysWithinZeroToFour(t *testing.T) {
	t.Parallel()

	for _, password := range []string{"", "a", "aB3$", "correct horse battery staple correct horse battery staple"} {
		score := PasswordStrengthScore(password)
		assert.GreaterOrEqual(t, score, 0)
		assert.LessOrEqual(t, score, 4)
	}
}

func TestIsBlocklisted_SeesThroughTheCommonDisguises(t *testing.T) {
	t.Parallel()

	tests := []struct {
		password      string
		wantMatch     bool
		wantDisguised bool
		why           string
	}{
		{"password", true, false, "the literal entry"},
		{"P@ssw0rd", true, true, "symbol and digit substitutions"},
		// Matches the literal entry "password1!" before any disguise-undoing is
		// needed — the list already carries the composition-rule spellings.
		{"Password1!", true, false, "a composition-rule spelling that is itself on the list"},
		{"Passw0rd1!", true, true, "composition-rule suffix plus a substitution"},
		{"L3tm31n", true, true, "'1' read as 'i', which is the ambiguous case"},
		{"  MONKEY  ", true, false, "case and surrounding whitespace are not a disguise"},
		{"$unshine", true, true, "leading symbol substitution"},
		// Regression: blanket leet rewriting turned this real entry into
		// "summerzozgi" and stopped matching it.
		{"summer2026!", true, false, "an entry that legitimately contains digits"},
		{"kwjhtrnbdxpq", false, false, "not on the list under any reading"},
	}

	for _, tc := range tests {
		t.Run(tc.password, func(t *testing.T) {
			t.Parallel()
			matched, disguised := isBlocklisted(tc.password)
			assert.Equal(t, tc.wantMatch, matched, tc.why)
			if tc.wantMatch {
				assert.Equal(t, tc.wantDisguised, disguised, tc.why)
			}
		})
	}
}

// A password merely CONTAINING a common word must still be allowed — matching is
// exact against the normalized forms, never substring.
func TestIsBlocklisted_DoesNotMatchOnSubstrings(t *testing.T) {
	t.Parallel()

	matched, _ := isBlocklisted("my-password-is-a-long-one")
	assert.False(t, matched)
}

// Unbounded substitution fan-out on a digit-heavy password would be a denial of
// service; the expansion is capped.
func TestLeetVariants_IsBounded(t *testing.T) {
	t.Parallel()

	assert.LessOrEqual(t, len(leetVariants("1111111111111111111111111111")), maxLeetVariants)
}

// The disguised forms must actually be rejected end-to-end, not merely
// normalized — that is the behaviour change users will feel.
func TestValidatePasswordPolicy_RejectsDisguisedBlocklistEntries(t *testing.T) {
	t.Parallel()

	policy := PasswordPolicy{MinLength: 1, MaxLength: 128, BlocklistEnabled: true}

	for _, password := range []string{"password", "P@ssw0rd", "PASSWORD", "changeme!", "L3tm31n"} {
		assert.Error(t, ValidatePasswordPolicy(password, policy), "expected %q to be rejected", password)
	}
	assert.NoError(t, ValidatePasswordPolicy("kwjhtrnbdxpq", policy))
}

func TestValidatePasswordPolicy_BlocklistDisabledSkipsTheCheck(t *testing.T) {
	t.Parallel()

	policy := PasswordPolicy{MinLength: 1, MaxLength: 128, BlocklistEnabled: false}
	assert.NoError(t, ValidatePasswordPolicy("password", policy))
}
