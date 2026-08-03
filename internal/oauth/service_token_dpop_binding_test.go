package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RFC 9449 §5 — a refresh token issued to a DPoP-proofing client is
// sender-constrained: redemption must require a proof over the SAME key.
//
// The column did not exist, so nothing was ever bound and redemption accepted
// whatever key the caller proved (or no key at all). A stolen refresh token was
// therefore fully spendable by an attacker with their own key — the binding
// existed on the access token only, which is the half that expires in minutes.
func TestExchangeRefreshToken_DPoPBinding(t *testing.T) {
	initTestJWTKeysService(t)
	ctx := context.Background()

	const boundJKT = "bound-key-thumbprint"

	// The client fixture is client_id 10 / identifier "my-client", tenant 1.
	activeToken := func(jkt *string) *OAuthRefreshToken {
		return &OAuthRefreshToken{
			OAuthRefreshTokenID: 1,
			FamilyID:            uuid.New(),
			ClientID:            10,
			UserID:              1,
			TenantID:            1,
			Scope:               []string{"openid"},
			ExpiresAt:           time.Now().Add(time.Hour),
			DPoPJKT:             jkt,
		}
	}

	newSvc := func(t *testing.T, stored *OAuthRefreshToken) (OAuthTokenService, *mockOAuthRefreshTokenRepo) {
		t.Helper()
		db, mock := newMockDB(t)
		expectClientLookup(mock, mockClientRows())
		repo := &mockOAuthRefreshTokenRepo{
			findByTokenHashFn: func(string) (*OAuthRefreshToken, error) { return stored, nil },
		}
		svc := newOAuthTokenSvc(db, &mockClientRepo{}, &mockOAuthAuthCodeRepo{}, repo,
			&mockUserRepo{},
			&mockUserIdentityRepo{findByUserIDAndClientIDFn: func(_, _ int64) (*UserIdentity, error) { return nil, nil }},
			&mockAuthEventService{})
		return svc, repo
	}

	t.Run("a proof over a different key is rejected and burns the family", func(t *testing.T) {
		stored := activeToken(ptr.Ptr(boundJKT))
		svc, repo := newSvc(t, stored)

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:      "refresh_token",
			RefreshToken:   "raw-token",
			DPoPThumbprint: "attacker-key-thumbprint",
		}, OAuthClientCredentials{ClientID: "my-client"})

		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "bound to a different DPoP key")
		// A bound token presented with the wrong key is in hands other than the
		// ones it was issued to — treat it exactly like reuse and burn the family.
		assert.Contains(t, repo.revokedFamilies, stored.FamilyID,
			"a key mismatch must revoke the whole family, not merely fail the call")
	})

	t.Run("a bound token presented with no proof at all is rejected", func(t *testing.T) {
		svc, _ := newSvc(t, activeToken(ptr.Ptr(boundJKT)))

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "raw-token",
		}, OAuthClientCredentials{ClientID: "my-client"})

		require.NotNil(t, oerr)
		assert.Contains(t, oerr.Description, "requires a DPoP proof",
			"omitting the proof must not silently downgrade the token to bearer")
	})

	t.Run("an unbound token is not forced to prove a key", func(t *testing.T) {
		svc, _ := newSvc(t, activeToken(nil))

		_, oerr := svc.Exchange(ctx, OAuthTokenRequestDTO{
			GrantType:    "refresh_token",
			RefreshToken: "raw-token",
		}, OAuthClientCredentials{ClientID: "my-client"})

		// Tokens issued before DPoP, and those from non-proofing clients, must
		// keep working. This exchange may still fail further down the pipeline on
		// unrelated mock plumbing — it just must not fail on DPoP.
		if oerr != nil {
			assert.NotContains(t, oerr.Description, "DPoP")
		}
	})
}
