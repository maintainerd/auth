package mfa

import (
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTOTPAndStep(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	at := time.Unix(1_700_000_000, 0).UTC()
	code, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      1,
		Digits:    totpDigits,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)

	step, ok, err := validateTOTPAndStep(code, secret, at)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, at.Unix()/totpPeriod, step)
}

func TestValidateTOTPAndStepRejectsInvalidCode(t *testing.T) {
	step, ok, err := validateTOTPAndStep("000000", "JBSWY3DPEHPK3PXP", time.Unix(1_700_000_000, 0).UTC())
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, step)
}
