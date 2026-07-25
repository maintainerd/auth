package secpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The blocklist is a hard org policy. Blocking "competitor.com" must also block
// its subdomains, or the block is trivially evaded via mail.competitor.com.
func TestEmailDomainBlocked_CatchesSubdomains(t *testing.T) {
	p := &RegistrationPolicy{BlockedEmailDomains: []string{"competitor.com"}}

	assert.True(t, p.EmailDomainBlocked("eve@competitor.com"), "apex must be blocked")
	assert.True(t, p.EmailDomainBlocked("eve@mail.competitor.com"), "subdomain must be blocked")
	assert.True(t, p.EmailDomainBlocked("eve@a.b.competitor.com"), "deep subdomain must be blocked")

	assert.False(t, p.EmailDomainBlocked("eve@notcompetitor.com"), "a different domain must not be caught")
	assert.False(t, p.EmailDomainBlocked("eve@mycompetitor.com"), "a suffix that is not a subdomain must not be caught")
	assert.False(t, p.EmailDomainBlocked("eve@acme.com"), "unrelated domain passes")
}

func TestEmailDomainBlocked_EdgeCases(t *testing.T) {
	t.Run("nil policy is not blocked", func(t *testing.T) {
		var p *RegistrationPolicy
		assert.False(t, p.EmailDomainBlocked("eve@competitor.com"))
	})
	t.Run("empty blocklist blocks nothing", func(t *testing.T) {
		p := &RegistrationPolicy{}
		assert.False(t, p.EmailDomainBlocked("eve@competitor.com"))
	})
	t.Run("no @ passes", func(t *testing.T) {
		p := &RegistrationPolicy{BlockedEmailDomains: []string{"competitor.com"}}
		assert.False(t, p.EmailDomainBlocked("not-an-email"))
	})
	t.Run("case-insensitive", func(t *testing.T) {
		p := &RegistrationPolicy{BlockedEmailDomains: []string{"Competitor.COM"}}
		assert.True(t, p.EmailDomainBlocked("eve@MAIL.competitor.com"))
	})
}

// EmailDomainAllowed must apply the same subdomain-aware blocklist (blocklist
// takes precedence over the allowlist).
func TestEmailDomainAllowed_BlocklistSubdomainPrecedence(t *testing.T) {
	p := &RegistrationPolicy{
		AllowedEmailDomains: []string{"acme.com"},
		BlockedEmailDomains: []string{"competitor.com"},
	}
	assert.True(t, p.EmailDomainAllowed("alice@acme.com"))
	assert.False(t, p.EmailDomainAllowed("bob@other.com"), "not on the allowlist")
	assert.False(t, p.EmailDomainAllowed("eve@mail.competitor.com"),
		"a blocked subdomain must be rejected even before allowlist evaluation")
}

// The allowlist stays exact/explicit-wildcard — a bare allow entry must NOT
// silently admit subdomains (a stricter allowlist is the safer default).
func TestEmailDomainAllowed_AllowlistIsExactOrWildcard(t *testing.T) {
	exact := &RegistrationPolicy{AllowedEmailDomains: []string{"acme.com"}}
	assert.True(t, exact.EmailDomainAllowed("a@acme.com"))
	assert.False(t, exact.EmailDomainAllowed("a@sub.acme.com"), "bare allow entry must not admit subdomains")

	wild := &RegistrationPolicy{AllowedEmailDomains: []string{"*.acme.com"}}
	assert.True(t, wild.EmailDomainAllowed("a@sub.acme.com"), "explicit wildcard admits subdomains")
	assert.True(t, wild.EmailDomainAllowed("a@acme.com"), "explicit wildcard also admits the apex")
}

// P3: account activation is decoupled from the email_verified claim. Activation
// still follows policy (auto_confirm / require_email_verification); the verified
// claim (set false at registration by the caller) is not derived from policy.
func TestInitialUserStatus_ActivationPolicy(t *testing.T) {
	tests := []struct {
		name  string
		p     *RegistrationPolicy
		email string
		want  string
	}{
		{"require verification -> pending", &RegistrationPolicy{RequireEmailVerification: true}, "a@b.com", "pending"},
		{"auto-confirm -> active", &RegistrationPolicy{RequireEmailVerification: true, AutoConfirmEnabled: true}, "a@b.com", "active"},
		{"verification not required -> active", &RegistrationPolicy{RequireEmailVerification: false}, "a@b.com", "active"},
		{"no email -> active", &RegistrationPolicy{RequireEmailVerification: true}, "", "active"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.p.InitialUserStatus(tc.email))
		})
	}
}
