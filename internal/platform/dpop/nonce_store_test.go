package dpop

import (
	"context"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeNonceStore struct {
	saved      []string
	consumeOK  bool
	consumeErr error
}

func (f *fakeNonceStore) SaveNonce(_, _ int64, nonce string, _ time.Time) error {
	f.saved = append(f.saved, nonce)
	return nil
}

func (f *fakeNonceStore) ConsumeNonce(_ string) (bool, error) {
	return f.consumeOK, f.consumeErr
}

func TestStoreNonceManager_IssueNonce(t *testing.T) {
	store := &fakeNonceStore{}
	m := NewStoreNonceManager(store)
	nonce, err := m.IssueNonce(context.Background(), 1, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, nonce)
	assert.Equal(t, []string{nonce}, store.saved)
}

func TestStoreNonceManager_ConsumeNonce(t *testing.T) {
	t.Run("valid nonce consumed", func(t *testing.T) {
		m := NewStoreNonceManager(&fakeNonceStore{consumeOK: true})
		assert.NoError(t, m.ConsumeNonce(context.Background(), "abc"))
	})
	t.Run("empty nonce is invalid", func(t *testing.T) {
		m := NewStoreNonceManager(&fakeNonceStore{consumeOK: true})
		assert.ErrorIs(t, m.ConsumeNonce(context.Background(), ""), ErrInvalidNonce)
	})
	t.Run("unknown/used/expired nonce is invalid", func(t *testing.T) {
		m := NewStoreNonceManager(&fakeNonceStore{consumeOK: false})
		assert.ErrorIs(t, m.ConsumeNonce(context.Background(), "abc"), ErrInvalidNonce)
	})
}

func TestExtractProofNonce(t *testing.T) {
	makeProof := func(claims jwtlib.MapClaims) string {
		tok := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, claims)
		s, err := tok.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)
		return s
	}

	t.Run("returns nonce claim", func(t *testing.T) {
		proof := makeProof(jwtlib.MapClaims{"nonce": "server-nonce-123", "htm": "POST"})
		assert.Equal(t, "server-nonce-123", ExtractProofNonce(proof))
	})
	t.Run("empty when no nonce claim", func(t *testing.T) {
		proof := makeProof(jwtlib.MapClaims{"htm": "POST"})
		assert.Equal(t, "", ExtractProofNonce(proof))
	})
	t.Run("empty for empty header", func(t *testing.T) {
		assert.Equal(t, "", ExtractProofNonce(""))
	})
	t.Run("empty for garbage", func(t *testing.T) {
		assert.Equal(t, "", ExtractProofNonce("not-a-jwt"))
	})
}
