package dpop

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	key := newTestP256Key(t)
	accessToken := "resource-token"
	proof := signDPoPProof(t, key, proofOptions{
		jti:    "jti-middleware",
		htm:    http.MethodGet,
		htu:    "https://api.example.com/resource",
		iat:    time.Now(),
		ath:    AccessTokenHash(accessToken),
		typ:    "dpop+jwt",
		header: ecJWK(t, &key.PublicKey),
	})

	tests := []struct {
		name          string
		proof         string
		auth          string
		requestURL    func(r *http.Request) string
		wantStatus    int
		wantClaims    bool
		wantNextCalls int
	}{
		{
			name:          "no proof passes through without claims",
			wantStatus:    http.StatusAccepted,
			wantNextCalls: 1,
		},
		{
			name:          "valid proof stores claims",
			proof:         proof,
			auth:          "DPoP " + accessToken,
			requestURL:    func(*http.Request) string { return "https://api.example.com/resource" },
			wantStatus:    http.StatusAccepted,
			wantClaims:    true,
			wantNextCalls: 1,
		},
		{
			name:       "invalid proof returns 401",
			proof:      proof,
			auth:       "DPoP wrong-token",
			requestURL: func(*http.Request) string { return "https://api.example.com/resource" },
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalls := 0
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalls++
				if tt.wantClaims {
					require.NotNil(t, ClaimsFromContext(r.Context()))
				} else {
					assert.Nil(t, ClaimsFromContext(r.Context()))
				}
				w.WriteHeader(http.StatusAccepted)
			})
			handler := Middleware(nil, tt.requestURL)(next)

			req := httptest.NewRequest(http.MethodGet, "/resource", nil)
			if tt.proof != "" {
				req.Header.Set("DPoP", tt.proof)
			}
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNextCalls, nextCalls)
			if tt.wantStatus == http.StatusUnauthorized {
				assert.Contains(t, rec.Body.String(), "invalid_dpop_proof")
			}
		})
	}
}

func TestClaimsFromContext_ReturnsNilForWrongType(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Nil(t, ClaimsFromContext(req.Context()))
}
