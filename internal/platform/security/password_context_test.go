package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OWASP ASVS V2.1.4 and NIST 800-63B both require context-specific word
// rejection. Without it "acme-corp" is a fully policy-compliant password at
// tenant Acme Corp, and the username is the single most common password guess
// an attacker makes against a named account.
func TestValidatePasswordPolicyForUser_RejectsIdentityRestatements(t *testing.T) {
	t.Parallel()

	policy := PasswordPolicy{MinLength: 8, MaxLength: 128}
	userCtx := PasswordUserContext{
		Username:   "jbarnes",
		Email:      "james.barnes@acme-corp.example",
		FirstName:  "James",
		LastName:   "Barnes",
		TenantName: "Acme Corp",
	}

	tests := []struct {
		name      string
		password  string
		wantLabel string
	}{
		{"the username itself", "jbarnes-2026-x", "username"},
		{"the first name", "jamesjamesjames", "first name"},
		{"the surname", "quiet-barnes-door", "last name"},
		{"the tenant name with a separator", "Acme-Corp-Winter", "organization name"},
		{"the tenant name run together", "acmecorprocks", "organization name"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePasswordPolicyForUser(context.Background(), tc.password, policy, userCtx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantLabel)
		})
	}
}

// The local part of the address is checked separately from the whole address,
// because the local part is the bit people actually reuse as a password.
func TestValidatePasswordPolicyForUser_RejectsTheEmailLocalPart(t *testing.T) {
	t.Parallel()

	err := ValidatePasswordPolicyForUser(context.Background(), "quiet-jb.ops-door",
		PasswordPolicy{MinLength: 8, MaxLength: 128},
		PasswordUserContext{Email: "jb.ops@acme-corp.example"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "email address")
}

// The error must name the FIELD, never echo the value — an error message is one
// of the easiest places to leak an identifier into a log or a screenshot.
func TestValidatePasswordPolicyForUser_ErrorDoesNotEchoTheIdentity(t *testing.T) {
	t.Parallel()

	err := ValidatePasswordPolicyForUser(context.Background(), "jbarnes-2026-x",
		PasswordPolicy{MinLength: 8, MaxLength: 128},
		PasswordUserContext{Username: "jbarnes"})

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "jbarnes")
}

func TestValidatePasswordPolicyForUser_AllowsUnrelatedPasswords(t *testing.T) {
	t.Parallel()

	err := ValidatePasswordPolicyForUser(context.Background(), "vault-crimson-ledger-92",
		PasswordPolicy{MinLength: 8, MaxLength: 128},
		PasswordUserContext{Username: "jbarnes", Email: "james.barnes@acme-corp.example", TenantName: "Acme Corp"})

	assert.NoError(t, err)
}

// Short identity values would veto ordinary passwords — a user named "Al" must
// not make every password containing "al" invalid.
func TestValidatePasswordPolicyForUser_IgnoresVeryShortIdentityValues(t *testing.T) {
	t.Parallel()

	err := ValidatePasswordPolicyForUser(context.Background(), "vault-crimson-ledger",
		PasswordPolicy{MinLength: 8, MaxLength: 128},
		PasswordUserContext{FirstName: "Al", LastName: "Vu"})

	assert.NoError(t, err)
}

// Context checking is independent of the blocklist toggle: a password that is
// just the username is not a "common password" problem, it is an
// account-takeover problem, and disabling the common-password list must not
// disable it.
func TestValidatePasswordPolicyForUser_AppliesEvenWithBlocklistDisabled(t *testing.T) {
	t.Parallel()

	err := ValidatePasswordPolicyForUser(context.Background(), "jbarnes-jbarnes",
		PasswordPolicy{MinLength: 8, MaxLength: 128, BlocklistEnabled: false},
		PasswordUserContext{Username: "jbarnes"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "username")
}

// The context-free entry point must keep behaving exactly as before, so the
// call sites that genuinely do not know the user are unaffected.
func TestValidatePasswordPolicyWithContext_HasNoIdentityContext(t *testing.T) {
	t.Parallel()

	assert.NoError(t, ValidatePasswordPolicyWithContext(context.Background(), "jbarnes-jbarnes",
		PasswordPolicy{MinLength: 8, MaxLength: 128}))
}
