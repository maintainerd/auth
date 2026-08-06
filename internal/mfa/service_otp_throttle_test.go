package mfa

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The two step-up OTP throttles were dead code: security.CheckRateLimit only
// READS a counter that security.RecordFailedAttempt writes, and the
// "mfa-sms-step-up:" / "mfa-email-otp-step-up:" keys were never incremented by
// anything, so both checks returned nil forever. These tests pin the property
// that made them dead — a send must BOOK itself against the same key it checks —
// rather than the specific key strings.
func TestCheckAndRecordOTPSend(t *testing.T) {
	swapSeams := func(t *testing.T, check func(string) error, record func(string)) {
		t.Helper()
		origCheck, origRecord := checkMFARateLimit, recordMFARateLimitAttempt
		t.Cleanup(func() { checkMFARateLimit, recordMFARateLimitAttempt = origCheck, origRecord })
		checkMFARateLimit, recordMFARateLimitAttempt = check, record
	}

	t.Run("an allowed send is recorded against the key it checked", func(t *testing.T) {
		var checked, recorded []string
		swapSeams(t,
			func(key string) error { checked = append(checked, key); return nil },
			func(key string) { recorded = append(recorded, key) },
		)

		require.NoError(t, checkAndRecordOTPSend("mfa_sms_step_up", mfaTestUserID, "+15550001111"))

		require.Len(t, checked, 1)
		assert.Equal(t, checked, recorded, "a send that is checked but never counted can never trip the limit")
	})

	t.Run("a throttled send is refused and not counted again", func(t *testing.T) {
		var recorded []string
		swapSeams(t,
			func(string) error { return errors.New("locked") },
			func(key string) { recorded = append(recorded, key) },
		)

		err := checkAndRecordOTPSend("mfa_sms_step_up", mfaTestUserID, "+15550001111")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many verification codes")
		assert.Empty(t, recorded)
	})

	t.Run("the key separates users and recipients", func(t *testing.T) {
		// Per-recipient scoping is what bounds bombing one number through many
		// accounts; per-user scoping is what bounds one account pumping the
		// operator's SMS bill.
		assert.NotEqual(t,
			otpSendThrottleKey("mfa_sms_enroll", 1, "+15550001111"),
			otpSendThrottleKey("mfa_sms_enroll", 2, "+15550001111"))
		assert.NotEqual(t,
			otpSendThrottleKey("mfa_sms_enroll", 1, "+15550001111"),
			otpSendThrottleKey("mfa_sms_enroll", 1, "+15550002222"))
		assert.NotEqual(t,
			otpSendThrottleKey("mfa_sms_enroll", 1, "+15550001111"),
			otpSendThrottleKey("mfa_sms_step_up", 1, "+15550001111"))
	})
}

// EnrollSMS had no throttle at all while sending to an attacker-supplied number,
// which made it the cheapest SMS-pump in the service. The throttle must reject
// before any OTP is generated, stored, or dispatched.
func TestMFAService_EnrollSMS_ThrottledBeforeSending(t *testing.T) {
	origCheck := checkMFARateLimit
	t.Cleanup(func() { checkMFARateLimit = origCheck })
	checkMFARateLimit = func(string) error { return errors.New("locked") }

	db, mock := newMockGormDB(t)
	svc := &mfaService{
		db:           db,
		mfaPhoneRepo: &mockMFAPhoneRepo{},
	}

	err := svc.EnrollSMS(t.Context(), mfaTestUserID, "+15550009999")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many verification codes")
	assertExpectationsMet(t, mock) // no OTP row written, nothing dispatched
}
