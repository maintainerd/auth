package dpop

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

const (
	nonceBytes  = 24
	nonceMaxAge = 5 * time.Minute
	nonceGC     = 10 * time.Minute
)

type nonceEntry struct {
	createdAt time.Time
}

type NonceManager struct {
	mu      sync.RWMutex
	nonces  map[string]nonceEntry
	stopGC  chan struct{}
	gcOnce  sync.Once
}

func NewNonceManager() *NonceManager {
	nm := &NonceManager{
		nonces: make(map[string]nonceEntry),
		stopGC: make(chan struct{}),
	}
	nm.startGC()
	return nm
}

func (nm *NonceManager) startGC() {
	go func() {
		ticker := time.NewTicker(nonceGC)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				nm.cleanup()
			case <-nm.stopGC:
				return
			}
		}
	}()
}

func (nm *NonceManager) cleanup() {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	cutoff := time.Now().Add(-nonceMaxAge)
	for k, e := range nm.nonces {
		if e.createdAt.Before(cutoff) {
			delete(nm.nonces, k)
		}
	}
}

func (nm *NonceManager) Stop() {
	nm.gcOnce.Do(func() {
		close(nm.stopGC)
	})
}

func (nm *NonceManager) Generate() string {
	b := make([]byte, nonceBytes)
	_, _ = rand.Read(b)
	nonce := base64.RawURLEncoding.EncodeToString(b)
	nm.mu.Lock()
	nm.nonces[nonce] = nonceEntry{createdAt: time.Now()}
	nm.mu.Unlock()
	return nonce
}

func (nm *NonceManager) Validate(nonce string) bool {
	if nonce == "" {
		return false
	}
	nm.mu.RLock()
	entry, ok := nm.nonces[nonce]
	nm.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Since(entry.createdAt) > nonceMaxAge {
		return false
	}
	return true
}

func (nm *NonceManager) SetNonceHeader(w http.ResponseWriter) {
	w.Header().Set("DPoP-Nonce", nm.Generate())
}
