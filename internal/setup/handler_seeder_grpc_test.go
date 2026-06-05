package setup

import (
	"context"
	"errors"
	"testing"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSeederGRPCHandler_NewSeederGRPCHandler(t *testing.T) {
	h := NewSeederGRPCHandler(nil)
	require.NotNil(t, h)
}

func TestSeederGRPCHandler_TriggerSeeder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		orig := setupRunSeeders
		setupRunSeeders = func(db *gorm.DB, version string) error {
			assert.Nil(t, db)
			assert.Equal(t, "v0.1.0", version)
			return nil
		}
		t.Cleanup(func() { setupRunSeeders = orig })

		resp, err := NewSeederGRPCHandler(nil).TriggerSeeder(context.Background(), &authv1.TriggerSeederRequest{})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "seeders completed", resp.Message)
	})

	t.Run("error", func(t *testing.T) {
		orig := setupRunSeeders
		setupRunSeeders = func(*gorm.DB, string) error {
			return errors.New("seed error")
		}
		t.Cleanup(func() { setupRunSeeders = orig })

		resp, err := NewSeederGRPCHandler(nil).TriggerSeeder(context.Background(), &authv1.TriggerSeederRequest{})

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}
