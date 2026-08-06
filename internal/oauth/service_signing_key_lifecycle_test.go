package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeSigningKeyRepo struct {
	keys        []SigningKey
	created     []*SigningKey
	retired     []string
	compromised []string
	findErr     error
	createErr   error
	retireErr   error
}

func (r *fakeSigningKeyRepo) FindActiveByTenantID(int64) ([]SigningKey, error) {
	return r.keys, r.findErr
}
func (r *fakeSigningKeyRepo) FindByKID(kid string) (*SigningKey, error) {
	for i := range r.keys {
		if r.keys[i].KID == kid {
			return &r.keys[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (r *fakeSigningKeyRepo) Create(k *SigningKey) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = append(r.created, k)
	// Newest first, matching signingKeyRepository.FindActiveByTenantID's
	// `ORDER BY tenant_id DESC NULLS LAST, created_at DESC`. Appending instead
	// would make GetActiveSigningKey return the boot key in tests while it
	// returns the freshly rotated one in production.
	r.keys = append([]SigningKey{*k}, r.keys...)
	return nil
}
func (r *fakeSigningKeyRepo) RetireByKID(kid string) error {
	if r.retireErr != nil {
		return r.retireErr
	}
	r.retired = append(r.retired, kid)
	return nil
}
func (r *fakeSigningKeyRepo) MarkCompromised(kid string) error {
	r.compromised = append(r.compromised, kid)
	return nil
}

func testPublicKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

// Rotation only ever swapped the process's in-memory key (jwt.RotateKeys), so on
// the DB-backed path signing_keys kept the boot key while tokens were signed by
// an unpersisted one — JWKS served a kid that no longer signed anything.
func TestRotateGlobalSigningKey(t *testing.T) {
	t.Run("the rotated key is persisted before it is installed", func(t *testing.T) {
		var installedKID string
		orig := installSigningKey
		defer func() { installSigningKey = orig }()
		installSigningKey = func(privPEM []byte, kid string) error {
			installedKID = kid
			assert.NotEmpty(t, privPEM)
			return nil
		}

		repo := &fakeSigningKeyRepo{}
		require.NoError(t, rotateGlobalSigningKey(context.Background(), repo))

		require.Len(t, repo.created, 1)
		assert.Equal(t, "active", repo.created[0].Status)
		assert.Equal(t, "RS256", repo.created[0].Algorithm)
		assert.NotEmpty(t, repo.created[0].PublicKeyPEM)
		assert.Equal(t, repo.created[0].KID, installedKID)
	})

	t.Run("a failed persist installs nothing", func(t *testing.T) {
		installed := false
		orig := installSigningKey
		defer func() { installSigningKey = orig }()
		installSigningKey = func([]byte, string) error { installed = true; return nil }

		repo := &fakeSigningKeyRepo{createErr: errors.New("db down")}
		require.Error(t, rotateGlobalSigningKey(context.Background(), repo))
		assert.False(t, installed,
			"installing a key the store does not have would leave the process signing with a kid no restart could load")
	})

	// Retiring on the spot would pull the previous key out of JWKS while tokens
	// signed with it are still inside their lifetime.
	t.Run("a recent superseded key is not retired", func(t *testing.T) {
		orig := installSigningKey
		defer func() { installSigningKey = orig }()
		installSigningKey = func([]byte, string) error { return nil }

		repo := &fakeSigningKeyRepo{keys: []SigningKey{{KID: "old", Status: "active", CreatedAt: nowForTest()}}}
		require.NoError(t, rotateGlobalSigningKey(context.Background(), repo))
		assert.Empty(t, repo.retired)
	})

	t.Run("a key older than the retention window is retired", func(t *testing.T) {
		orig := installSigningKey
		defer func() { installSigningKey = orig }()
		installSigningKey = func([]byte, string) error { return nil }

		stale := nowForTest().Add(-2 * signingKeyRetentionWindow)
		repo := &fakeSigningKeyRepo{keys: []SigningKey{{KID: "old", Status: "active", CreatedAt: stale}}}
		require.NoError(t, rotateGlobalSigningKey(context.Background(), repo))
		assert.Equal(t, []string{"old"}, repo.retired)
	})

	// The periodic runner rotates through jwt.RotateKeys(), which only swaps the
	// process's in-memory key: the kid that signs tokens after a tick exists in no
	// table, so JWKS cannot publish it, a second replica cannot verify it, and a
	// restart cannot load it. Every token signed since the last restart becomes
	// unverifiable. A rotation tick must leave the store agreeing with the signer,
	// which is what routing the tick through Rotate/RotateGlobalSigningKey buys.
	t.Run("one rotation tick leaves the persisted active key as the signing key", func(t *testing.T) {
		var installedKID string
		orig := installSigningKey
		defer func() { installSigningKey = orig }()
		installSigningKey = func(_ []byte, kid string) error { installedKID = kid; return nil }

		repo := &fakeSigningKeyRepo{keys: []SigningKey{{KID: "boot", Status: "active", CreatedAt: nowForTest()}}}
		// Rotate only nil-checks the handle before delegating to the repo, so an
		// empty one is enough to exercise the tick without a database.
		svc := NewKeyRotationService(repo, &gorm.DB{})

		require.NoError(t, svc.Rotate(context.Background()))

		require.Len(t, repo.created, 1)
		rotated := repo.created[0].KID
		assert.NotEqual(t, "boot", rotated)
		assert.Equal(t, rotated, installedKID, "the installed signer must be the row that was persisted")

		active, err := svc.GetActiveSigningKey(context.Background(), 0)
		require.NoError(t, err)
		assert.Equal(t, rotated, active.KID,
			"a restart or a second replica resolves the signer through signing_keys, so the rotated kid has to be the active row")

		jwks, err := svc.ListJWKS(context.Background(), 0)
		require.NoError(t, err)
		published := make([]string, 0, len(jwks))
		for _, k := range jwks {
			published = append(published, k.Kid)
		}
		assert.Contains(t, published, rotated,
			"JWKS is served from signing_keys; a signer missing from it cannot be verified by any relying party")
	})
}

// RetireByKID and MarkCompromised were implemented with zero callers, and the
// seeded security:rotate-keys permission guarded nothing.
func TestKeyRotationService_Lifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("ListKeys never returns private key material", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{keys: []SigningKey{{
			KID: "k1", Algorithm: "RS256", Use: "sig", Status: "active",
			PublicKeyPEM: testPublicKeyPEM(t), PrivateKeyEncrypted: []byte("secret"),
			CreatedAt: nowForTest(),
		}}}
		svc := NewKeyRotationService(repo)

		keys, err := svc.ListKeys(ctx, 0)
		require.NoError(t, err)
		require.Len(t, keys, 1)

		encoded, err := json.Marshal(keys[0])
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "secret")
		assert.NotContains(t, string(encoded), "private")
	})

	// Retiring the only active key empties the published key set while tokens are
	// still being issued, so every verification starts failing.
	t.Run("the last active key cannot be retired", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{keys: []SigningKey{{KID: "only", Status: "active"}}}
		svc := NewKeyRotationService(repo)

		require.Error(t, svc.Retire(ctx, "only"))
		assert.Empty(t, repo.retired)
	})

	t.Run("a superseded key can be retired", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{keys: []SigningKey{
			{KID: "old", Status: "active"}, {KID: "new", Status: "active"},
		}}
		svc := NewKeyRotationService(repo)

		require.NoError(t, svc.Retire(ctx, "old"))
		assert.Equal(t, []string{"old"}, repo.retired)
	})

	// No last-key guard on this path: refusing to disown a KNOWN-compromised key
	// because it is the only one would keep it signing and verifying.
	t.Run("the last key can still be marked compromised", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{keys: []SigningKey{{KID: "only", Status: "active"}}}
		svc := NewKeyRotationService(repo)

		require.NoError(t, svc.MarkCompromised(ctx, "only"))
		assert.Equal(t, []string{"only"}, repo.compromised)
	})

	t.Run("rotation without a database handle fails rather than pretending", func(t *testing.T) {
		svc := NewKeyRotationService(&fakeSigningKeyRepo{})
		require.Error(t, svc.Rotate(ctx))
	})
}

