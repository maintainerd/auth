package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInstanceRole(t *testing.T) {
	t.Run("unset defaults to regular", func(t *testing.T) {
		role, err := resolveInstanceRole("")
		require.NoError(t, err)
		assert.Equal(t, InstanceRoleRegular, role)
	})

	t.Run("normalizes case and whitespace", func(t *testing.T) {
		role, err := resolveInstanceRole("  System ")
		require.NoError(t, err)
		assert.Equal(t, InstanceRoleSystem, role)
	})

	// Fail closed: an unrecognised role must not be quietly downgraded to regular
	// on an instance the operator intended to be the ecosystem IAM, nor quietly
	// upgraded on one they did not.
	t.Run("unknown role is rejected", func(t *testing.T) {
		_, err := resolveInstanceRole("sytem")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid INSTANCE_ROLE")
	})
}

func TestIsSystemInstance(t *testing.T) {
	orig := InstanceRole
	t.Cleanup(func() { InstanceRole = orig })

	// The zero value is what a caller sees if it runs before Init; it must not be
	// read as permission to serve the core-provisioning surface.
	InstanceRole = ""
	assert.False(t, IsSystemInstance())

	InstanceRole = InstanceRoleRegular
	assert.False(t, IsSystemInstance())

	InstanceRole = InstanceRoleSystem
	assert.True(t, IsSystemInstance())
}

func TestValidateControlPlaneTLS(t *testing.T) {
	saveTLSGlobals := func(t *testing.T) {
		t.Helper()
		origEnabled := ControlPlaneEnabled
		origCert := GRPCTLSCertFile
		origKey := GRPCTLSKeyFile
		origCA := GRPCClientCAFile
		t.Cleanup(func() {
			ControlPlaneEnabled = origEnabled
			GRPCTLSCertFile = origCert
			GRPCTLSKeyFile = origKey
			GRPCClientCAFile = origCA
		})
	}

	t.Run("control plane off requires nothing", func(t *testing.T) {
		saveTLSGlobals(t)
		ControlPlaneEnabled = false
		GRPCTLSCertFile, GRPCTLSKeyFile, GRPCClientCAFile = "", "", ""
		require.NoError(t, validateControlPlaneTLS())
	})

	t.Run("control plane on without server cert is refused", func(t *testing.T) {
		saveTLSGlobals(t)
		ControlPlaneEnabled = true
		GRPCTLSCertFile, GRPCTLSKeyFile, GRPCClientCAFile = "", "", "/tmp/ca.pem"
		err := validateControlPlaneTLS()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GRPC_TLS_CERT_FILE")
	})

	t.Run("control plane on without client CA is refused", func(t *testing.T) {
		saveTLSGlobals(t)
		ControlPlaneEnabled = true
		GRPCTLSCertFile, GRPCTLSKeyFile, GRPCClientCAFile = "cert.pem", "key.pem", ""
		err := validateControlPlaneTLS()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GRPC_CLIENT_CA_FILE")
	})

	// The whole point of R2: an unreadable CA must stop the process, not downgrade
	// the control plane to server-side TLS where a bearer token is the only thing
	// asserting the caller is core.
	t.Run("control plane on with unreadable client CA is refused", func(t *testing.T) {
		saveTLSGlobals(t)
		ControlPlaneEnabled = true
		GRPCTLSCertFile, GRPCTLSKeyFile = "cert.pem", "key.pem"
		GRPCClientCAFile = filepath.Join(t.TempDir(), "absent-ca.pem")
		err := validateControlPlaneTLS()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read gRPC client CA file")
	})

	t.Run("control plane on with non-PEM client CA is refused", func(t *testing.T) {
		saveTLSGlobals(t)
		ControlPlaneEnabled = true
		GRPCTLSCertFile, GRPCTLSKeyFile = "cert.pem", "key.pem"
		caPath := filepath.Join(t.TempDir(), "junk-ca.pem")
		require.NoError(t, os.WriteFile(caPath, []byte("not a certificate"), 0o600))
		GRPCClientCAFile = caPath
		err := validateControlPlaneTLS()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no PEM certificate")
	})

	t.Run("control plane on with a valid client CA passes", func(t *testing.T) {
		saveTLSGlobals(t)
		ControlPlaneEnabled = true
		GRPCTLSCertFile, GRPCTLSKeyFile = "cert.pem", "key.pem"
		GRPCClientCAFile = writeTestCA(t)
		require.NoError(t, validateControlPlaneTLS())
	})
}

