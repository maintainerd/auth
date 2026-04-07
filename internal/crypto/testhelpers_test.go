package crypto

import (
	"crypto/rand"
	"errors"
	"testing"
)

// errReader is a reader that always returns an error.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("forced crypto/rand failure")
}

func withFailingRand(t *testing.T) {
	t.Helper()
	orig := rand.Reader
	rand.Reader = errReader{}
	t.Cleanup(func() { rand.Reader = orig })
}
