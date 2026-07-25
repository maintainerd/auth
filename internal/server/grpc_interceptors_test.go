package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestGRPCLimiter(t *testing.T) {
	limiter := newGRPCLimiter(2, time.Minute)
	now := time.Now()
	assert.True(t, limiter.Allow("svc", now))
	assert.True(t, limiter.Allow("svc", now.Add(time.Second)))
	assert.False(t, limiter.Allow("svc", now.Add(2*time.Second)))
	assert.True(t, limiter.Allow("svc", now.Add(2*time.Minute)))
}

func TestGRPCInterceptors_BasicBranches(t *testing.T) {
	const openMethod = "/test.Service/Open"
	grpcServicePermissions[openMethod] = ""
	t.Cleanup(func() { delete(grpcServicePermissions, openMethod) })

	t.Run("recovery unary converts panic", func(t *testing.T) {
		interceptor := grpcRecoveryUnaryInterceptor()
		_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Panic"}, func(context.Context, any) (any, error) {
			panic("boom")
		})
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("recovery stream converts panic", func(t *testing.T) {
		interceptor := grpcRecoveryStreamInterceptor()
		err := interceptor(nil, &testServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/svc/Panic"}, func(any, grpc.ServerStream) error {
			panic("boom")
		})
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("logging and timeout pass through", func(t *testing.T) {
		logging := grpcLoggingUnaryInterceptor()
		timeout := grpcTimeoutUnaryInterceptor(time.Second)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcRequestIDKey, "req-123"))
		resp, err := logging(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Ok"}, func(ctx context.Context, req any) (any, error) {
			assert.Equal(t, "req-123", grpcRequestIDFromContext(ctx))
			return timeout(ctx, req, &grpc.UnaryServerInfo{FullMethod: "/svc/Ok"}, func(ctx context.Context, _ any) (any, error) {
				_, ok := ctx.Deadline()
				assert.True(t, ok)
				return "ok", nil
			})
		})
		require.NoError(t, err)
		assert.Equal(t, "ok", resp)
	})

	t.Run("auth rejects missing metadata", func(t *testing.T) {
		_, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(1, time.Minute), openMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("auth accepts service token", func(t *testing.T) {
		initServerTestJWTKeys(t)
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "auth",
			SubjectType: "service",
			AMR:         []string{"client_credentials"},
			ACR:         "1",
			SessionID:   "session-1",
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

		authCtx, err := authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(1, time.Minute), openMethod)
		require.NoError(t, err)
		claims := middleware.JWTClaimsFromContext(authCtx)
		require.NotNil(t, claims)
		assert.Equal(t, "auth", claims.Service)
		assert.Equal(t, "service", claims.SubjectType)
		assert.Equal(t, []string{"client_credentials"}, claims.AMR)
		assert.Equal(t, "1", claims.ACR)
	})

	t.Run("auth rejects malformed bearer", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "basic nope"))
		_, err := authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(1, time.Minute), openMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("auth rejects non-service token", func(t *testing.T) {
		initServerTestJWTKeys(t)
		token, err := jwt.GenerateAccessTokenWithOptions("user-1", "read", "https://auth.example.com", "auth", "client-1", "provider-1", nil)
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

		_, err = authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(1, time.Minute), openMethod)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("auth rate limits principal", func(t *testing.T) {
		initServerTestJWTKeys(t)
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "auth",
			SubjectType: "service",
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
		limiter := newGRPCLimiter(1, time.Minute)
		_, err = authenticateAndAuthorizeGRPC(ctx, &Application{}, limiter, openMethod)
		require.NoError(t, err)
		_, err = authenticateAndAuthorizeGRPC(ctx, &Application{}, limiter, openMethod)
		assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	})

	t.Run("auth unary interceptor returns auth error", func(t *testing.T) {
		interceptor := grpcAuthUnaryInterceptor(&Application{}, newGRPCLimiter(1, time.Minute))
		_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: openMethod}, func(context.Context, any) (any, error) {
			return "unexpected", nil
		})

		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("auth unary interceptor passes auth context to handler", func(t *testing.T) {
		initServerTestJWTKeys(t)
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "auth",
			SubjectType: "service",
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
		interceptor := grpcAuthUnaryInterceptor(&Application{}, newGRPCLimiter(1, time.Minute))

		resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: openMethod}, func(ctx context.Context, _ any) (any, error) {
			require.NotNil(t, middleware.JWTClaimsFromContext(ctx))
			return "ok", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "ok", resp)
	})

	t.Run("auth stream interceptor returns auth error", func(t *testing.T) {
		interceptor := grpcAuthStreamInterceptor(&Application{}, newGRPCLimiter(1, time.Minute))
		err := interceptor(nil, &testServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: openMethod}, func(any, grpc.ServerStream) error {
			return nil
		})

		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("authz denies protected permission", func(t *testing.T) {
		initServerTestJWTKeys(t)
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "auth",
			SubjectType: "service",
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
		grpcServicePermissions["/test.Service/NeedsPermission"] = "tenant:read"
		t.Cleanup(func() { delete(grpcServicePermissions, "/test.Service/NeedsPermission") })

		_, err = authenticateAndAuthorizeGRPC(ctx, &Application{AuthorizationService: mockGRPCAuthz{allowed: false}}, newGRPCLimiter(2, time.Minute), "/test.Service/NeedsPermission")
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("authz allows protected permission", func(t *testing.T) {
		initServerTestJWTKeys(t)
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "auth",
			SubjectType: "service",
			// A service token must carry its tenant: the policy lookup resolves the
			// principal by name, which is unique per tenant, not globally.
			ExtraClaims: map[string]any{"tenant_id": 1},
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
		grpcServicePermissions["/test.Service/NeedsPermission"] = "tenant:read"
		t.Cleanup(func() { delete(grpcServicePermissions, "/test.Service/NeedsPermission") })

		authCtx, err := authenticateAndAuthorizeGRPC(ctx, &Application{AuthorizationService: mockGRPCAuthz{allowed: true}}, newGRPCLimiter(2, time.Minute), "/test.Service/NeedsPermission")
		require.NoError(t, err)
		assert.NotNil(t, middleware.JWTClaimsFromContext(authCtx))
	})

	// Without a tenant the policy lookup resolved the principal across all tenants
	// and returned the lowest service_id — the system tenant's service — collapsing
	// every service principal onto the platform's own policies.
	t.Run("authz denies a service token with no tenant", func(t *testing.T) {
		initServerTestJWTKeys(t)
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "auth",
			SubjectType: "service",
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
		grpcServicePermissions["/test.Service/NeedsPermission"] = "tenant:read"
		t.Cleanup(func() { delete(grpcServicePermissions, "/test.Service/NeedsPermission") })

		_, err = authenticateAndAuthorizeGRPC(ctx, &Application{AuthorizationService: mockGRPCAuthz{allowed: true}}, newGRPCLimiter(2, time.Minute), "/test.Service/NeedsPermission")
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("step-up denies protected method without elevated acr", func(t *testing.T) {
		initServerTestJWTKeys(t)
		const method = "/test.Service/NeedsStepUp"
		grpcServicePermissions[method] = ""
		grpcStepUpMethods[method] = struct{}{}
		t.Cleanup(func() {
			delete(grpcServicePermissions, method)
			delete(grpcStepUpMethods, method)
		})
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			Service:     "auth",
			SubjectType: "service",
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

		_, err = authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(2, time.Minute), method)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("step-up allows protected method with elevated acr", func(t *testing.T) {
		initServerTestJWTKeys(t)
		const method = "/test.Service/NeedsStepUp"
		grpcServicePermissions[method] = ""
		grpcStepUpMethods[method] = struct{}{}
		t.Cleanup(func() {
			delete(grpcServicePermissions, method)
			delete(grpcStepUpMethods, method)
		})
		token, err := jwt.GenerateAccessTokenWithOptions("svc-auth", "read", "https://auth.example.com", "auth", "client-1", "provider-1", &jwt.AccessTokenOptions{
			ACR:         jwt.ACRLevel2,
			Service:     "auth",
			SubjectType: "service",
		})
		require.NoError(t, err)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))

		authCtx, err := authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(2, time.Minute), method)
		require.NoError(t, err)
		assert.Equal(t, jwt.ACRLevel2, middleware.JWTClaimsFromContext(authCtx).ACR)
	})

	t.Run("unknown method skips auth", func(t *testing.T) {
		ctx, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(1, time.Minute), "/unknown.Service/Call")
		require.NoError(t, err)
		assert.NotNil(t, ctx)
	})

	t.Run("helpers", func(t *testing.T) {
		assert.Equal(t, []string{"a"}, stringSliceClaim([]any{"a", 1}))
		assert.Nil(t, stringSliceClaim("a"))
		assert.Equal(t, "sub:subj", grpcPrincipalKey(nilClaims("subj", "", "")))
		assert.Equal(t, "client:client", grpcPrincipalKey(nilClaims("subj", "", "client")))
		assert.Equal(t, "svc:svc", grpcPrincipalKey(nilClaims("subj", "svc", "client")))

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer invalid"))
		_, err := grpcJWTClaims(ctx)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))

		logGRPC(peer.NewContext(context.Background(), &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1")}}), "/svc/Ok", time.Now(), nil)
		logGRPC(peer.NewContext(context.Background(), &peer.Peer{Addr: testAddr("pipe")}), "/svc/Ok", time.Now(), nil)
	})

	t.Run("request id helpers accept aliases and synthesize missing id", func(t *testing.T) {
		assert.Equal(t, "req-a", grpcRequestIDFromMetadata(metadata.NewIncomingContext(context.Background(), metadata.Pairs("request-id", " req-a "))))
		assert.Equal(t, "req-b", grpcRequestIDFromMetadata(metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-correlation-id", "req-b"))))
		assert.Equal(t, "req-c", grpcRequestIDFromMetadata(metadata.NewIncomingContext(context.Background(), metadata.Pairs("correlation-id", "req-c"))))
		assert.Empty(t, grpcRequestIDFromMetadata(metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcRequestIDKey, " "))))

		ctx, requestID := grpcContextWithRequestID(context.Background())
		assert.NotEmpty(t, requestID)
		assert.Equal(t, requestID, grpcRequestIDFromContext(ctx))
	})

	t.Run("stream logging timeout and auth interceptors pass through", func(t *testing.T) {
		stream := &testServerStream{ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcRequestIDKey, "stream-req"))}
		logging := grpcLoggingStreamInterceptor()
		timeout := grpcTimeoutStreamInterceptor(time.Second)
		auth := grpcAuthStreamInterceptor(&Application{}, newGRPCLimiter(1, time.Minute))

		require.NoError(t, logging(nil, stream, &grpc.StreamServerInfo{FullMethod: "/unknown.Service/Stream"}, func(_ any, logged grpc.ServerStream) error {
			return timeout(nil, logged, &grpc.StreamServerInfo{FullMethod: "/unknown.Service/Stream"}, func(_ any, wrapped grpc.ServerStream) error {
				assert.Equal(t, "stream-req", grpcRequestIDFromContext(wrapped.Context()))
				_, ok := wrapped.Context().Deadline()
				assert.True(t, ok)
				return auth(nil, wrapped, &grpc.StreamServerInfo{FullMethod: "/unknown.Service/Stream"}, func(_ any, authed grpc.ServerStream) error {
					assert.NotNil(t, authed.Context())
					return nil
				})
			})
		}))
	})
}

