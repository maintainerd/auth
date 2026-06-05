package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/maintainerd/auth/internal/iam"
	"github.com/maintainerd/auth/internal/platform/config"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/setup"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/maintainerd/auth/internal/tenant"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// StartGRPCServer binds to shared.DefaultGRPCAddr and serves until ctx is cancelled, at which
// point it drains in-flight RPCs via GracefulStop. It returns an error for any
// fatal startup failure so that main() can handle it appropriately instead of
// calling os.Exit inside a library function.
func StartGRPCServer(ctx context.Context, application *Application) error {
	lis, err := net.Listen("tcp", shared.DefaultGRPCAddr)
	if err != nil {
		return fmt.Errorf("gRPC failed to listen on %s: %w", shared.DefaultGRPCAddr, err)
	}
	return serveGRPC(ctx, application, lis)
}

func serveGRPC(ctx context.Context, application *Application, lis net.Listener) error {
	opts, err := grpcServerOptions(application)
	if err != nil {
		return err
	}
	s := grpc.NewServer(opts...)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.SetupService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.TenantService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.TenantSettingService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.ServiceService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.APIService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.PermissionService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.PolicyService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.RoleService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(authv1.AuthorizationService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	if config.AppEnv != "production" {
		reflection.Register(s)
	}

	authv1.RegisterSetupServiceServer(s, setup.NewSetupGRPCHandler(application.SetupService))
	authv1.RegisterTenantServiceServer(s, tenant.NewTenantGRPCHandler(application.TenantService, application.TenantMemberService))
	authv1.RegisterTenantSettingServiceServer(s, tenant.NewTenantSettingGRPCHandler(application.TenantService, application.TenantSettingService))
	authv1.RegisterServiceServiceServer(s, iam.NewServiceGRPCHandler(application.TenantService, application.ServiceService))
	authv1.RegisterAPIServiceServer(s, iam.NewAPIGRPCHandler(application.TenantService, application.APIService))
	authv1.RegisterPermissionServiceServer(s, iam.NewPermissionGRPCHandler(application.TenantService, application.PermissionService))
	authv1.RegisterPolicyServiceServer(s, iam.NewPolicyGRPCHandler(application.TenantService, application.PolicyService))
	authv1.RegisterRoleServiceServer(s, iam.NewRoleGRPCHandler(application.TenantService, application.RoleService))
	authv1.RegisterAuthorizationServiceServer(s, iam.NewAuthorizationGRPCHandler(application.AuthorizationService))

	// Stop the server when the context is cancelled (e.g. after REST servers drain).
	go func() {
		<-ctx.Done()
		slog.Info("gRPC shutdown signal received, draining connections...")
		healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		s.GracefulStop()
	}()

	slog.Info("gRPC server starting", "addr", shared.DefaultGRPCAddr)
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}
	return nil
}

func grpcServerOptions(application *Application) ([]grpc.ServerOption, error) {
	opts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(grpcUnaryInterceptors(application)...),
		grpc.ChainStreamInterceptor(grpcStreamInterceptors(application)...),
		grpc.MaxRecvMsgSize(10 * 1024 * 1024),
		grpc.MaxSendMsgSize(10 * 1024 * 1024),
	}

	tlsConfig, err := loadGRPCTLSConfig()
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	return opts, nil
}

func loadGRPCTLSConfig() (*tls.Config, error) {
	certFile := config.GRPCTLSCertFile
	keyFile := config.GRPCTLSKeyFile
	if certFile == "" || keyFile == "" {
		if config.AppEnv == "production" {
			return nil, fmt.Errorf("gRPC TLS cert and key are required in production")
		}
		slog.Warn("gRPC TLS disabled for non-production environment")
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load gRPC TLS certificate: %w", err)
	}
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	if config.GRPCRequireMTLS {
		if config.GRPCClientCAFile == "" {
			return nil, fmt.Errorf("GRPC_CLIENT_CA_FILE is required when GRPC_REQUIRE_MTLS=true")
		}
		caBytes, err := os.ReadFile(config.GRPCClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("read gRPC client CA file: %w", err)
		}
		clientCAs := x509.NewCertPool()
		if !clientCAs.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("parse gRPC client CA file")
		}
		tlsConfig.ClientCAs = clientCAs
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return tlsConfig, nil
}
