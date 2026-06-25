package idp

import (
	"strings"

	"github.com/maintainerd/auth/internal/platform/crypto"
)

// providerClientSecretRedacted is the sentinel the admin form echoes back when
// the operator did not retype the secret. Reads never return the secret anymore
// (the column is simply not selected), but the form may still POST this value,
// so the write path treats it — and a blank value — as "unchanged" and preserves
// the stored secret. This is the write-only secret contract.
const providerClientSecretRedacted = "***REDACTED***"

// encryptProviderClientSecret encrypts a plaintext upstream client secret for
// storage in the provider_client_secret_encrypted column. A blank secret yields
// a nil pointer so providers without upstream credentials (e.g. the system
// provider) store NULL.
func encryptProviderClientSecret(plaintext string) (*string, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, nil
	}
	enc, err := crypto.EncryptAtRest(plaintext)
	if err != nil {
		return nil, err
	}
	return &enc, nil
}

// preserveProviderClientSecret implements the write-only secret contract on
// update. When the incoming plaintext is blank or the redaction sentinel, the
// previously stored (already-encrypted) secret is carried over unchanged.
// Otherwise the new plaintext is encrypted and returned.
func preserveProviderClientSecret(incomingPlaintext string, existingEncrypted *string) (*string, error) {
	incomingPlaintext = strings.TrimSpace(incomingPlaintext)
	if incomingPlaintext == "" || incomingPlaintext == providerClientSecretRedacted {
		return existingEncrypted, nil
	}
	return encryptProviderClientSecret(incomingPlaintext)
}
