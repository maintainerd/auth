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

	t.Run("not found when nil", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, nil },
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
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
			"twilio", "AC123", "token123", "+15551234567", "", boolPtr(false),
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
			"vonage", "AC456", "", "+15559876543", "MySender", nil,
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
			"twilio", "AC123", "new-token", "+15551234567", "", boolPtr(true),
		)
		require.NoError(t, err)
		assert.NotEmpty(t, existing.AuthTokenEncrypted)
		assert.NotEqual(t, "new-token", existing.AuthTokenEncrypted)
		dec, decErr := crypto.DecryptAtRest(existing.AuthTokenEncrypted)
		require.NoError(t, decErr)
		assert.Equal(t, "new-token", dec)
		assert.True(t, existing.TestMode)
	})

	t.Run("FindByTenantID error", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, errors.New("db") },
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", "", nil)
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		svc := newSMSConfigSvc(&mockSMSConfigRepo{
			findByTenantIDFn: func(_ int64) (*SMSConfig, error) { return nil, nil },
			createOrUpdateFn: func(_ *SMSConfig) (*SMSConfig, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", "", nil)
		require.Error(t, err)
	})
}
