package valid

import "testing"

// FuzzValidators ensures the email and phone validators — which run on untrusted
// user input at registration/login — never panic on arbitrary strings.
func FuzzValidators(f *testing.F) {
	for _, seed := range []string{
		"", "user@example.com", "no-at-sign", "@", "a@", "@b.co",
		"a@b", "a b@c.d", "+15551234567", "not a phone", "\x00",
		"𝕏@example.com", "a@" + string(rune(0x202E)) + "b.co",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = IsValidEmail(s)
		_ = IsValidPhoneNumber(s)
	})
}
