package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeSignature(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		got := computeSignature("top-secret", 1710000000, []byte(`{"event":"user.created"}`))

		assert.Equal(t, "sha256=f40bb5e790bb72011c153b79fcf004c3c9ecf2a21f5ee91c488838329dcc1022", got)
	})
}
