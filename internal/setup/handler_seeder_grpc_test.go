package setup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeederHandler_NewSeederHandler(t *testing.T) {
	h := NewSeederHandler(nil)
	require.NotNil(t, h)
}
