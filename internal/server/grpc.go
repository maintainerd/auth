package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/setup"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const grpcHealthRefreshInterval = 5 * time.Second

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
	setGRPCServiceHealth(healthServer, healthpb.HealthCheckResponse_SERVING)
	updateGRPCOverallHealth(ctx, healthServer, application)
	go refreshGRPCOverallHealth(ctx, healthServer, application, grpcHealthRefreshInterval)

	if config.AppEnv != "production" {
		reflection.Register(s)
	}

	// gRPC exposes ONLY the machine-to-machine surface:
	//   - CORE provisioning: IAM primitives (service/api/permission/policy/role),
	//     client, tenant records, and bootstrap setup.
	//   - Peer services / BFF: user-info reads, token introspection, authorization (PDP).
	// Tenant admin/UX/comms operations (branding, templates, email/SMS config,
	// webhooks, IdP, registration flows, invites, security settings, IP rules, audit
	// browsing, tenant settings) are REST control-plane only and are intentionally
	// NOT registered here. Their proto/handlers remain in-package but, with no gRPC
	// handler registered, the server returns UNIMPLEMENTED before any interceptor.
	authv1.RegisterSetupServiceServer(s, setup.NewSetupGRPCHandler(application.SetupService))
	authv1.RegisterTenantServiceServer(s, tenant.NewTenantGRPCHandler(application.TenantService, application.TenantMemberService))
	authv1.RegisterServiceServiceServer(s, iam.NewServiceGRPCHandler(application.TenantService, application.ServiceService, application.AuthorizationService))
	authv1.RegisterAPIServiceServer(s, iam.NewAPIGRPCHandler(application.TenantService, application.APIService))
	authv1.RegisterPermissionServiceServer(s, iam.NewPermissionGRPCHandler(application.TenantService, application.PermissionService))
	authv1.RegisterPolicyServiceServer(s, iam.NewPolicyGRPCHandler(application.TenantService, application.PolicyService))
	authv1.RegisterRoleServiceServer(s, iam.NewRoleGRPCHandler(application.TenantService, application.RoleService))
	authv1.RegisterAuthorizationServiceServer(s, iam.NewAuthorizationGRPCHandler(application.AuthorizationService))
	authv1.RegisterClientServiceServer(s, client.NewClientGRPCHandler(clientTenantResolver{application.TenantService}, application.ClientService))
	authv1.RegisterUserServiceServer(s, user.NewUserGRPCHandler(userTenantResolver{application.TenantService}, application.UserService))
	authv1.RegisterUserProfileServiceServer(s, user.NewUserProfileGRPCHandler(userTenantResolver{application.TenantService}, application.ProfileService))
	authv1.RegisterOAuthIntrospectionServiceServer(s, oauth.NewOAuthIntrospectionGRPCHandler(application.OAuthTokenService))

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

func setGRPCServiceHealth(healthServer *health.Server, status healthpb.HealthCheckResponse_ServingStatus) {
	healthServer.SetServingStatus(authv1.SetupService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.TenantService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.ServiceService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.APIService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.PermissionService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.PolicyService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.RoleService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.AuthorizationService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.ClientService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.UserService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.UserProfileService_ServiceDesc.ServiceName, status)
	healthServer.SetServingStatus(authv1.OAuthIntrospectionService_ServiceDesc.ServiceName, status)
}

func refreshGRPCOverallHealth(ctx context.Context, healthServer *health.Server, application *Application, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updateGRPCOverallHealth(ctx, healthServer, application)
		}
	}
}

func updateGRPCOverallHealth(ctx context.Context, healthServer *health.Server, application *Application) {
	healthServer.SetServingStatus("", grpcOverallHealthStatus(ctx, application))
}

func grpcOverallHealthStatus(ctx context.Context, application *Application) healthpb.HealthCheckResponse_ServingStatus {
	if application == nil || application.DB == nil {
		return healthpb.HealthCheckResponse_NOT_SERVING
	}
	sqlDB, err := application.DB.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		return healthpb.HealthCheckResponse_NOT_SERVING
	}
	if application.RedisClient != nil && application.RedisClient.Ping(ctx).Err() != nil {
		return healthpb.HealthCheckResponse_NOT_SERVING
	}
	if jwt.GetPublicKey() == nil {
		return healthpb.HealthCheckResponse_NOT_SERVING
	}
	return healthpb.HealthCheckResponse_SERVING
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

type clientTenantResolver struct {
	svc tenant.TenantService
}

func (r clientTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*client.TenantServiceDataResult, error) {
	t, err := r.svc.GetByUUID(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	return &client.TenantServiceDataResult{
		TenantID:    t.TenantID,
		TenantUUID:  t.TenantUUID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		Description: t.Description,
		Status:      t.Status,
		IsSystem:    t.IsSystem,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}, nil
}

type userTenantResolver struct {
	svc tenant.TenantService
}

func (r userTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*user.TenantServiceDataResult, error) {
	t, err := r.svc.GetByUUID(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	return &user.TenantServiceDataResult{
		TenantID:    t.TenantID,
		TenantUUID:  t.TenantUUID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		Description: t.Description,
		Status:      t.Status,
		IsSystem:    t.IsSystem,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}, nil
}
