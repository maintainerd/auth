package notifier

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSMSConfigSvc(repo *mockSMSConfigRepo) SMSConfigService {
	return NewSMSConfigService(repo)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestSMSConfigService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		sc := &SMSConfig{
			SMSConfigUUID: uuid.New(),
			TenantID:      1,
			Provider:      "twilio",
			AccountSID:    "AC123",
			FromNumber:    "+15551234567",
			Status:        shared.StatusActive,
		}
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return sc, nil },
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, sc.SMSConfigUUID, res.SMSConfigUUID)
		assert.Equal(t, "twilio", res.Provider)
	})

	t.Run("not found returns defaults", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, nil },
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, 1000, res.DailySendLimit)
		assert.Equal(t, shared.StatusActive, res.Status)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, errors.New("db") },
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestSMSConfigService_Update(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	t.Run("creates new when not found", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, nil },
			createOrUpdateFn: func(e *SMSConfig) (*SMSConfig, error) {
				e.SMSConfigUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Update(context.Background(), 1,
			"twilio", "AC123", "token123", "+15551234567", "", nil, boolPtr(false),
		)
		require.NoError(t, err)
		assert.Equal(t, "twilio", res.Provider)
		assert.Equal(t, shared.StatusActive, res.Status)
	})

	t.Run("updates existing with auth_token preserved on blank", func(t *testing.T) {
		existing := &SMSConfig{
			SMSConfigUUID:      uuid.New(),
			TenantID:           1,
			AuthTokenEncrypted: "old-token",
			Status:             shared.StatusActive,
		}
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *SMSConfig) (*SMSConfig, error) { return e, nil },
		})
		_, err := svc.Update(context.Background(), 1,
			"vonage", "AC456", "", "+15559876543", "MySender", nil, nil,
		)
		require.NoError(t, err)
		assert.Equal(t, "old-token", existing.AuthTokenEncrypted)
	})

	t.Run("updates existing with new auth_token", func(t *testing.T) {
		existing := &SMSConfig{
			SMSConfigUUID:      uuid.New(),
			TenantID:           1,
			AuthTokenEncrypted: "old-token",
			Status:             shared.StatusActive,
		}
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *SMSConfig) (*SMSConfig, error) { return e, nil },
		})
		_, err := svc.Update(context.Background(), 1,
			"twilio", "AC123", "new-token", "+15551234567", "", nil, boolPtr(true),
		)
		require.NoError(t, err)
		assert.NotEmpty(t, existing.AuthTokenEncrypted)
		assert.NotEqual(t, "new-token", existing.AuthTokenEncrypted)
		dec, decErr := crypto.DecryptAtRest(existing.AuthTokenEncrypted)
		require.NoError(t, decErr)
		assert.Equal(t, "new-token", dec)
		assert.True(t, existing.TestMode)
	})

	t.Run("clears secret when provider changes without a new secret", func(t *testing.T) {
		existing := &SMSConfig{
			SMSConfigUUID:      uuid.New(),
			TenantID:           1,
			Provider:           "twilio",
			AuthTokenEncrypted: "old-twilio-token",
			Status:             shared.StatusActive,
		}
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *SMSConfig) (*SMSConfig, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1,
			"vonage", "key456", "", "+15559876543", "MySender", nil, nil, // blank token while switching providers
		)
		require.NoError(t, err)
		assert.Equal(t, "vonage", res.Provider)
		// The old Twilio token must not carry over to the new provider.
		assert.Empty(t, existing.AuthTokenEncrypted)
	})

	t.Run("encrypt auth token error", func(t *testing.T) {
		original := crypto.EncryptAtRest
		t.Cleanup(func() { crypto.EncryptAtRest = original })
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failed") }
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, nil },
		})

		_, err := svc.Update(context.Background(), 1,
			"twilio", "AC123", "token123", "+15551234567", "", nil, boolPtr(false),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypt failed")
	})

	t.Run("FindByTenantID error", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, errors.New("db") },
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", "", nil, nil)
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, nil },
			createOrUpdateFn: func(_ *SMSConfig) (*SMSConfig, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", "", nil, nil)
		require.Error(t, err)
	})
}
