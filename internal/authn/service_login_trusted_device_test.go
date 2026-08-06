package authn

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// mockTrustedDeviceAuthenticator implements both MFAFactorAuthenticator and
// MFATrustedDeviceAuthenticator, since loginMFAChallengeResponse type-asserts
// the injected authenticator for the trusted-device half.
type mockTrustedDeviceAuthenticator struct {
	mockMFAAuthenticator
	// validForTenant records the tenant each TrustedDeviceValid call was scoped
	// to, and only that tenant's token is honoured.
	validForTenant int64
	calls          []int64
}

func (m *mockTrustedDeviceAuthenticator) TrustedDeviceValid(_ context.Context, _, tenantID int64, _ string) (bool, error) {
	m.calls = append(m.calls, tenantID)
	return tenantID == m.validForTenant, nil
}

func (m *mockTrustedDeviceAuthenticator) IssueTrustedDevice(context.Context, int64, int64, string, int) (string, error) {
	return "", nil
}

func (m *mockTrustedDeviceAuthenticator) RevokeTrustedDeviceByToken(context.Context, string) error {
	return nil
}

func trustedDeviceLoginService(t *testing.T, mfaConfig string, auth *mockTrustedDeviceAuthenticator) *loginService {
	t.Helper()
	return &loginService{
		securitySettingRepo: &mockSecuritySettingRepo{
			findDefaultByTenantIDFn: func(int64) (*secpolicy.SecuritySetting, error) {
				return &secpolicy.SecuritySetting{MFAConfig: datatypes.JSON([]byte(mfaConfig))}, nil
			},
		},
		mfaAuthenticator: auth,
	}
}

func TestLoginMFAChallengeResponse_TrustedDeviceIsSubordinateToPolicy(t *testing.T) {
	const tenantID = int64(7)
	enrolledTOTP := func(int64) ([]string, error) { return []string{"totp"}, nil }

	t.Run("optional mode still honours a trusted device", func(t *testing.T) {
		auth := &mockTrustedDeviceAuthenticator{
			mockMFAAuthenticator: mockMFAAuthenticator{enrolledFn: enrolledTOTP},
			validForTenant:       tenantID,
		}
		svc := trustedDeviceLoginService(t, `{"mode":"optional","allowed_methods":["totp"],"trusted_device_period_days":30}`, auth)

		ctx := contextWithTrustedDeviceToken(context.Background(), "trust-me")
		resp, err := svc.loginMFAChallengeResponse(ctx, &User{UserID: 1, UserUUID: uuid.New(), IsTOTPEnabled: true}, tenantID, false)

		require.NoError(t, err)
		assert.Nil(t, resp, "a remembered browser skips MFA when the tenant only makes it optional")
	})

	t.Run("enforced mode overrides a trusted device", func(t *testing.T) {
		// The skip used to return BEFORE the policy was even loaded, so a browser
		// trusted under a lax tenant walked straight past a tenant that
		// hard-requires MFA.
		auth := &mockTrustedDeviceAuthenticator{
			mockMFAAuthenticator: mockMFAAuthenticator{enrolledFn: enrolledTOTP},
			validForTenant:       tenantID,
		}
		svc := trustedDeviceLoginService(t, `{"mode":"enforced","allowed_methods":["totp"],"trusted_device_period_days":30}`, auth)
		initTestJWTKeysService(t)

		ctx := contextWithTrustedDeviceToken(context.Background(), "trust-me")
		resp, err := svc.loginMFAChallengeResponse(ctx, &User{UserID: 1, UserUUID: uuid.New(), IsTOTPEnabled: true}, tenantID, false)

		require.NoError(t, err)
		require.NotNil(t, resp, "enforced MFA must still challenge a remembered browser")
		assert.True(t, resp.MFARequired)
		assert.Empty(t, auth.calls, "the trusted-device lookup should not even be reached under enforced mode")
	})

	t.Run("risk-based step-up overrides a trusted device", func(t *testing.T) {
		// forceStepUp is raised precisely because this login looks wrong; a cookie
		// the attacker already holds must not defeat it.
		auth := &mockTrustedDeviceAuthenticator{
			mockMFAAuthenticator: mockMFAAuthenticator{enrolledFn: enrolledTOTP},
			validForTenant:       tenantID,
		}
		svc := trustedDeviceLoginService(t, `{"mode":"optional","allowed_methods":["totp"],"trusted_device_period_days":30}`, auth)
		initTestJWTKeysService(t)

		ctx := contextWithTrustedDeviceToken(context.Background(), "trust-me")
		resp, err := svc.loginMFAChallengeResponse(ctx, &User{UserID: 1, UserUUID: uuid.New(), IsTOTPEnabled: true}, tenantID, true)

		require.NoError(t, err)
		require.NotNil(t, resp, "a flagged login must still be challenged")
		assert.True(t, resp.MFARequired)
		assert.Empty(t, auth.calls)
	})

	t.Run("the trusted-device lookup is scoped to the login tenant", func(t *testing.T) {
		// Trust granted in tenant A must not carry into tenant B on the same
		// account.
		auth := &mockTrustedDeviceAuthenticator{
			mockMFAAuthenticator: mockMFAAuthenticator{enrolledFn: enrolledTOTP},
			validForTenant:       99, // trusted somewhere else
		}
		svc := trustedDeviceLoginService(t, `{"mode":"optional","allowed_methods":["totp"],"trusted_device_period_days":30}`, auth)
		initTestJWTKeysService(t)

		ctx := contextWithTrustedDeviceToken(context.Background(), "trust-me")
		resp, err := svc.loginMFAChallengeResponse(ctx, &User{UserID: 1, UserUUID: uuid.New(), IsTOTPEnabled: true}, tenantID, false)

		require.NoError(t, err)
		require.NotNil(t, resp, "a token from another tenant must not skip MFA here")
		assert.True(t, resp.MFARequired)
		assert.Equal(t, []int64{tenantID}, auth.calls, "the lookup must be scoped to the tenant being logged into")
	})
}
