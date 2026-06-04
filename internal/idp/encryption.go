package idp

import (
	"encoding/json"

	"github.com/maintainerd/auth/internal/platform/crypto"
	"gorm.io/datatypes"
)

const idpClientSecretKey = "client_secret"

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

func redactIdpConfig(config datatypes.JSON) *datatypes.JSON {
	if len(config) == 0 {
		return &config
	}
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return &config
	}
	if _, ok := m[idpClientSecretKey]; ok {
		m[idpClientSecretKey] = "***REDACTED***"
	}
	b, _ := json.Marshal(m)
	result := datatypes.JSON(b)
	return &result
}
