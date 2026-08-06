package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// withControlPlaneConfig sets the control-plane deployment knobs for one test and
// restores them, so a test that turns the control plane on cannot leak a system
// instance into the next test in this package.
func withControlPlaneConfig(t *testing.T, enabled bool, role string) {
	t.Helper()
	origEnabled := config.ControlPlaneEnabled
	origRole := config.InstanceRole
	t.Cleanup(func() {
		config.ControlPlaneEnabled = origEnabled
		config.InstanceRole = origRole
	})
	config.ControlPlaneEnabled = enabled
	config.InstanceRole = role
}

// withSystemControlPlane configures the instance the way core provisions the
// ecosystem's system IAM: control plane on, role system. Tests that exercise a
// core-provisioning RPC need it, because the default posture is standalone.
func withSystemControlPlane(t *testing.T) {
	t.Helper()
	withControlPlaneConfig(t, true, config.InstanceRoleSystem)
}

// mtlsMaterial is a throwaway PKI: one CA that issues both the server's
// certificate and core's client certificate, which is the shape the control
// plane requires.
type mtlsMaterial struct {
	caFile         string
	serverCertFile string
	serverKeyFile  string
	clientCert     tls.Certificate
	caPool         *x509.CertPool
}

func writeMTLSMaterial(t *testing.T) mtlsMaterial {
	t.Helper()
	dir := t.TempDir()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "maintainerd-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	issue := func(commonName string, extKeyUsage []x509.ExtKeyUsage, dnsNames []string, ips []net.IP) ([]byte, []byte) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		template := &x509.Certificate{
			SerialNumber: big.NewInt(time.Now().UnixNano()),
			Subject:      pkix.Name{CommonName: commonName},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:  extKeyUsage,
			DNSNames:     dnsNames,
			IPAddresses:  ips,
		}
		der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
		require.NoError(t, err)
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
			pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	}

	serverCertPEM, serverKeyPEM := issue("localhost",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})
	clientCertPEM, clientKeyPEM := issue("maintainerd-core",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, nil, nil)

	material := mtlsMaterial{
		caFile:         filepath.Join(dir, "ca.pem"),
		serverCertFile: filepath.Join(dir, "server-cert.pem"),
		serverKeyFile:  filepath.Join(dir, "server-key.pem"),
	}
	require.NoError(t, os.WriteFile(material.caFile, caPEM, 0600))
	require.NoError(t, os.WriteFile(material.serverCertFile, serverCertPEM, 0600))
	require.NoError(t, os.WriteFile(material.serverKeyFile, serverKeyPEM, 0600))

	material.clientCert, err = tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	require.NoError(t, err)
	material.caPool = x509.NewCertPool()
	require.True(t, material.caPool.AppendCertsFromPEM(caPEM))
	return material
}

// withTLSConfig points the TLS knobs at test material and restores them.
func withTLSConfig(t *testing.T, certFile, keyFile, caFile string, requireMTLS bool) {
	t.Helper()
	origEnv := config.AppEnv
	origCert := config.GRPCTLSCertFile
	origKey := config.GRPCTLSKeyFile
	origCA := config.GRPCClientCAFile
	origMTLS := config.GRPCRequireMTLS
	t.Cleanup(func() {
		config.AppEnv = origEnv
		config.GRPCTLSCertFile = origCert
		config.GRPCTLSKeyFile = origKey
		config.GRPCClientCAFile = origCA
		config.GRPCRequireMTLS = origMTLS
	})
	config.GRPCTLSCertFile = certFile
	config.GRPCTLSKeyFile = keyFile
	config.GRPCClientCAFile = caFile
	config.GRPCRequireMTLS = requireMTLS
}

// R1. The control plane is opt-in: with CONTROL_PLANE_ENABLED off no socket is
// opened at all. Binding an address the test already owns is the proof — if
// startGRPCServerOn attempted the bind it would fail with "failed to listen",
// so a nil return can only mean it never tried.
func TestControlPlaneDisabled_BindsNoListener(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = occupied.Close() }()
	addr := occupied.Addr().String()

	t.Run("disabled: never binds, so an occupied address is not an error", func(t *testing.T) {
		withControlPlaneConfig(t, false, config.InstanceRoleRegular)
		require.NoError(t, startGRPCServerOn(context.Background(), &Application{}, addr))
	})

	// The inverse, which is what makes the assertion above mean something: with the
	// control plane ON the same call DOES reach the bind and reports the conflict.
	t.Run("enabled: binds, so an occupied address is a listen error", func(t *testing.T) {
		withSystemControlPlane(t)
		material := writeMTLSMaterial(t)
		withTLSConfig(t, material.serverCertFile, material.serverKeyFile, material.caFile, true)

		err := startGRPCServerOn(context.Background(), &Application{}, addr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gRPC failed to listen")
	})
}

