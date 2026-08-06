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
	"github.com/maintainerd/maintainerd-auth/internal/federation"
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
//
// The listener is OPT-IN. gRPC here is not a second API for the same product; it
// is the machine control plane an orchestrator (maintainerd core) drives to
// create tenants, services and clients. The default deployment is STANDALONE —
// an organisation running this as their own IAM, with no orchestrator anywhere —
// and binding the port there hands out a control plane nobody asked for. So with
// CONTROL_PLANE_ENABLED unset the socket is never opened: not bound-and-
// firewalled, not bound-and-authenticated, not bound.
func StartGRPCServer(ctx context.Context, application *Application) error {
	return startGRPCServerOn(ctx, application, shared.DefaultGRPCAddr)
}

// startGRPCServerOn takes the bind address as a parameter so the opt-in gate can
// be tested against an address the test controls. Proving the listener is NOT
// bound means observing that an already-occupied address produces no error, and
// that is only deterministic on an address the test owns.
func startGRPCServerOn(ctx context.Context, application *Application, addr string) error {
	if !config.ControlPlaneEnabled {
		// Said once, at startup, because the other way an operator learns this is
		// core failing with connection refused against a healthy instance.
		slog.Info("gRPC control plane disabled; no listener bound (standalone deployment)",
			"addr", addr, "enable_with", "CONTROL_PLANE_ENABLED=true")
		return nil
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("gRPC failed to listen on %s: %w", addr, err)
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

	services := grpcServices(application)
	for _, svc := range services {
		svc.register(s)
	}
	setGRPCServiceHealth(healthServer, services, healthpb.HealthCheckResponse_SERVING)
	updateGRPCOverallHealth(ctx, healthServer, application)
	go refreshGRPCOverallHealth(ctx, healthServer, application, grpcHealthRefreshInterval)

	if config.AppEnv != "production" {
		reflection.Register(s)
	}

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

// grpcService pairs a service's registered name with the call that binds its
// handler, so registration and health advertisement cannot describe different
// surfaces.
type grpcService struct {
	name     string
	register func(grpc.ServiceRegistrar)
}

// grpcServices is the single definition of the gRPC surface this server serves.
//
// It exposes ONLY the machine-to-machine surface:
//   - CORE provisioning: IAM primitives (service/api/permission/policy/role),
//     client, tenant records + tenant settings, and bootstrap setup.
//   - Peer services / BFF: user-info reads, token introspection, authorization (PDP).
//
// This list is the WHOLE of maintainerd.auth.v1: the contract used to declare a
// further twelve services (auth events, branding, email/SMS config and
// templates, identity providers, invites, IP restriction rules, registration
// flows, security settings, webhook endpoints) that had no handler anywhere in
// the tree, so callers got UNIMPLEMENTED from RPCs the contract promised. Those
// service blocks were deleted from the protos — they are control-plane REST only
// — and TestGRPCContractIsFullyServed now asserts the declared set and the
// registered set are identical, so a re-added RPC cannot ship unserved.
//
// Registration and setGRPCServiceHealth both read from this ONE list because
// they used to be two hand-maintained lists that had already drifted:
// TenantSettingService was registered but never health-advertised, so a caller
// probing that service got SERVICE_UNKNOWN from a service that was serving.
func grpcServices(application *Application) []grpcService {
	return []grpcService{
		{authv1.SetupService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterSetupServiceServer(s, setup.NewSetupGRPCHandler(application.SetupService))
		}},
		{authv1.TenantService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterTenantServiceServer(s, tenant.NewTenantGRPCHandler(application.TenantService, application.TenantMemberService))
		}},
		{authv1.TenantSettingService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterTenantSettingServiceServer(s, tenant.NewTenantSettingGRPCHandler(application.TenantService, application.TenantMemberService, application.TenantSettingService))
		}},
		{authv1.ServiceService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterServiceServiceServer(s, iam.NewServiceGRPCHandler(application.TenantService, application.ServiceService, application.AuthorizationService))
		}},
		{authv1.APIService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterAPIServiceServer(s, iam.NewAPIGRPCHandler(application.TenantService, application.APIService))
		}},
		{authv1.PermissionService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterPermissionServiceServer(s, iam.NewPermissionGRPCHandler(application.TenantService, application.PermissionService))
		}},
		{authv1.PolicyService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterPolicyServiceServer(s, iam.NewPolicyGRPCHandler(application.TenantService, application.PolicyService))
		}},
		{authv1.RoleService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterRoleServiceServer(s, iam.NewRoleGRPCHandler(application.TenantService, application.RoleService))
		}},
		{authv1.WorkloadIdentityFederationService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterWorkloadIdentityFederationServiceServer(s, federation.NewWorkloadIdentityFederationGRPCHandler(
				federationTenantResolver{application.TenantService}, application.WorkloadIdentityFederationService))
		}},
		{authv1.AuthorizationService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterAuthorizationServiceServer(s, iam.NewAuthorizationGRPCHandler(application.AuthorizationService))
		}},
		{authv1.ClientService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterClientServiceServer(s, client.NewClientGRPCHandler(clientTenantResolver{application.TenantService}, application.ClientService))
		}},
		{authv1.UserService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterUserServiceServer(s, user.NewUserGRPCHandler(userTenantResolver{application.TenantService}, application.UserService))
		}},
		{authv1.UserProfileService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterUserProfileServiceServer(s, user.NewUserProfileGRPCHandler(userTenantResolver{application.TenantService}, application.ProfileService))
		}},
		{authv1.OAuthIntrospectionService_ServiceDesc.ServiceName, func(s grpc.ServiceRegistrar) {
			authv1.RegisterOAuthIntrospectionServiceServer(s, oauth.NewOAuthIntrospectionGRPCHandler(application.OAuthTokenService))
		}},
	}
}

