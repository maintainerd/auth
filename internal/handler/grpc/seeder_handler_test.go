// NOTE: SeederHandler is a temporary feature created solely to test gRPC
// functionality. It has no production use at this time.
package grpc

import (
	"context"
	"testing"

	authv1 "github.com/maintainerd/auth/internal/gen/go/auth/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeederHandler_NewSeederHandler(t *testing.T) {
	h := NewSeederHandler(nil)
	require.NotNil(t, h)
}

func TestSeederHandler_TriggerSeeder(t *testing.T) {
	h := NewSeederHandler(nil)
	resp, err := h.TriggerSeeder(context.Background(), &authv1.TriggerSeederRequest{})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "Received", resp.Message)
}
