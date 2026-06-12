package idp

import (
	"encoding/json"
	"strings"

	"github.com/maintainerd/auth/internal/platform/crypto"
	"gorm.io/datatypes"
)

const idpClientSecretKey = "client_secret"

// idpClientSecretRedacted is the placeholder GET responses return in place of
// the stored secret. The form echoes it back on save when the operator did not
// retype the secret, so it must be treated as "unchanged" on write.
const idpClientSecretRedacted = "***REDACTED***"

func encryptIdpConfig(config datatypes.JSON) (datatypes.JSON, error) {
	if len(config) == 0 {
		return config, nil
	}
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return config, err
	}
	if v, ok := m[idpClientSecretKey]; ok {
		if s, ok := v.(string); ok && s != "" {
			enc, err := crypto.EncryptAtRest(s)
			if err != nil {
				return nil, err
			}
			m[idpClientSecretKey] = enc
		}
	}
	b, _ := json.Marshal(m)
	return datatypes.JSON(b), nil
}

func decryptIdpConfig(config datatypes.JSON) datatypes.JSON {
	if len(config) == 0 {
		return config
	}
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return config
	}
	if v, ok := m[idpClientSecretKey]; ok {
		if s, ok := v.(string); ok && s != "" {
			m[idpClientSecretKey] = crypto.SafeDecryptAtRest(s)
		}
	}
	b, _ := json.Marshal(m)
	return datatypes.JSON(b)
}

// preserveIdpClientSecret implements the write-only secret contract on update.
// `incoming` is the already-encrypted config built from the request; `existing`
// is the currently stored (encrypted) config. When the request omits the secret,
// leaves it blank, or echoes back the redaction sentinel, the previously stored
// secret is carried over so a routine edit never clobbers it. A genuinely new
// secret (already encrypted by encryptIdpConfig) is left untouched.
func preserveIdpClientSecret(incoming datatypes.JSON, existing datatypes.JSON) datatypes.JSON {
	var in map[string]any
	if len(incoming) == 0 || json.Unmarshal(incoming, &in) != nil {
		return incoming
	}

	v, ok := in[idpClientSecretKey]
	s, _ := v.(string)
	provided := ok && strings.TrimSpace(s) != "" && s != idpClientSecretRedacted
	if provided {
		return incoming
	}

	// No usable secret in the request — carry over the stored one if present.
	if len(existing) > 0 {
		var ex map[string]any
		if json.Unmarshal(existing, &ex) == nil {
			if exVal, exOk := ex[idpClientSecretKey].(string); exOk && exVal != "" {
				in[idpClientSecretKey] = exVal
				b, _ := json.Marshal(in)
				return datatypes.JSON(b)
			}
		}
	}

	// Nothing to preserve; drop the empty/redacted placeholder entirely.
	delete(in, idpClientSecretKey)
	b, _ := json.Marshal(in)
	return datatypes.JSON(b)
}

func redactIdpConfig(config datatypes.JSON) *datatypes.JSON {
	if len(config) == 0 {
		return &config
	}
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return &config
	}
	if v, ok := m[idpClientSecretKey]; ok {
		// Redact any set value; only an explicit empty string is left as-is so the
		// form can distinguish "no secret configured" from "secret hidden".
		if s, isStr := v.(string); !isStr || s != "" {
			m[idpClientSecretKey] = idpClientSecretRedacted
		}
	}
	b, _ := json.Marshal(m)
	result := datatypes.JSON(b)
	return &result
}
