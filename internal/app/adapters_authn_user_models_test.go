package app

import (
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/authn"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

// TestToAuthnUserCopiesMFAFlags guards the login MFA trigger: the authn login
// flow decides whether to challenge for MFA from the user's enrolled factors,
// so toAuthnUser MUST carry the MFA flags across the package boundary. Dropping
// them (the original bug) made userHasAnyMFAFactor always false → no login MFA.
func TestToAuthnUserCopiesMFAFlags(t *testing.T) {
	enabledAt := time.Unix(1_700_000_000, 0).UTC()
	tempExpiresAt := time.Unix(1_700_100_000, 0).UTC()
	src := &user.User{
		UserID:                     7,
		IsTOTPEnabled:              true,
		IsWebAuthnEnabled:          true,
		MFAEnabledAt:               &enabledAt,
		TemporaryPasswordExpiresAt: &tempExpiresAt,
	}

	got := toAuthnUser(src)
	if !got.IsTOTPEnabled {
		t.Error("IsTOTPEnabled not carried to authn.User")
	}
	if !got.IsWebAuthnEnabled {
		t.Error("IsWebAuthnEnabled not carried to authn.User")
	}
	if got.MFAEnabledAt == nil || !got.MFAEnabledAt.Equal(enabledAt) {
		t.Error("MFAEnabledAt not carried to authn.User")
	}
	if got.TemporaryPasswordExpiresAt == nil || !got.TemporaryPasswordExpiresAt.Equal(tempExpiresAt) {
		t.Error("TemporaryPasswordExpiresAt not carried to authn.User")
	}

	// Round-trip back to the user model must preserve them too.
	back := toUserUser(&authn.User{
		IsTOTPEnabled:              true,
		IsWebAuthnEnabled:          true,
		MFAEnabledAt:               &enabledAt,
		TemporaryPasswordExpiresAt: &tempExpiresAt,
	})
	if !back.IsTOTPEnabled || !back.IsWebAuthnEnabled || back.MFAEnabledAt == nil {
		t.Error("toUserUser dropped MFA flags")
	}
	if back.TemporaryPasswordExpiresAt == nil || !back.TemporaryPasswordExpiresAt.Equal(tempExpiresAt) {
		t.Error("toUserUser dropped TemporaryPasswordExpiresAt")
	}
}
