package iam

import (
	"context"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoalesceAuthEventService(t *testing.T) {
	t.Run("keeps provided service", func(t *testing.T) {
		provided := authevent.NoopService()

		got := coalesceAuthEventService(provided)

		assert.Equal(t, provided, got)
	})

	t.Run("uses noop service when nil", func(t *testing.T) {
		got := coalesceAuthEventService(nil)

		require.NotNil(t, got)
		got.Log(context.Background(), authevent.AuthEventInput{})
		got.Shutdown()
	})
}