func setGRPCServiceHealth(healthServer *health.Server, services []grpcService, status healthpb.HealthCheckResponse_ServingStatus) {
	for _, svc := range services {
		healthServer.SetServingStatus(svc.name, status)
	}
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
	// Last gate before the server exists. loadGRPCTLSConfig is supposed to have
	// guaranteed this, but the cost of it ever being wrong is a control plane that
	// creates tenants for an unverified peer, so the invariant is asserted where
	// the credentials are actually installed rather than assumed from the function
	// that produced them.
	if config.ControlPlaneEnabled {
		if tlsConfig == nil {
			return nil, fmt.Errorf("refusing to serve the control plane without TLS: CONTROL_PLANE_ENABLED=true requires GRPC_TLS_CERT_FILE, GRPC_TLS_KEY_FILE and GRPC_CLIENT_CA_FILE")
		}
		if tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
			return nil, fmt.Errorf("refusing to serve the control plane without mutual TLS: client certificates must be required and verified, so GRPC_CLIENT_CA_FILE must name the CA that issues core's client certificate")
		}
	}
	if tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	return opts, nil
}

func loadGRPCTLSConfig() (*tls.Config, error) {
	certFile := config.GRPCTLSCertFile
	keyFile := config.GRPCTLSKeyFile

	// Mutual TLS is not an operator preference on the control plane, it is the
	// only thing that PROVES the peer is core. A bearer token asserts "it is
	// core"; a verified client certificate demonstrates it. So the requirement is
	// re-derived from CONTROL_PLANE_ENABLED here instead of trusting
	// GRPC_REQUIRE_MTLS: config.Init already forces that flag true, and deriving
	// it again means there is no ordering, no test seam and no future embedder
	// that can produce a control plane running on server-side TLS alone.
	// GRPC_REQUIRE_MTLS remains meaningful only for a non-control-plane listener.
	requireMTLS := config.GRPCRequireMTLS || config.ControlPlaneEnabled

	if certFile == "" || keyFile == "" {
		// The non-production "just run it in the clear" convenience is exactly the
		// downgrade this requirement removes: it would put the tenant/service/client
		// creation surface on a plaintext socket. It stays available only to the
		// listener that is not the control plane.
		if config.ControlPlaneEnabled {
			return nil, fmt.Errorf("GRPC_TLS_CERT_FILE and GRPC_TLS_KEY_FILE are required when CONTROL_PLANE_ENABLED=true: the control plane is never served in the clear")
		}
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
	if requireMTLS {
		requiredBy := "GRPC_REQUIRE_MTLS=true"
		if config.ControlPlaneEnabled {
			requiredBy = "CONTROL_PLANE_ENABLED=true"
		}
		if config.GRPCClientCAFile == "" {
			return nil, fmt.Errorf("GRPC_CLIENT_CA_FILE is required when %s", requiredBy)
		}
		caBytes, err := os.ReadFile(config.GRPCClientCAFile)
		if err != nil {
			// Refusing to start beats starting without client verification: an
			// unreadable CA is an operator mistake, and the "helpful" fallback would
			// silently reopen the port to anyone who can route to it.
			return nil, fmt.Errorf("read gRPC client CA file %q (required by %s): %w", config.GRPCClientCAFile, requiredBy, err)
		}
		clientCAs := x509.NewCertPool()
		if !clientCAs.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("gRPC client CA file %q contains no PEM certificate (required by %s)", config.GRPCClientCAFile, requiredBy)
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

// federationTenantResolver adapts the tenant service to the resolver the
// federation gRPC handler needs.
type federationTenantResolver struct {
	svc tenant.TenantService
}

func (r federationTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (int64, error) {
	t, err := r.svc.GetByUUID(ctx, tenantUUID)
	if err != nil {
		return 0, err
	}
	return t.TenantID, nil
}
