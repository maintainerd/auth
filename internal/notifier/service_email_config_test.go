package notifier

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEmailConfigSvc(repo *mockEmailConfigRepo) EmailConfigService {
	return NewEmailConfigService(repo)
}

func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestEmailConfigService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ec := &EmailConfig{
			EmailConfigUUID: uuid.New(),
			TenantID:        1,
			Provider:        "smtp",
			Host:            "smtp.example.com",
			Port:            587,
			Username:        "user",
			FromAddress:     "noreply@example.com",
			Encryption:      strPtr("tls"),
			Status:          shared.StatusActive,
		}
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return ec, nil },
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, ec.EmailConfigUUID, res.EmailConfigUUID)
		assert.Equal(t, "smtp", res.Provider)
		assert.Equal(t, "tls", res.Encryption)
	})

	t.Run("not found returns defaults", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, nil },
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, "smtp", res.Provider)
		assert.Equal(t, shared.StatusActive, res.Status)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, errors.New("db") },
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db")
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestEmailConfigService_Update(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	t.Run("creates new when not found", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, nil },
			createOrUpdateFn: func(e *EmailConfig) (*EmailConfig, error) {
				e.EmailConfigUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Update(context.Background(), 1,
			"ses", "smtp.ses.amazonaws.com", 587,
			"key", "secret",
			"noreply@example.com", "Acme", "support@example.com",
			"tls", "", boolPtr(true),
		)
		require.NoError(t, err)
		assert.Equal(t, "ses", res.Provider)
		assert.True(t, res.TestMode)
		assert.Equal(t, shared.StatusActive, res.Status)
	})

	t.Run("updates existing with password preserved on blank", func(t *testing.T) {
		existing := &EmailConfig{
			EmailConfigUUID:   uuid.New(),
			TenantID:          1,
			PasswordEncrypted: "old-secret",
			Status:            shared.StatusActive,
		}
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *EmailConfig) (*EmailConfig, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1,
			"smtp", "mail.example.com", 465,
			"user", "", // blank password — should be preserved
			"noreply@example.com", "Acme", "",
			"ssl", "", nil,
		)
		require.NoError(t, err)
		assert.Equal(t, "smtp", res.Provider)
		// Password should remain unchanged (still "old-secret") on the model
		assert.Equal(t, "old-secret", existing.PasswordEncrypted)
	})

	t.Run("updates existing with new password", func(t *testing.T) {
		existing := &EmailConfig{
			EmailConfigUUID:   uuid.New(),
			TenantID:          1,
			PasswordEncrypted: "old-secret",
			Status:            shared.StatusActive,
		}
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *EmailConfig) (*EmailConfig, error) { return e, nil },
		})
		_, err := svc.Update(context.Background(), 1,
			"smtp", "mail.example.com", 465,
			"user", "new-secret",
			"noreply@example.com", "Acme", "",
			"ssl", "", boolPtr(false),
		)
		require.NoError(t, err)
		assert.NotEmpty(t, existing.PasswordEncrypted)
		assert.NotEqual(t, "new-secret", existing.PasswordEncrypted)
		pw, decErr := crypto.DecryptAtRest(existing.PasswordEncrypted)
		require.NoError(t, decErr)
		assert.Equal(t, "new-secret", pw)
	})

	t.Run("blank encryption is stored as NULL not empty string", func(t *testing.T) {
		var saved *EmailConfig
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, nil },
			createOrUpdateFn: func(e *EmailConfig) (*EmailConfig, error) {
				saved = e
				e.EmailConfigUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Update(context.Background(), 1,
			"smtp", "smtp.gmail.com", 587,
			"user@example.com", "secret",
			"noreply@example.com", "Acme", "",
			"", "", nil, // blank encryption must not violate the CHECK constraint
		)
		require.NoError(t, err)
		// nil pointer => GORM writes SQL NULL, which the encryption CHECK allows.
		assert.Nil(t, saved.Encryption)
		assert.Equal(t, "", res.Encryption)
	})

	t.Run("non-empty encryption is persisted", func(t *testing.T) {
		var saved *EmailConfig
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, nil },
			createOrUpdateFn: func(e *EmailConfig) (*EmailConfig, error) {
				saved = e
				return e, nil
			},
		})
		res, err := svc.Update(context.Background(), 1,
			"smtp", "smtp.example.com", 465,
			"user", "secret",
			"noreply@example.com", "Acme", "",
			"ssl", "", nil,
		)
		require.NoError(t, err)
		require.NotNil(t, saved.Encryption)
		assert.Equal(t, "ssl", *saved.Encryption)
		assert.Equal(t, "ssl", res.Encryption)
	})

	t.Run("clears secret when provider changes without a new secret", func(t *testing.T) {
		existing := &EmailConfig{
			EmailConfigUUID:   uuid.New(),
			TenantID:          1,
			Provider:          "smtp",
			PasswordEncrypted: "old-smtp-secret",
			Status:            shared.StatusActive,
		}
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *EmailConfig) (*EmailConfig, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1,
			"sendgrid", "", 0,
			"", "", // blank password while switching providers
			"noreply@example.com", "Acme", "",
			"", "", nil,
		)
		require.NoError(t, err)
		assert.Equal(t, "sendgrid", res.Provider)
		// The old SMTP secret must not carry over to the new provider.
		assert.Empty(t, existing.PasswordEncrypted)
	})

	t.Run("encrypt password error", func(t *testing.T) {
		original := crypto.EncryptAtRest
		t.Cleanup(func() { crypto.EncryptAtRest = original })
		crypto.EncryptAtRest = func(string) (string, error) { return "", errors.New("encrypt failed") }
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, nil },
		})

		_, err := svc.Update(context.Background(), 1,
			"smtp", "mail.example.com", 465,
			"user", "new-secret",
			"noreply@example.com", "Acme", "",
			"ssl", "", boolPtr(false),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypt failed")
	})

	t.Run("FindByTenantID error", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, errors.New("db") },
		})
		_, err := svc.Update(context.Background(), 1, "", "", 0, "", "", "", "", "", "", "", nil)
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*EmailConfig, error) { return nil, nil },
			createOrUpdateFn: func(_ *EmailConfig) (*EmailConfig, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, "", "", 0, "", "", "", "", "", "", "", nil)
		require.Error(t, err)
	})
}
