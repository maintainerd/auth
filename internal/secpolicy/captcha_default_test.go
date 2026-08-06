package secpolicy

import "testing"

// Captcha is deferred to a later release: no first-party registration form emits
// a captcha_token. Seeding this true meant that the moment an operator set
// CAPTCHA_SECRET, every tenant carrying the seeded default rejected 100% of
// self-service registration, with no console lever to turn it back off.
//
// The feature may ship later; until it does, the shipped default must be off.
func TestSeededDefaultDisablesCaptchaOnSignup(t *testing.T) {
	reg, ok := DefaultSecuritySettingConfig("registration")
	if !ok {
		t.Fatal("registration defaults missing")
	}
	v, present := reg["captcha_on_signup"]
	if !present {
		t.Fatal("captcha_on_signup must be present in the seeded defaults")
	}
	if v != false {
		t.Fatalf("captcha_on_signup must default to false while captcha is unshipped, got %v", v)
	}
}
