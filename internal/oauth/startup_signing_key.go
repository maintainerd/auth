package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"

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
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("signing key: generate RSA key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return fmt.Errorf("signing key: marshal public key: %w", err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	kid := jwtplatform.GenerateSecureID()

	storedKey := privPEM
	kekID := "plaintext"
	if kek := config.AppEncryptionKey; len(kek) == 32 {
		encrypted, encErr := crypto.EncryptBytes(privPEM, kek)
		if encErr != nil {
			return fmt.Errorf("signing key: encrypt private key: %w", encErr)
		}
		storedKey = encrypted
		kekID = "aes256gcm-v1"
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
