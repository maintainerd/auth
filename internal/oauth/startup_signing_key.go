package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	jwtplatform "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"gorm.io/gorm"
)

// EnsureGlobalSigningKeyFromDB ensures an RS256 signing key exists for the
// global scope (tenant_id IS NULL). Intended for startup when JWT_PRIVATE_KEY
// env var is not set. If the signing_keys table already has an active global
// key its private key is loaded into the JWT key store. Otherwise a fresh
// RSA-2048 key pair is generated, persisted in the DB, and installed.
func EnsureGlobalSigningKeyFromDB(ctx context.Context, db *gorm.DB) error {
	repo := NewSigningKeyRepository(db)
	return ensureGlobalSigningKey(ctx, repo)
}

// installSigningKey is indirected so the platform lane can swap in a
// rotate-preserving installer (one that promotes the current active key to the
// verification-only set instead of discarding it) without this file changing,
// and so tests can observe what was installed.
var installSigningKey = jwtplatform.InstallKeyFromPrivatePEM

// RotateGlobalSigningKey mints a fresh global RS256 key, PERSISTS it to
// signing_keys, installs it as the active signing key, and retires rows old
// enough that nothing they signed can still be valid.
//
// The key-rotation runner calls jwt.RotateKeys(), which only swaps the process's
// in-memory key. On the DB-backed path — the self-host default whenever
// JWT_PRIVATE_KEY is unset — that leaves signing_keys holding the boot key while
// tokens are signed by an unpersisted one, so the published key set does not
// contain the key in use; and it makes every replica sign under a private kid
// that no other replica and no restart can ever verify. Rotation has to go
// through the store the key set is published from, which is what this does.
//
// Superseded keys are NOT retired immediately: FindActiveByTenantID (and
// therefore JWKS) only returns active rows, so retiring on the spot would pull
// the previous key out of the published set while tokens signed with it are
// still inside their lifetime. They age out after signingKeyRetentionWindow
// instead. GetActiveSigningKey/ensureGlobalSigningKey take the newest row, so
// keeping the old ones active does not affect which key signs.
func RotateGlobalSigningKey(ctx context.Context, db *gorm.DB) error {
	return rotateGlobalSigningKey(ctx, NewSigningKeyRepository(db))
}

// signingKeyRetentionWindow is how long a superseded key stays published. It
// matches the refresh-token TTL, which is the longest any token this server
// issued can remain valid, mirroring keyStore.rotate's own pruning rule.
var signingKeyRetentionWindow = jwtplatform.RefreshTokenTTL

func rotateGlobalSigningKey(ctx context.Context, repo SigningKeyRepository) error {
	previous, err := repo.FindActiveByTenantID(0)
	if err != nil {
		return fmt.Errorf("signing key: query DB: %w", err)
	}

	_, privPEM, pubPEM, err := generateGlobalKeyMaterial()
	if err != nil {
		return err
	}

	storedKey, kekID, err := encryptPrivateKeyForStorage(privPEM)
	if err != nil {
		return err
	}

	kid := jwtplatform.GenerateSecureID()
	if err := repo.Create(&SigningKey{
		KID:                 kid,
		Algorithm:           "RS256",
		Use:                 "sig",
		Status:              "active",
		PublicKeyPEM:        pubPEM,
		PrivateKeyEncrypted: storedKey,
		KeyEncryptionKeyID:  kekID,
	}); err != nil {
		return fmt.Errorf("signing key: store rotated key: %w", err)
	}

	// Install only AFTER the row is committed. Installing first would leave the
	// process signing with a key no restart and no other replica could load —
	// exactly the failure this function exists to remove.
	if err := installSigningKey(privPEM, kid); err != nil {
		return fmt.Errorf("signing key: install rotated key: %w", err)
	}

	cutoff := time.Now().Add(-signingKeyRetentionWindow)
	for _, old := range previous {
		if old.KID == kid || old.CreatedAt.After(cutoff) {
			continue
		}
		if err := repo.RetireByKID(old.KID); err != nil {
			slog.WarnContext(ctx, "superseded signing key could not be retired; JWKS keeps publishing it",
				"kid", old.KID, "error", err)
		}
	}

	slog.InfoContext(ctx, "rotated RS256 signing key", "kid", kid, "superseded", len(previous))
	return nil
}