// R2. Enabling the control plane makes mutual TLS mandatory, and there is no
// switch left that turns it back off. GRPC_REQUIRE_MTLS=false is the escape
// hatch this test exists to prove is gone.
func TestControlPlaneRequiresMutualTLS(t *testing.T) {
	t.Run("GRPC_REQUIRE_MTLS=false cannot downgrade the control plane", func(t *testing.T) {
		withSystemControlPlane(t)
		material := writeMTLSMaterial(t)
		withTLSConfig(t, material.serverCertFile, material.serverKeyFile, material.caFile, false)

		tlsConfig, err := loadGRPCTLSConfig()
		require.NoError(t, err)
		require.NotNil(t, tlsConfig)
		assert.Equal(t, tls.RequireAndVerifyClientCert, tlsConfig.ClientAuth)
		assert.NotNil(t, tlsConfig.ClientCAs)
	})

	// Previously a non-production deployment with no cert/key served gRPC in the
	// clear. Inverted: the control plane refuses rather than fall back to plaintext.
	t.Run("no server cert refuses to start even outside production", func(t *testing.T) {
		withSystemControlPlane(t)
		withTLSConfig(t, "", "", "", false)
		config.AppEnv = "development"

		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CONTROL_PLANE_ENABLED=true")
	})

	t.Run("missing client CA refuses to start", func(t *testing.T) {
		withSystemControlPlane(t)
		material := writeMTLSMaterial(t)
		withTLSConfig(t, material.serverCertFile, material.serverKeyFile, "", false)

		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GRPC_CLIENT_CA_FILE")
		assert.Contains(t, err.Error(), "CONTROL_PLANE_ENABLED=true")
	})

	t.Run("unreadable client CA refuses to start", func(t *testing.T) {
		withSystemControlPlane(t)
		material := writeMTLSMaterial(t)
		withTLSConfig(t, material.serverCertFile, material.serverKeyFile,
			filepath.Join(t.TempDir(), "absent-ca.pem"), false)

		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read gRPC client CA file")
	})

	// An unparsable CA yields an EMPTY pool, and RequireAndVerifyClientCert with an
	// empty pool rejects every peer — a control plane that boots "secure" and
	// refuses everyone looks like a network fault, not a config error.
	t.Run("client CA with no certificate refuses to start", func(t *testing.T) {
		withSystemControlPlane(t)
		material := writeMTLSMaterial(t)
		junk := filepath.Join(t.TempDir(), "junk-ca.pem")
		require.NoError(t, os.WriteFile(junk, []byte("not a certificate"), 0600))
		withTLSConfig(t, material.serverCertFile, material.serverKeyFile, junk, false)

		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no PEM certificate")
	})

	// The assertion at the point the credentials are installed, independent of how
	// loadGRPCTLSConfig behaves.
	t.Run("server options refuse a control plane without verified client certs", func(t *testing.T) {
		withSystemControlPlane(t)
		withTLSConfig(t, "", "", "", false)
		config.AppEnv = "development"

		_, err := grpcServerOptions(&Application{})
		require.Error(t, err)
	})
}

// R2 at runtime: a real listener with the control plane on rejects a client that
// presents no certificate and serves one whose certificate the configured CA
// issued.
func TestControlPlaneMutualTLSHandshake(t *testing.T) {
	withSystemControlPlane(t)
	material := writeMTLSMaterial(t)
	withTLSConfig(t, material.serverCertFile, material.serverKeyFile, material.caFile, false)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveGRPC(ctx, &Application{}, lis) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	dial := func(t *testing.T, creds credentials.TransportCredentials) error {
		t.Helper()
		conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(creds))
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()
		callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer callCancel()
		_, err = grpc_health_v1.NewHealthClient(conn).Check(callCtx, &grpc_health_v1.HealthCheckRequest{})
		return err
	}

	t.Run("a client with no certificate is refused", func(t *testing.T) {
		err := dial(t, credentials.NewTLS(&tls.Config{RootCAs: material.caPool, ServerName: "localhost", MinVersion: tls.VersionTLS12}))
		require.Error(t, err)
	})

	t.Run("a client with a CA-issued certificate is served", func(t *testing.T) {
		err := dial(t, credentials.NewTLS(&tls.Config{
			RootCAs:      material.caPool,
			Certificates: []tls.Certificate{material.clientCert},
			ServerName:   "localhost",
			MinVersion:   tls.VersionTLS12,
		}))
		require.NoError(t, err)
	})
}

