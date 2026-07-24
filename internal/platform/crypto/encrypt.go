package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
)

const GCMNonceSize = 12

var EncryptBytes = encryptBytes

func encryptBytes(plaintext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	result := gcm.Seal(nonce, nonce, plaintext, nil) // #nosec G407 -- random nonce prepended to ciphertext
	return result, nil
}

var DecryptBytes = decryptBytes

func decryptBytes(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (AES-256)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	if len(ciphertext) < GCMNonceSize {
		return nil, fmt.Errorf("ciphertext too short (%d bytes)", len(ciphertext))
	}
	nonce, ciphertextOnly := ciphertext[:GCMNonceSize], ciphertext[GCMNonceSize:]
	result, err := gcm.Open(nil, nonce, ciphertextOnly, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return result, nil
}

func EncryptString(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	ct, err := EncryptBytes([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(ct), nil
}

func DecryptString(ciphertext string, key []byte) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt: invalid base64: %w", err)
	}
	pt, err := DecryptBytes(raw, key)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func appEncryptionKey() []byte {
	return config.AppEncryptionKey
}

// atRestKeyPrefix tags a ciphertext with the id of the key that produced it.
//
// Stored values used to be a bare base64 blob with no indication of which key
// encrypted them, so rotating APP_ENCRYPTION_KEY silently made every stored secret
// (client secrets, webhook signing secrets, TOTP seeds) undecryptable — and
// SafeDecryptAtRest then returned the raw ciphertext as if it were the plaintext.
// The tag lets a retired key decrypt its own rows while new writes use the current
// key, which is what makes a rotation possible at all.
//
// Format: "k1:<key-id>:<base64 ciphertext>". Untagged values are legacy and are
// tried against every configured key.
const atRestKeyPrefix = "k1:"

// atRestKeyID is a short, stable fingerprint of a key. It is derived rather than
// configured so no deployment has to name its keys, and it is truncated so the
// stored tag reveals nothing usable about the key itself.
func atRestKeyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:4])
}

// atRestDecryptionKeys is the current key followed by the retired decrypt-only keys.
func atRestDecryptionKeys() [][]byte {
	keys := make([][]byte, 0, 1+len(config.AppEncryptionPreviousKeys))
	if len(config.AppEncryptionKey) > 0 {
		keys = append(keys, config.AppEncryptionKey)
	}
	keys = append(keys, config.AppEncryptionPreviousKeys...)
	return keys
}

var EncryptAtRest = func(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := appEncryptionKey()
	encoded, err := EncryptString(plaintext, key)
	if err != nil {
		return "", err
	}
	return atRestKeyPrefix + atRestKeyID(key) + ":" + encoded, nil
}

var DecryptAtRest = func(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	if keyID, encoded, ok := splitAtRestValue(ciphertext); ok {
		for _, key := range atRestDecryptionKeys() {
			if atRestKeyID(key) == keyID {
				return DecryptString(encoded, key)
			}
		}
		// Naming the missing key id is what makes a botched rotation diagnosable;
		// the id is a truncated hash, so it is safe to surface.
		return "", fmt.Errorf("decrypt: no configured key matches key id %q", keyID)
	}

	// Legacy untagged value: try every key, current first.
	var lastErr error
	for _, key := range atRestDecryptionKeys() {
		plaintext, err := DecryptString(ciphertext, key)
		if err == nil {
			return plaintext, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no encryption key configured")
	}
	return "", fmt.Errorf("decrypt: no configured key could decrypt this value: %w", lastErr)
}

// splitAtRestValue parses the "k1:<key-id>:<ciphertext>" envelope.
func splitAtRestValue(value string) (keyID, encoded string, ok bool) {
	if !strings.HasPrefix(value, atRestKeyPrefix) {
		return "", "", false
	}
	rest := value[len(atRestKeyPrefix):]
	idx := strings.Index(rest, ":")
	if idx <= 0 {
		return "", "", false
	}
	return rest[:idx], rest[idx+1:], true
}

// IsEncryptedAtRest reports whether a stored value carries the key tag, i.e. whether
// it was written by the current envelope format. A re-encryption pass uses this to
// tell "already migrated" from "legacy, needs rewriting".
func IsEncryptedAtRest(value string) bool {
	_, _, ok := splitAtRestValue(value)
	return ok
}

// SafeDecryptAtRest decrypts a stored value, falling back to the value itself when it
// cannot be decrypted.
//
// The fallback exists for rows written before encryption was introduced, which hold
// plaintext. It is a footgun for anything else: a rotation that loses a key would
// hand callers the ciphertext, and a webhook would then sign with it and fail
// verification with no clue why. So a value that IS tagged as encrypted never falls
// back silently — it logs and returns empty, which fails closed.
func SafeDecryptAtRest(value string) string {
	if value == "" {
		return ""
	}
	dec, err := DecryptAtRest(value)
	if err == nil {
		return dec
	}
	if IsEncryptedAtRest(value) {
		slog.Error("failed to decrypt a value encrypted at rest; "+
			"check APP_ENCRYPTION_KEY and APP_ENCRYPTION_KEYS_PREVIOUS", "error", err)
		return ""
	}
	// Untagged and undecryptable: legacy plaintext.
	return value
}