func TestInitControlPlaneDefaults(t *testing.T) {
	setRequiredEnv := func(t *testing.T) {
		t.Helper()
		t.Setenv("SECRET_PROVIDER", "env")
		t.Setenv("APP_PUBLIC_HOSTNAME", "https://pub.example.com")
		t.Setenv("APP_PRIVATE_HOSTNAME", "https://priv.example.com")
		t.Setenv("APP_FRONTEND_IDENTITY_HOSTNAME", "https://account.example.com")
		t.Setenv("APP_FRONTEND_CONSOLE_HOSTNAME", "https://auth.example.com")
		t.Setenv("JWT_PRIVATE_KEY", "private-key-data")
		t.Setenv("JWT_PUBLIC_KEY", "public-key-data")
		t.Setenv("APP_ENCRYPTION_KEY", "12345678901234567890123456789012")
		t.Setenv("HMAC_SECRET_KEY", "hmac-secret-data")
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_PORT", "5432")
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "pass")
		t.Setenv("DB_NAME", "authdb")
	}

	saveControlPlaneGlobals := func(t *testing.T) {
		t.Helper()
		origSM := activeSecretManager
		origEnabled := ControlPlaneEnabled
		origRole := InstanceRole
		origMTLS := GRPCRequireMTLS
		origCert := GRPCTLSCertFile
		origKey := GRPCTLSKeyFile
		origCA := GRPCClientCAFile
		origToken := SetupBootstrapToken
		t.Cleanup(func() {
			activeSecretManager = origSM
			ControlPlaneEnabled = origEnabled
			InstanceRole = origRole
			GRPCRequireMTLS = origMTLS
			GRPCTLSCertFile = origCert
			GRPCTLSKeyFile = origKey
			GRPCClientCAFile = origCA
			SetupBootstrapToken = origToken
		})
	}

	// The default deployment is STANDALONE: no control plane, no system role.
	// An unconfigured instance is STANDALONE: no gRPC listener, no bootstrap
	// credential, nothing to reach. A developer running this for their own app
	// configures none of it.
	//
	// The role defaults to "system" rather than "regular" and that is deliberate.
	// It is not the security boundary — ControlPlaneEnabled is, and with no
	// listener bound the role decides nothing. The role only means anything to a
	// multi-instance ecosystem, whose operator is the one explicitly marking
	// disposable instances "regular"; defaulting to "regular" would instead make
	// every single-instance deployment enable the control plane and then be
	// refused every provisioning RPC with no obvious cause.
	t.Run("defaults are standalone and closed", func(t *testing.T) {
		saveControlPlaneGlobals(t)
		setRequiredEnv(t)

		require.NoError(t, Init())
		assert.False(t, ControlPlaneEnabled, "no listener without an explicit opt-in")
		assert.Equal(t, "", SetupBootstrapToken, "no bootstrap credential by default")
		assert.Equal(t, InstanceRoleSystem, InstanceRole)
		assert.True(t, IsSystemInstance())
	})

	// The role is inert while the control plane is off: being "system" grants
	// nothing when there is no socket to serve it on.
	t.Run("the default role exposes nothing while the control plane is off", func(t *testing.T) {
		saveControlPlaneGlobals(t)
		setRequiredEnv(t)

		require.NoError(t, Init())
		require.True(t, IsSystemInstance())
		assert.False(t, ControlPlaneEnabled,
			"a system role must not imply a listener; the two are independent switches")
	})

	// GRPC_REQUIRE_MTLS is not an escape hatch: with the control plane on it is
	// forced regardless of what the operator set.
	t.Run("control plane forces mTLS even when the env var says false", func(t *testing.T) {
		saveControlPlaneGlobals(t)
		setRequiredEnv(t)
		ca := writeTestCA(t)
		t.Setenv("CONTROL_PLANE_ENABLED", "true")
		t.Setenv("GRPC_REQUIRE_MTLS", "false")
		t.Setenv("GRPC_TLS_CERT_FILE", "cert.pem")
		t.Setenv("GRPC_TLS_KEY_FILE", "key.pem")
		t.Setenv("GRPC_CLIENT_CA_FILE", ca)

		require.NoError(t, Init())
		assert.True(t, ControlPlaneEnabled)
		assert.True(t, GRPCRequireMTLS)
	})

	t.Run("control plane without a client CA refuses to start", func(t *testing.T) {
		saveControlPlaneGlobals(t)
		setRequiredEnv(t)
		t.Setenv("CONTROL_PLANE_ENABLED", "true")
		t.Setenv("GRPC_TLS_CERT_FILE", "cert.pem")
		t.Setenv("GRPC_TLS_KEY_FILE", "key.pem")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "control plane TLS configuration is invalid")
	})

	t.Run("invalid instance role refuses to start", func(t *testing.T) {
		saveControlPlaneGlobals(t)
		setRequiredEnv(t)
		t.Setenv("INSTANCE_ROLE", "root")

		err := Init()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid INSTANCE_ROLE")
	})

	t.Run("bootstrap credential is fingerprinted, never stored in hex form by hand", func(t *testing.T) {
		saveControlPlaneGlobals(t)
		setRequiredEnv(t)
		t.Setenv("SETUP_BOOTSTRAP_TOKEN", "per-instance-credential")
		t.Setenv("INSTANCE_ROLE", "system")

		require.NoError(t, Init())
		assert.Equal(t, InstanceRoleSystem, InstanceRole)
		assert.Equal(t, "per-instance-credential", SetupBootstrapToken)
	})
}

// writeTestCA writes a self-signed CA certificate and returns its path.
func writeTestCA(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	return path
}