func TestGRPCInterceptors_BootstrapAndDefaultDeny(t *testing.T) {
	const bootstrapMethod = "/maintainerd.auth.v1.SetupService/CreateTenant"
	const ghostAppMethod = "/maintainerd.auth.v1.GhostService/DoThing"

	origToken := config.SetupBootstrapToken
	t.Cleanup(func() { config.SetupBootstrapToken = origToken })

	t.Run("app-prefixed unlisted method is default-denied", func(t *testing.T) {
		_, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(1, time.Minute), ghostAppMethod)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("non-app unknown method still passes (health/reflection)", func(t *testing.T) {
		ctx, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(1, time.Minute), "/grpc.health.v1.Health/Check")
		require.NoError(t, err)
		assert.NotNil(t, ctx)
	})

	t.Run("bootstrap denied when token unset", func(t *testing.T) {
		config.SetupBootstrapToken = ""
		_, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(1, time.Minute), bootstrapMethod)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("bootstrap rejects missing token", func(t *testing.T) {
		config.SetupBootstrapToken = "s3cr3t-bootstrap"
		_, err := authenticateAndAuthorizeGRPC(context.Background(), &Application{}, newGRPCLimiter(1, time.Minute), bootstrapMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("bootstrap rejects wrong token", func(t *testing.T) {
		config.SetupBootstrapToken = "s3cr3t-bootstrap"
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcSetupTokenKey, "nope"))
		_, err := authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(1, time.Minute), bootstrapMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("bootstrap accepts valid token", func(t *testing.T) {
		config.SetupBootstrapToken = "s3cr3t-bootstrap"
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcSetupTokenKey, "s3cr3t-bootstrap"))
		_, err := authenticateAndAuthorizeGRPC(ctx, &Application{}, newGRPCLimiter(2, time.Minute), bootstrapMethod)
		require.NoError(t, err)
	})

	t.Run("bootstrap rate limits", func(t *testing.T) {
		config.SetupBootstrapToken = "s3cr3t-bootstrap"
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcSetupTokenKey, "s3cr3t-bootstrap"))
		limiter := newGRPCLimiter(1, time.Minute)
		_, err := authenticateAndAuthorizeGRPC(ctx, &Application{}, limiter, bootstrapMethod)
		require.NoError(t, err)
		_, err = authenticateAndAuthorizeGRPC(ctx, &Application{}, limiter, bootstrapMethod)
		assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
}

func TestGRPCTLSConfig(t *testing.T) {
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

	t.Run("development without cert allows plaintext", func(t *testing.T) {
		config.AppEnv = "development"
		config.GRPCTLSCertFile = ""
		config.GRPCTLSKeyFile = ""
		cfg, err := loadGRPCTLSConfig()
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("production without cert fails closed", func(t *testing.T) {
		config.AppEnv = "production"
		config.GRPCTLSCertFile = ""
		config.GRPCTLSKeyFile = ""
		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
	})

	t.Run("valid cert loads", func(t *testing.T) {
		certFile, keyFile := writeTestCert(t)
		config.AppEnv = "production"
		config.GRPCTLSCertFile = certFile
		config.GRPCTLSKeyFile = keyFile
		config.GRPCRequireMTLS = false
		cfg, err := loadGRPCTLSConfig()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Len(t, cfg.Certificates, 1)
	})

	t.Run("invalid cert returns error", func(t *testing.T) {
		dir := t.TempDir()
		certFile := filepath.Join(dir, "cert.pem")
		keyFile := filepath.Join(dir, "key.pem")
		require.NoError(t, os.WriteFile(certFile, []byte("bad"), 0600))
		require.NoError(t, os.WriteFile(keyFile, []byte("bad"), 0600))
		config.AppEnv = "production"
		config.GRPCTLSCertFile = certFile
		config.GRPCTLSKeyFile = keyFile
		config.GRPCRequireMTLS = false
		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
	})

	t.Run("mtls requires ca", func(t *testing.T) {
		certFile, keyFile := writeTestCert(t)
		config.AppEnv = "production"
		config.GRPCTLSCertFile = certFile
		config.GRPCTLSKeyFile = keyFile
		config.GRPCRequireMTLS = true
		config.GRPCClientCAFile = ""
		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
	})

	t.Run("mtls loads client ca", func(t *testing.T) {
		certFile, keyFile := writeTestCert(t)
		config.AppEnv = "production"
		config.GRPCTLSCertFile = certFile
		config.GRPCTLSKeyFile = keyFile
		config.GRPCRequireMTLS = true
		config.GRPCClientCAFile = certFile
		cfg, err := loadGRPCTLSConfig()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.NotNil(t, cfg.ClientCAs)
	})

	t.Run("mtls rejects invalid ca", func(t *testing.T) {
		certFile, keyFile := writeTestCert(t)
		badCA := filepath.Join(t.TempDir(), "bad-ca.pem")
		require.NoError(t, os.WriteFile(badCA, []byte("bad"), 0600))
		config.AppEnv = "production"
		config.GRPCTLSCertFile = certFile
		config.GRPCTLSKeyFile = keyFile
		config.GRPCRequireMTLS = true
		config.GRPCClientCAFile = badCA
		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
	})

	t.Run("mtls returns ca read error", func(t *testing.T) {
		certFile, keyFile := writeTestCert(t)
		config.AppEnv = "production"
		config.GRPCTLSCertFile = certFile
		config.GRPCTLSKeyFile = keyFile
		config.GRPCRequireMTLS = true
		config.GRPCClientCAFile = filepath.Join(t.TempDir(), "missing-ca.pem")
		_, err := loadGRPCTLSConfig()
		require.Error(t, err)
	})
}

func TestGRPCServerOptions(t *testing.T) {
	origEnv := config.AppEnv
	origCert := config.GRPCTLSCertFile
	origKey := config.GRPCTLSKeyFile
	t.Cleanup(func() {
		config.AppEnv = origEnv
		config.GRPCTLSCertFile = origCert
		config.GRPCTLSKeyFile = origKey
	})

	config.AppEnv = "development"
	config.GRPCTLSCertFile = ""
	config.GRPCTLSKeyFile = ""
	opts, err := grpcServerOptions(&Application{})
	require.NoError(t, err)
	assert.NotEmpty(t, opts)

	certFile, keyFile := writeTestCert(t)
	config.AppEnv = "production"
	config.GRPCTLSCertFile = certFile
	config.GRPCTLSKeyFile = keyFile
	config.GRPCRequireMTLS = false
	opts, err = grpcServerOptions(&Application{})
	require.NoError(t, err)
	assert.NotEmpty(t, opts)

	config.AppEnv = "production"
	config.GRPCTLSCertFile = ""
	config.GRPCTLSKeyFile = ""
	_, err = grpcServerOptions(&Application{})
	require.Error(t, err)
}

func TestStartGRPCServer_ListenError(t *testing.T) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		t.Skipf("default gRPC port already unavailable: %v", err)
	}
	defer func() { _ = lis.Close() }()

	err = StartGRPCServer(context.Background(), &Application{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gRPC failed to listen")
}

func TestStartGRPCServer_Success(t *testing.T) {
	origEnv := config.AppEnv
	origCert := config.GRPCTLSCertFile
	origKey := config.GRPCTLSKeyFile
	t.Cleanup(func() {
		config.AppEnv = origEnv
		config.GRPCTLSCertFile = origCert
		config.GRPCTLSKeyFile = origKey
	})
	config.AppEnv = "development"
	config.GRPCTLSCertFile = ""
	config.GRPCTLSKeyFile = ""

	conn, err := net.Listen("tcp", ":50051")
	if err != nil {
		t.Skipf("default gRPC port unavailable: %v", err)
	}
	_ = conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- StartGRPCServer(ctx, &Application{}) }()

	require.Eventually(t, func() bool {
		c, err := grpc.NewClient("127.0.0.1:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return false
		}
		defer func() { _ = c.Close() }()
		c.Connect()
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer waitCancel()
		for {
			state := c.GetState()
			if state == connectivity.Ready {
				return true
			}
			if !c.WaitForStateChange(waitCtx, state) {
				return c.GetState() == connectivity.Ready
			}
		}
	}, 2*time.Second, 50*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

func TestServeGRPC_ErrorBranches(t *testing.T) {
	t.Run("options error", func(t *testing.T) {
		origEnv := config.AppEnv
		origCert := config.GRPCTLSCertFile
		origKey := config.GRPCTLSKeyFile
		t.Cleanup(func() {
			config.AppEnv = origEnv
			config.GRPCTLSCertFile = origCert
			config.GRPCTLSKeyFile = origKey
		})
		config.AppEnv = "production"
		config.GRPCTLSCertFile = ""
		config.GRPCTLSKeyFile = ""
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = lis.Close() }()
		require.Error(t, serveGRPC(context.Background(), &Application{}, lis))
	})

	t.Run("serve error", func(t *testing.T) {
		origEnv := config.AppEnv
		origCert := config.GRPCTLSCertFile
		origKey := config.GRPCTLSKeyFile
		t.Cleanup(func() {
			config.AppEnv = origEnv
			config.GRPCTLSCertFile = origCert
			config.GRPCTLSKeyFile = origKey
		})
		config.AppEnv = "development"
		config.GRPCTLSCertFile = ""
		config.GRPCTLSKeyFile = ""
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		require.NoError(t, lis.Close())
		require.Error(t, serveGRPC(context.Background(), &Application{}, lis))
	})
}

func TestServeGRPC_HealthAndShutdown(t *testing.T) {
	origEnv := config.AppEnv
	origCert := config.GRPCTLSCertFile
	origKey := config.GRPCTLSKeyFile
	t.Cleanup(func() {
		config.AppEnv = origEnv
		config.GRPCTLSCertFile = origCert
		config.GRPCTLSKeyFile = origKey
	})
	config.AppEnv = "development"
	config.GRPCTLSCertFile = ""
	config.GRPCTLSKeyFile = ""

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveGRPC(ctx, &Application{}, lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	checkCtx, checkCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer checkCancel()
	resp, err := grpc_health_v1.NewHealthClient(conn).Check(checkCtx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)

	cancel()
	require.NoError(t, <-done)
}

func nilClaims(sub, service, client string) *middleware.JWTClaims {
	return &middleware.JWTClaims{Sub: sub, Service: service, ClientID: client}
}

type testServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *testServerStream) Context() context.Context {
	return s.ctx
}

type testAddr string

func (a testAddr) Network() string { return "test" }
func (a testAddr) String() string  { return string(a) }

func writeTestCert(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	certOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyOut := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	require.NoError(t, os.WriteFile(certFile, certOut, 0600))
	require.NoError(t, os.WriteFile(keyFile, keyOut, 0600))
	return certFile, keyFile
}

func initServerTestJWTKeys(t *testing.T) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)})

	origPriv := config.JWTPrivateKey
	origPub := config.JWTPublicKey
	t.Cleanup(func() {
		config.JWTPrivateKey = origPriv
		config.JWTPublicKey = origPub
		jwt.ResetJWTKeys()
	})
	config.JWTPrivateKey = privPEM
	config.JWTPublicKey = pubPEM
	require.NoError(t, jwt.InitJWTKeys())
}

type mockGRPCAuthz struct {
	allowed bool
}

func (m mockGRPCAuthz) PolicyBundle(context.Context, iam.ServiceIdentity) (*iam.PolicyBundle, string, error) {
	return nil, "", nil
}

func (m mockGRPCAuthz) Authorize(context.Context, iam.AuthzRequest) iam.Decision {
	if m.allowed {
		return iam.Decision{Allowed: true, Reason: "matched allow"}
	}
	return iam.Decision{Allowed: false, Reason: "no matching allow"}
}
