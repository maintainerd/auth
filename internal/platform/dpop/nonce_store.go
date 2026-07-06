package dpop

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	// storeNonceBytes is the size of a server-issued DPoP nonce (RFC 9449 §8).
	storeNonceBytes = 32
	// storeNonceTTL is the lifetime of a server-issued nonce.
	storeNonceTTL = 5 * time.Minute
)

// ErrInvalidNonce is returned by ConsumeNonce when the presented nonce is
// unknown, already used, or expired.
var ErrInvalidNonce = errors.New("dpop: invalid or expired nonce")

// NonceStore persists server-issued, single-use DPoP nonces. It is implemented
// by the oauth_dpop_nonces repository and injected at the composition root, so
// this platform package stays domain-agnostic. The method signatures match the
// repository exactly, so the repository satisfies this interface directly.
type NonceStore interface {
	// SaveNonce persists a freshly issued nonce for a client with a TTL.
	SaveNonce(tenantID, clientID int64, nonce string, expiresAt time.Time) error
	// ConsumeNonce atomically validates and marks a nonce used. Returns ok=false
	// when the nonce is unknown, already used, or expired.
	ConsumeNonce(nonce string) (ok bool, err error)
}

// StoreNonceManager issues and consumes DB-backed, single-use, per-client DPoP
// server nonces (RFC 9449 §8), unlike the in-memory NonceManager. It is used by
// the token endpoint's nonce gate for DPoP-required clients.
type StoreNonceManager struct {
	store NonceStore
}

// NewStoreNonceManager creates a store-backed nonce manager.
func NewStoreNonceManager(store NonceStore) *StoreNonceManager {
	return &StoreNonceManager{store: store}
}

// IssueNonce generates a fresh 32-byte base64url nonce, persists it for the
// client with a 5-minute TTL, and returns it for the DPoP-Nonce response header.
func (m *StoreNonceManager) IssueNonce(_ context.Context, tenantID, clientID int64) (string, error) {
	b := make([]byte, storeNonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(b)
	if err := m.store.SaveNonce(tenantID, clientID, nonce, time.Now().Add(storeNonceTTL)); err != nil {
		return "", err
	}
	return nonce, nil
}

// ConsumeNonce validates and single-use-consumes a nonce. Returns
// ErrInvalidNonce when the nonce is unknown, already used, or expired.
func (m *StoreNonceManager) ConsumeNonce(_ context.Context, nonce string) error {
	if nonce == "" {
		return ErrInvalidNonce
	}
	ok, err := m.store.ConsumeNonce(nonce)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidNonce
	}
	return nil
}

// ExtractProofNonce reads the `nonce` claim from a DPoP proof JWT without
// verifying its signature. The nonce's security derives from being
// server-issued and single-use (validated against NonceStore), so reading it
// unverified here is safe — the proof signature is validated separately.
// Returns "" when the proof is absent or has no nonce claim.
func ExtractProofNonce(proofHeader string) string {
	if proofHeader == "" {
		return ""
	}
	unverified, _, err := new(jwtlib.Parser).ParseUnverified(proofHeader, jwtlib.MapClaims{})
	if err != nil {
		return ""
	}
	mc, ok := unverified.Claims.(jwtlib.MapClaims)
	if !ok {
		return ""
	}
	nonce, _ := mc["nonce"].(string)
	return nonce
}
