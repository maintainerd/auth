package security

import "testing"

// TestRedactedOTP_DefaultRedacts locks in the security invariant: with no
// explicit dev opt-in (the default in every deployed environment), an OTP code
// is never returned for logging — so OTP material cannot leak into logs.
func TestRedactedOTP_DefaultRedacts(t *testing.T) {
	// devLogOTP reflects the environment at package init; the test environment has
	// MAINTAINERD_DEV_LOG_OTP unset, so redaction must be in force.
	if devLogOTP {
		t.Skip("MAINTAINERD_DEV_LOG_OTP is set in this environment; redaction is intentionally bypassed")
	}
	if got := RedactedOTP("123456"); got != "[redacted]" {
		t.Fatalf("RedactedOTP leaked the code by default: got %q, want %q", got, "[redacted]")
	}
}
