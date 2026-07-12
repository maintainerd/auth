package geoip

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopResolver(t *testing.T) {
	r := NewNoop()
	loc, ok := r.Lookup("8.8.8.8")
	assert.False(t, ok)
	assert.Empty(t, loc)
	assert.NoError(t, r.Close())
}

func TestNew_EmptyPathIsNoop(t *testing.T) {
	// No DB configured → feature disabled, no error.
	r, err := New("   ")
	require.NoError(t, err)
	loc, ok := r.Lookup("8.8.8.8")
	assert.False(t, ok)
	assert.Empty(t, loc)
}

func TestNew_UnreadablePathDegradesGracefully(t *testing.T) {
	// A configured-but-missing DB must not fail startup: return a no-op + error.
	r, err := New("/nonexistent/GeoLite2-City.mmdb")
	require.Error(t, err)
	require.NotNil(t, r)
	_, ok := r.Lookup("8.8.8.8")
	assert.False(t, ok)
}