func generateGlobalKeyMaterial() (*rsa.PrivateKey, []byte, string, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signing key: generate RSA key: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})
	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, nil, "", fmt.Errorf("signing key: marshal public key: %w", err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privKey, privPEM, pubPEM, nil
}

// encryptPrivateKeyForStorage returns the bytes to persist and the KEK id that
// describes them. Storage falls back to plaintext only when APP_ENCRYPTION_KEY
// is absent, which is the same posture ensureGlobalSigningKey already had.
func encryptPrivateKeyForStorage(privPEM []byte) ([]byte, string, error) {
	if kek := config.AppEncryptionKey; len(kek) == 32 {
		encrypted, err := crypto.EncryptBytes(privPEM, kek)
		if err != nil {
			return nil, "", fmt.Errorf("signing key: encrypt private key: %w", err)
		}
		return encrypted, "aes256gcm-v1", nil
	}
	return privPEM, "plaintext", nil
}

func ensureGlobalSigningKey(ctx context.Context, repo SigningKeyRepository) error {
	// tenantID=0: FindActiveByTenantID returns WHERE (tenant_id=0 OR tenant_id IS NULL).
	// No tenant ever has ID 0 (BIGSERIAL), so this effectively matches only global rows.
	keys, err := repo.FindActiveByTenantID(0)
	if err != nil {
		return fmt.Errorf("signing key: query DB: %w", err)
	}

	if len(keys) > 0 {
		k := keys[0]
		if len(k.PrivateKeyEncrypted) == 0 {
			return fmt.Errorf("signing key: global key %q exists in DB but has no stored private key; set JWT_PRIVATE_KEY env var", k.KID)
		}
		privPEM := k.PrivateKeyEncrypted
		if k.KeyEncryptionKeyID == "aes256gcm-v1" {
			kek := config.AppEncryptionKey
			if len(kek) != 32 {
				return fmt.Errorf("signing key: key %q is encrypted but APP_ENCRYPTION_KEY is not set", k.KID)
			}
			decrypted, err := crypto.DecryptBytes(k.PrivateKeyEncrypted, kek)
			if err != nil {
				return fmt.Errorf("signing key: decrypt DB key %q: %w", k.KID, err)
			}
			privPEM = decrypted
		}
		if err := jwtplatform.InstallKeyFromPrivatePEM(privPEM, k.KID); err != nil {
			return fmt.Errorf("signing key: install from DB: %w", err)
		}
		slog.InfoContext(ctx, "loaded RS256 signing key from DB", "kid", k.KID)
		return nil
	}

	// No global key in DB — generate a new RSA-2048 key pair.
	_, privPEM, pubPEM, err := generateGlobalKeyMaterial()
	if err != nil {
		return err
	}

	kid := jwtplatform.GenerateSecureID()

	storedKey, kekID, err := encryptPrivateKeyForStorage(privPEM)
	if err != nil {
		return err
	}

	key := &SigningKey{
		KID:                 kid,
		Algorithm:           "RS256",
		Use:                 "sig",
		Status:              "active",
		PublicKeyPEM:        pubPEM,
		PrivateKeyEncrypted: storedKey,
		KeyEncryptionKeyID:  kekID,
	}
	if err := repo.Create(key); err != nil {
		return fmt.Errorf("signing key: store in DB: %w", err)
	}

	if err := jwtplatform.InstallKeyFromPrivatePEM(privPEM, kid); err != nil {
		return fmt.Errorf("signing key: install generated key: %w", err)
	}

	slog.InfoContext(ctx, "auto-generated RS256 signing key", "kid", kid, "encrypted", kekID == "aes256gcm-v1")
	return nil
}
