package mfa

import (
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMFAService_IssueTrustedDevice(t *testing.T) {
	t.Run("upserts one row and returns a non-empty secret", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "user_trusted_devices"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_trusted_device_id"}).AddRow(int64(1)))
		mock.ExpectCommit()

		svc := &mfaService{db: db}
		raw, err := svc.IssueTrustedDevice(t.Context(), mfaTestUserID, 1, "device-abc", 30)

		require.NoError(t, err)
		assert.NotEmpty(t, raw, "a plaintext trust secret must be returned to the client")
		assertExpectationsMet(t, mock)
	})

	t.Run("no-op when trusted-device period is disabled", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		svc := &mfaService{db: db}

		raw, err := svc.IssueTrustedDevice(t.Context(), mfaTestUserID, 1, "device-abc", 0)

		require.NoError(t, err)
		assert.Empty(t, raw)
		assertExpectationsMet(t, mock) // no DB interaction expected
	})

	t.Run("wraps storage errors", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin().WillReturnError(errors.New("db down"))

		svc := &mfaService{db: db}
		_, err := svc.IssueTrustedDevice(t.Context(), mfaTestUserID, 1, "device-abc", 30)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "trusted device storage failed")
		assertExpectationsMet(t, mock)
	})
}

func TestMFAService_TrustedDeviceValid(t *testing.T) {
	t.Run("valid token returns true and bumps last_seen", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT .* FROM "user_trusted_devices"`).
			WillReturnRows(sqlmock.NewRows([]string{"user_trusted_device_id"}).AddRow(int64(7)))
		mock.ExpectBegin()
		expectMFAUpdate(mock, "user_trusted_devices").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		svc := &mfaService{db: db}
		ok, err := svc.TrustedDeviceValid(t.Context(), mfaTestUserID, mfaTestTenantID, "raw-secret")

		require.NoError(t, err)
		assert.True(t, ok)
		assertExpectationsMet(t, mock)
	})

	t.Run("unknown token returns false", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT .* FROM "user_trusted_devices"`).
			WillReturnError(gorm.ErrRecordNotFound)

		svc := &mfaService{db: db}
		ok, err := svc.TrustedDeviceValid(t.Context(), mfaTestUserID, mfaTestTenantID, "nope")

		require.NoError(t, err)
		assert.False(t, ok)
		assertExpectationsMet(t, mock)
	})

	t.Run("blank token returns false without touching the db", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		svc := &mfaService{db: db}

		ok, err := svc.TrustedDeviceValid(t.Context(), mfaTestUserID, mfaTestTenantID, "   ")

		require.NoError(t, err)
		assert.False(t, ok)
		assertExpectationsMet(t, mock)
	})

	t.Run("lookup is scoped to the tenant", func(t *testing.T) {
		// Trust is granted per tenant. Without the tenant predicate a token
		// issued in a lax tenant skipped MFA in a strict `mode: enforced` one on
		// the same account.
		db, mock := newMockGormDB(t)
		mock.ExpectQuery(`SELECT .* FROM "user_trusted_devices" WHERE .*tenant_id = .*`).
			WillReturnError(gorm.ErrRecordNotFound)

		svc := &mfaService{db: db}
		ok, err := svc.TrustedDeviceValid(t.Context(), mfaTestUserID, mfaTestTenantID, "raw-secret")

		require.NoError(t, err)
		assert.False(t, ok)
		assertExpectationsMet(t, mock)
	})
}

func TestMFAService_RevokeTrustedDeviceByToken(t *testing.T) {
	t.Run("soft-deletes the matching row", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		// Soft delete is issued as UPDATE ... SET deleted_at.
		expectMFAUpdate(mock, "user_trusted_devices").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		svc := &mfaService{db: db}
		err := svc.RevokeTrustedDeviceByToken(t.Context(), "raw-secret")

		require.NoError(t, err)
		assertExpectationsMet(t, mock)
	})

	t.Run("no-op on empty token", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		svc := &mfaService{db: db}

		require.NoError(t, svc.RevokeTrustedDeviceByToken(t.Context(), ""))
		assertExpectationsMet(t, mock)
	})
}

func TestDeviceFingerprint(t *testing.T) {
	// The per-browser deviceID drives the fingerprint: same id → same fp,
	// independent of a shared User-Agent.
	assert.Equal(t, deviceFingerprint(1, "dev-1", "UA-x"), deviceFingerprint(1, "dev-1", "UA-y"))
	// Distinct browsers (distinct device ids) never collide.
	assert.NotEqual(t, deviceFingerprint(1, "dev-1", "UA-x"), deviceFingerprint(1, "dev-2", "UA-x"))
	// Falls back to the User-Agent only when no deviceID is present.
	assert.NotEqual(t, deviceFingerprint(1, "", "UA-x"), deviceFingerprint(1, "", "UA-y"))
	// The fingerprint is bound to the user.
	assert.NotEqual(t, deviceFingerprint(1, "dev-1", "UA-x"), deviceFingerprint(2, "dev-1", "UA-x"))
}

func TestInetOrNil(t *testing.T) {
	assert.Nil(t, inetOrNil(""))
	assert.Nil(t, inetOrNil("   "))
	assert.Nil(t, inetOrNil("not-an-ip"))
	require.NotNil(t, inetOrNil("203.0.113.7"))
	assert.Equal(t, "203.0.113.7", *inetOrNil("203.0.113.7"))
	require.NotNil(t, inetOrNil("2001:db8::1"))
}