func TestOAuthSigningKeyHandler(t *testing.T) {
	t.Run("retiring the last key is a 409, not a 500", func(t *testing.T) {
		repo := &fakeSigningKeyRepo{keys: []SigningKey{{KID: "only", Status: "active"}}}
		h := NewOAuthSigningKeyHandler(NewKeyRotationService(repo))

		r := withChiParam(httptest.NewRequest(http.MethodPost, "/oauth/signing-keys/only/retire", nil), "kid", "only")
		w := httptest.NewRecorder()
		h.Retire(w, r)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("a missing kid is a 400", func(t *testing.T) {
		h := NewOAuthSigningKeyHandler(NewKeyRotationService(&fakeSigningKeyRepo{}))
		r := httptest.NewRequest(http.MethodPost, "/oauth/signing-keys//retire", nil)
		w := httptest.NewRecorder()
		h.Retire(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// A JWKS must contain every key that can verify a token this server issued. The
// DB rows used to win outright, so once the rotation runner swapped the
// in-memory key the published set no longer contained the signing key.
func TestOAuthDiscoveryHandler_JWKSUnionsMemoryAndDB(t *testing.T) {
	initTestJWTKeysService(t)

	repo := &fakeSigningKeyRepo{keys: []SigningKey{{
		KID: "db-key", Algorithm: "RS256", Use: "sig", Status: "active",
		PublicKeyPEM: testPublicKeyPEM(t), CreatedAt: nowForTest(),
	}}}
	h := NewOAuthDiscoveryHandler(NewKeyRotationService(repo))

	w := httptest.NewRecorder()
	h.JWKS(w, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var doc JWKSResponseDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))

	kids := map[string]bool{}
	for _, k := range doc.Keys {
		kids[k.Kid] = true
	}
	assert.True(t, kids["db-key"], "the DB-backed key must be published")

	memory := jwt.GetAllPublicKeys()
	require.NotEmpty(t, memory)
	assert.True(t, kids[memory[0].KID], "the key actually signing tokens must be published")
}
