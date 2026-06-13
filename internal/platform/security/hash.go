package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// BcryptCost is the work factor used for all bcrypt hashes. 12 meets the
// minimum recommended by OWASP (2023) and satisfies SOC2/ISO27001 requirements.
const BcryptCost = 12

const (
	passwordHashEnvelope = "$maintainerd$"
	passwordSaltSize     = 16
	passwordKeySize      = 32
)

// HashPassword hashes a password using bcrypt. The caller's ctx is propagated
// into the tracing span.
func HashPassword(ctx context.Context, password []byte) ([]byte, error) {
	_, span := otel.Tracer("security").Start(ctx, "security.hash_password")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword(bcryptInput(password), BcryptCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash password failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return hash, nil
}

// HashPasswordWithPolicy hashes a password using the tenant-configured storage
// algorithm. Bcrypt remains the legacy/default-compatible format; newer KDFs
// use a self-describing envelope so ComparePassword can verify all formats.
func HashPasswordWithPolicy(ctx context.Context, password []byte, policy PasswordPolicy) ([]byte, error) {
	algorithm := strings.ToLower(strings.TrimSpace(policy.HashAlgorithm))
	if algorithm == "" || algorithm == "bcrypt" {
		return HashPassword(ctx, password)
	}

	_, span := otel.Tracer("security").Start(ctx, "security.hash_password_with_policy")
	defer span.End()

	hash, err := hashPasswordKDF(password, algorithm)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash password failed")
		return nil, err
	}
	span.SetStatus(codes.Ok, "")
	return []byte(hash), nil
}

// HashClientSecret hashes a client secret with bcrypt. Client secrets are
// high-entropy random strings, but bcrypt ensures the hash is useless even if
// the database is compromised.
func HashClientSecret(ctx context.Context, secret string) (string, error) {
	_, span := otel.Tracer("security").Start(ctx, "security.hash_client_secret")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword(bcryptInput([]byte(secret)), BcryptCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "hash client secret failed")
		return "", err
	}
	span.SetStatus(codes.Ok, "")
	return string(hash), nil
}

// passwordHashFormat classifies a stored hash by inspecting a cheap prefix so
// the comparison can select exactly one algorithm family. Branching on the
// stored format (rather than try-KDF-then-try-bcrypt with early returns) avoids
// disclosing which algorithm is stored through response timing.
type passwordHashFormat int

const (
	hashFormatBcrypt passwordHashFormat = iota // bcrypt ("$2...") and unknown formats
	hashFormatKDF                              // maintainerd KDF envelope ("$maintainerd$...")
)

func detectPasswordHashFormat(stored string) passwordHashFormat {
	if strings.HasPrefix(stored, passwordHashEnvelope) {
		return hashFormatKDF
	}
	// bcrypt hashes start with "$2"; any other/legacy value is verified on the
	// bcrypt path, which fails closed for malformed input.
	return hashFormatBcrypt
}

// ComparePassword validates passwords hashed by HashPassword. It also accepts
// legacy raw bcrypt hashes so existing users are not locked out during rollout.
// The stored hash format is detected once and a single algorithm family is run,
// so the comparison does not leak the stored algorithm via timing.
func ComparePassword(hash []byte, password []byte) bool {
	switch detectPasswordHashFormat(string(hash)) {
	case hashFormatKDF:
		return verifyPasswordKDF(string(hash), password)
	default:
		return compareBcrypt(hash, password)
	}
}

// CompareClientSecret performs a constant-time bcrypt comparison between a
// plaintext client secret and its stored hash, preventing timing attacks. The
// stored hash format is detected once so only a single algorithm family runs.
func CompareClientSecret(plaintext, hash string) bool {
	switch detectPasswordHashFormat(hash) {
	case hashFormatKDF:
		return verifyPasswordKDF(hash, []byte(plaintext))
	default:
		return compareBcrypt([]byte(hash), []byte(plaintext))
	}
}

// compareBcrypt verifies a secret against a bcrypt hash, accepting both the
// current sha256-prehashed input and the legacy raw input. Both variants share
// the bcrypt algorithm, so this stays within a single algorithm family.
func compareBcrypt(hash, secret []byte) bool {
	if bcrypt.CompareHashAndPassword(hash, bcryptInput(secret)) == nil {
		return true
	}
	return bcrypt.CompareHashAndPassword(hash, secret) == nil
}

func bcryptInput(input []byte) []byte {
	sum := sha256.Sum256(input)
	return sum[:]
}

func hashPasswordKDF(password []byte, algorithm string) (string, error) {
	salt := make([]byte, passwordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	input := bcryptInput(password)

	var params string
	var key []byte
	switch algorithm {
	case "argon2id":
		const timeCost uint32 = 3
		const memoryKiB uint32 = 64 * 1024
		const threads uint8 = 4
		params = fmt.Sprintf("m=%d,t=%d,p=%d", memoryKiB, timeCost, threads)
		key = argon2.IDKey(input, salt, timeCost, memoryKiB, threads, passwordKeySize)
	case "scrypt":
		const n = 32768
		const r = 8
		const p = 1
		var err error
		params = fmt.Sprintf("n=%d,r=%d,p=%d", n, r, p)
		key, err = scrypt.Key(input, salt, n, r, p, passwordKeySize)
		if err != nil {
			return "", err
		}
	case "pbkdf2":
		const iterations = 600000
		params = fmt.Sprintf("i=%d", iterations)
		key = pbkdf2.Key(input, salt, iterations, passwordKeySize, sha256.New)
	default:
		return "", fmt.Errorf("unsupported password hash algorithm %q", algorithm)
	}

	enc := base64.RawStdEncoding
	return passwordHashEnvelope + algorithm + "$" + params + "$" + enc.EncodeToString(salt) + "$" + enc.EncodeToString(key), nil
}

func verifyPasswordKDF(stored string, password []byte) bool {
	if !strings.HasPrefix(stored, passwordHashEnvelope) {
		return false
	}
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[1] != "maintainerd" {
		return false
	}
	algorithm, params := parts[2], parts[3]
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := enc.DecodeString(parts[5])
	if err != nil {
		return false
	}

	input := bcryptInput(password)
	actual, err := derivePasswordKDF(input, salt, algorithm, parseHashParams(params), len(expected))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func derivePasswordKDF(input, salt []byte, algorithm string, params map[string]int, keyLen int) ([]byte, error) {
	switch algorithm {
	case "argon2id":
		memory := uint32(paramOrDefault(params, "m", 64*1024))
		timeCost := uint32(paramOrDefault(params, "t", 3))
		threads := uint8(paramOrDefault(params, "p", 4))
		return argon2.IDKey(input, salt, timeCost, memory, threads, uint32(keyLen)), nil
	case "scrypt":
		return scrypt.Key(input, salt, paramOrDefault(params, "n", 32768), paramOrDefault(params, "r", 8), paramOrDefault(params, "p", 1), keyLen)
	case "pbkdf2":
		return pbkdf2.Key(input, salt, paramOrDefault(params, "i", 600000), keyLen, sha256.New), nil
	default:
		return nil, fmt.Errorf("unsupported password hash algorithm %q", algorithm)
	}
}

func parseHashParams(raw string) map[string]int {
	out := map[string]int{}
	for _, part := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(value)
		if err == nil && n > 0 {
			out[key] = n
		}
	}
	return out
}

func paramOrDefault(params map[string]int, key string, fallback int) int {
	if v, ok := params[key]; ok && v > 0 {
		return v
	}
	return fallback
}
