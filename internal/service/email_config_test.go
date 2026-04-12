package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEmailConfigSvc(repo *mockEmailConfigRepo) EmailConfigService {
	return NewEmailConfigService(repo)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestEmailConfigService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ec := &model.EmailConfig{
			EmailConfigUUID: uuid.New(),
			TenantID:        1,
			Provider:        "smtp",
			Host:            "smtp.example.com",
			Port:            587,
			Username:        "user",
			FromAddress:     "noreply@example.com",
			Encryption:      "tls",
			Status:          model.StatusActive,
		}
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return ec, nil },
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, ec.EmailConfigUUID, res.EmailConfigUUID)
		assert.Equal(t, "smtp", res.Provider)
	})

	t.Run("not found when nil", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return nil, nil },
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return nil, errors.New("db") },
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
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return nil, nil },
			createOrUpdateFn: func(e *model.EmailConfig) (*model.EmailConfig, error) {
				e.EmailConfigUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Update(context.Background(), 1,
			"ses", "smtp.ses.amazonaws.com", 587,
			"key", "secret",
			"noreply@example.com", "Acme", "support@example.com",
			"tls", boolPtr(true),
		)
		require.NoError(t, err)
		assert.Equal(t, "ses", res.Provider)
		assert.True(t, res.TestMode)
		assert.Equal(t, model.StatusActive, res.Status)
	})

	t.Run("updates existing with password preserved on blank", func(t *testing.T) {
		existing := &model.EmailConfig{
			EmailConfigUUID:   uuid.New(),
			TenantID:          1,
			PasswordEncrypted: "old-secret",
			Status:            model.StatusActive,
		}
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *model.EmailConfig) (*model.EmailConfig, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1,
			"smtp", "mail.example.com", 465,
			"user", "", // blank password — should be preserved
			"noreply@example.com", "Acme", "",
			"ssl", nil,
		)
		require.NoError(t, err)
		assert.Equal(t, "smtp", res.Provider)
		// Password should remain unchanged (still "old-secret") on the model
		assert.Equal(t, "old-secret", existing.PasswordEncrypted)
	})

	t.Run("updates existing with new password", func(t *testing.T) {
		existing := &model.EmailConfig{
			EmailConfigUUID:   uuid.New(),
			TenantID:          1,
			PasswordEncrypted: "old-secret",
			Status:            model.StatusActive,
		}
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return existing, nil },
			createOrUpdateFn: func(e *model.EmailConfig) (*model.EmailConfig, error) { return e, nil },
		})
		_, err := svc.Update(context.Background(), 1,
			"smtp", "mail.example.com", 465,
			"user", "new-secret",
			"noreply@example.com", "Acme", "",
			"ssl", boolPtr(false),
		)
		require.NoError(t, err)
		assert.Equal(t, "new-secret", existing.PasswordEncrypted)
	})

	t.Run("FindByTenantID error", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return nil, errors.New("db") },
		})
		_, err := svc.Update(context.Background(), 1, "", "", 0, "", "", "", "", "", "", nil)
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		svc := newEmailConfigSvc(&mockEmailConfigRepo{
			findByTenantIDFn: func(_ int64) (*model.EmailConfig, error) { return nil, nil },
			createOrUpdateFn: func(_ *model.EmailConfig) (*model.EmailConfig, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, "", "", 0, "", "", "", "", "", "", nil)
		require.Error(t, err)
	})
}
