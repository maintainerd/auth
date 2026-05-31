package setup

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeederHandler_NewSeederHandler(t *testing.T) {
	h := &SeederHandler{}
	require.NotNil(t, h)
}
