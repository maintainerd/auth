package user

import (
	"testing"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/stretchr/testify/assert"
)

func TestCoalesceAuthEventService(t *testing.T) {
	t.Run("nil returns NoopService", func(t *testing.T) {
		svc := coalesceAuthEventService(nil)
		assert.Equal(t, authevent.NoopService(), svc)
	})

	t.Run("non-nil returns same instance", func(t *testing.T) {
		custom := authevent.NoopService()
		svc := coalesceAuthEventService(custom)
		assert.Equal(t, custom, svc)
	})
}
