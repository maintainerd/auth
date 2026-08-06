package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedSecretManager stands in for a remote provider with per-key outcomes.
type scriptedSecretManager struct {
	values map[string]string
	errs   map[string]error
	calls  int
}

func (m *scriptedSecretManager) GetSecret(key string) ([]byte, error) {
	m.calls++
	if err, ok := m.errs[key]; ok {
		return nil, err
	}
	if v, ok := m.values[key]; ok {
		return []byte(v), nil
	}
	return nil, errors.New("scripted: " + key + ": " + ErrSecretNotFound.Error())
}

func (m *scriptedSecretManager) GetSecretString(key string) (string, error) {
	b, err := m.GetSecret(key)
	return string(b), err
}

// notFound mimics a provider that positively reports a key as absent.
func notFoundErr(key string) error {
	return errors.Join(errors.New("scripted: "+key), ErrSecretNotFound)
}

func useScriptedProvider(t *testing.T, m *scriptedSecretManager) {
	t.Helper()
	saveActiveSecretManager(t)
	activeSecretManager = m
}

// The intended model: put whichever secrets you like in the manager, leave the
// rest in the environment, migrate incrementally.
func TestProviderThenEnvFallback(t *testing.T) {
	t.Run("a value present in the provider wins over the environment", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{values: map[string]string{"DB_PASSWORD": "from-vault"}})
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("DB_PASSWORD", "from-env")

		got, err := LoadSecretString("DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "from-vault", got, "the manager must be authoritative when it has the key")
	})

	t.Run("a key absent from the provider falls back to the environment", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{errs: map[string]error{"DB_PASSWORD": notFoundErr("DB_PASSWORD")}})
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("DB_PASSWORD", "from-env")

		got, err := LoadSecretString("DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "from-env", got)
	})

	// THE important case. If the store is unreachable, "no value" is not a fact
	// — falling back here would boot the service on a stale environment value
	// while the operator believes the manager is authoritative.
	t.Run("a provider OUTAGE never falls back to the environment", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{
			errs: map[string]error{"DB_PASSWORD": errors.New("vault: connection refused")},
		})
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("DB_PASSWORD", "stale-env-value")

		_, err := LoadSecretString("DB_PASSWORD")
		require.Error(t, err, "an unreachable store must fail startup, not silently use the environment")
		assert.NotContains(t, err.Error(), "stale-env-value")
	})

	t.Run("a permission error never falls back either", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{
			errs: map[string]error{"DB_PASSWORD": errors.New("vault: permission denied")},
		})
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("DB_PASSWORD", "stale-env-value")

		_, err := LoadSecretString("DB_PASSWORD")
		require.Error(t, err)
	})

	t.Run("SECRET_STRICT makes the provider authoritative", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{errs: map[string]error{"DB_PASSWORD": notFoundErr("DB_PASSWORD")}})
		t.Setenv("SECRET_STRICT", "true")
		t.Setenv("DB_PASSWORD", "from-env")

		_, err := LoadSecretString("DB_PASSWORD")
		require.Error(t, err, "strict mode must refuse to read the environment")
		assert.Contains(t, err.Error(), "SECRET_STRICT")
	})

	t.Run("missing everywhere is an error naming both sources", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{errs: map[string]error{"DB_PASSWORD": notFoundErr("DB_PASSWORD")}})
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("DB_PASSWORD", "")

		_, err := LoadSecretString("DB_PASSWORD")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("optional secrets fall back the same way", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{errs: map[string]error{"REDIS_PASSWORD": notFoundErr("REDIS_PASSWORD")}})
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("REDIS_PASSWORD", "hunter2")

		got, err := LoadSecretStringOptional("REDIS_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, "hunter2", got)
	})

	t.Run("an optional secret absent from both is empty, not an error", func(t *testing.T) {
		useScriptedProvider(t, &scriptedSecretManager{errs: map[string]error{"REDIS_PASSWORD": notFoundErr("REDIS_PASSWORD")}})
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("REDIS_PASSWORD", "")

		got, err := LoadSecretStringOptional("REDIS_PASSWORD")
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	// A definitive "absent" will not change on retry; only transient remote
	// failures are worth retrying.
	t.Run("a not-found answer is not retried", func(t *testing.T) {
		m := &scriptedSecretManager{errs: map[string]error{"DB_PASSWORD": notFoundErr("DB_PASSWORD")}}
		useScriptedProvider(t, m)
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("DB_PASSWORD", "from-env")

		_, err := LoadSecretString("DB_PASSWORD")
		require.NoError(t, err)
		assert.Equal(t, 1, m.calls, "a definitive absence must not cost three round trips")
	})

	// With SECRET_PROVIDER=env there is no second source to consult.
	t.Run("the env provider does not fall back to itself", func(t *testing.T) {
		useEnvSecretManager(t)
		t.Setenv("SECRET_STRICT", "false")
		t.Setenv("DB_PASSWORD", "")

		_, err := LoadSecretString("DB_PASSWORD")
		require.Error(t, err)
	})
}
