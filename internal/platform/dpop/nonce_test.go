package dpop

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonceManager(t *testing.T) {
	nm := NewNonceManager()
	t.Cleanup(func() {
		nm.Stop()
		time.Sleep(10 * time.Millisecond)
	})

	nonce := nm.Generate()
	require.NotEmpty(t, nonce)
	assert.True(t, nm.Validate(nonce))
	assert.False(t, nm.Validate(""))
	assert.False(t, nm.Validate("missing"))

	nm.mu.Lock()
	nm.nonces[nonce] = nonceEntry{createdAt: time.Now().Add(-6 * time.Minute)}
	nm.mu.Unlock()
	assert.False(t, nm.Validate(nonce))

	expired := nm.Generate()
	fresh := nm.Generate()
	nm.mu.Lock()
	nm.nonces[expired] = nonceEntry{createdAt: time.Now().Add(-6 * time.Minute)}
	nm.mu.Unlock()

	nm.cleanup()

	nm.mu.RLock()
	_, hasExpired := nm.nonces[expired]
	_, hasFresh := nm.nonces[fresh]
	nm.mu.RUnlock()
	assert.False(t, hasExpired)
	assert.True(t, hasFresh)

	rec := httptest.NewRecorder()
	nm.SetNonceHeader(rec)
	headerNonce := rec.Header().Get("DPoP-Nonce")
	require.NotEmpty(t, headerNonce)
	assert.True(t, nm.Validate(headerNonce))

	nm.Stop()
	assert.NotPanics(t, nm.Stop)

	time.Sleep(10 * time.Millisecond)
}

func TestNonceManager_GCStop(t *testing.T) {
	nm := NewNonceManager()

	nm.Stop()
	time.Sleep(10 * time.Millisecond)
	assert.NotPanics(t, nm.Stop)
}