// R4. Core-provisioning RPCs are served only by the ecosystem's SYSTEM instance.
func TestInstanceRoleGate(t *testing.T) {
	provisioning := grpcMethod(authv1.TenantService_ServiceDesc.ServiceName, "CreateTenant")

	t.Run("a regular instance refuses a core-provisioning RPC", func(t *testing.T) {
		withControlPlaneConfig(t, true, config.InstanceRoleRegular)
		err := authorizeInstanceRole(provisioning)
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "INSTANCE_ROLE")
		assert.Contains(t, status.Convert(err).Message(), provisioning)
	})

	t.Run("the system instance serves it", func(t *testing.T) {
		withSystemControlPlane(t)
		require.NoError(t, authorizeInstanceRole(provisioning))
	})

	// Fail closed: a role nobody recognises, and the zero value seen before Init
	// has run, are both "not the system instance".
	for _, role := range []string{"", "System", "sYstem ", "primary", "SYSTEM_INSTANCE"} {
		t.Run("an unrecognised role "+role+" is refused", func(t *testing.T) {
			withControlPlaneConfig(t, true, role)
			err := authorizeInstanceRole(provisioning)
			require.Error(t, err, "role %q must not be read as the system instance", role)
			assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		})
	}

	// The listener should not exist at all in this state; the interceptor refuses
	// anyway rather than depend on that being true.
	t.Run("a system role with the control plane off is still refused", func(t *testing.T) {
		withControlPlaneConfig(t, false, config.InstanceRoleSystem)
		err := authorizeInstanceRole(provisioning)
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "CONTROL_PLANE_ENABLED")
	})

	t.Run("peer-service reads stay available on a regular instance", func(t *testing.T) {
		withControlPlaneConfig(t, true, config.InstanceRoleRegular)
		for method := range grpcPeerServiceMethods {
			assert.NoError(t, authorizeInstanceRole(method), "%s is a peer-service read", method)
		}
	})

	// Core bootstraps every instance it provisions, so the setup surface is not
	// system-only — except RegisterControlService, which installs the ecosystem's
	// own control principal and means nothing on a developer's instance.
	t.Run("bootstrap stays open on a regular instance except RegisterControlService", func(t *testing.T) {
		withControlPlaneConfig(t, true, config.InstanceRoleRegular)
		require.NoError(t, authorizeInstanceRole(grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CreateTenant")))
		require.NoError(t, authorizeInstanceRole(grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "CreateAdmin")))

		err := authorizeInstanceRole(grpcMethod(authv1.SetupService_ServiceDesc.ServiceName, "RegisterControlService"))
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})

	t.Run("every classified core-provisioning method is system-only", func(t *testing.T) {
		for method := range grpcServicePermissions {
			service := grpcServiceOfMethod(method)
			if _, core := grpcCoreProvisioningServices[service]; !core {
				continue
			}
			if _, exempt := grpcPeerServiceMethods[method]; exempt {
				// A peer-service exemption must never be a mutation.
				assert.Empty(t, grpcServicePermissions[method], "%s is exempt from the system-instance gate but is PDP-gated, so it is not a peer read", method)
				continue
			}
			assert.True(t, grpcRequiresSystemInstance(method), "%s is on a core-provisioning service but is not system-only", method)
		}
	})

	t.Run("an unparsable method path matches no service", func(t *testing.T) {
		assert.Empty(t, grpcServiceOfMethod("no-leading-slash/Method"))
		assert.Empty(t, grpcServiceOfMethod("/NoMethodSegment"))
		assert.Empty(t, grpcServiceOfMethod("//Method"))
		assert.Equal(t, authv1.TenantService_ServiceDesc.ServiceName, grpcServiceOfMethod(provisioning))
	})
}

// The gate runs ahead of authentication, so on a regular instance the RPC is
// refused with a message that tells core it reached the wrong instance rather
// than an Unauthenticated that reads like a credential problem — and the handler
// never runs.
func TestInstanceRoleGate_RefusesBeforeAuthentication(t *testing.T) {
	withControlPlaneConfig(t, true, config.InstanceRoleRegular)
	method := grpcMethod(authv1.ClientService_ServiceDesc.ServiceName, "CreateClient")

	_, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(5, time.Minute), method)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))

	handlerRan := false
	interceptor := grpcAuthUnaryInterceptor(&Application{}, newGRPCLimiter(5, time.Minute))
	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method}, func(context.Context, any) (any, error) {
		handlerRan = true
		return "ok", nil
	})
	require.Error(t, err)
	assert.False(t, handlerRan)
}
