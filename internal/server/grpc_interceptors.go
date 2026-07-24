package server

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	defaultGRPCTimeout   = 60 * time.Second
	defaultGRPCRateLimit = 600
	defaultGRPCWindow    = time.Minute
	grpcRequestIDKey     = "x-request-id"
	// grpcAppServicePrefix is the RPC path prefix for the application proto package.
	// Any method under it that is not explicitly classified is denied (fail closed);
	// infrastructure services (health, reflection) fall outside it and pass through.
	grpcAppServicePrefix = "/maintainerd.auth.v1."
	// grpcSetupTokenKey carries the SETUP_BOOTSTRAP_TOKEN on bootstrap RPCs.
	grpcSetupTokenKey = "x-setup-token"
)

type grpcRequestIDContextKey struct{}

type grpcLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]grpcLimitBucket
}

type grpcLimitBucket struct {
	start time.Time
	count int
}

func newGRPCLimiter(limit int, window time.Duration) *grpcLimiter {
	return &grpcLimiter{limit: limit, window: window, buckets: make(map[string]grpcLimitBucket)}
}

func (l *grpcLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[key]
	if bucket.start.IsZero() || now.Sub(bucket.start) >= l.window {
		l.buckets[key] = grpcLimitBucket{start: now, count: 1}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}

func grpcUnaryInterceptors(application *Application) []grpc.UnaryServerInterceptor {
	limiter := newGRPCLimiter(defaultGRPCRateLimit, defaultGRPCWindow)
	return []grpc.UnaryServerInterceptor{
		grpcRecoveryUnaryInterceptor(),
		grpcLoggingUnaryInterceptor(),
		grpcTimeoutUnaryInterceptor(defaultGRPCTimeout),
		grpcAuthUnaryInterceptor(application, limiter),
	}
}

func grpcStreamInterceptors(application *Application) []grpc.StreamServerInterceptor {
	limiter := newGRPCLimiter(defaultGRPCRateLimit, defaultGRPCWindow)
	return []grpc.StreamServerInterceptor{
		grpcRecoveryStreamInterceptor(),
		grpcLoggingStreamInterceptor(),
		grpcTimeoutStreamInterceptor(defaultGRPCTimeout),
		grpcAuthStreamInterceptor(application, limiter),
	}
}

func grpcRecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("gRPC panic recovered", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		resp, err = handler(ctx, req)
		return resp, apperror.ToGRPCError(err)
	}
}

func grpcRecoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("gRPC panic recovered", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return apperror.ToGRPCError(handler(srv, stream))
	}
}

func grpcLoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		ctx, requestID := grpcContextWithRequestID(ctx)
		resp, err := handler(ctx, req)
		_ = grpc.SetTrailer(ctx, metadata.Pairs(grpcRequestIDKey, requestID))
		logGRPC(ctx, info.FullMethod, start, err)
		return resp, err
	}
}

func grpcLoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx, requestID := grpcContextWithRequestID(stream.Context())
		wrapped := &grpcContextServerStream{ServerStream: stream, ctx: ctx}
		err := handler(srv, wrapped)
		_ = grpc.SetTrailer(ctx, metadata.Pairs(grpcRequestIDKey, requestID))
		logGRPC(ctx, info.FullMethod, start, err)
		return err
	}
}

func grpcTimeoutUnaryInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(timeoutCtx, req)
	}
}

func grpcTimeoutStreamInterceptor(timeout time.Duration) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		timeoutCtx, cancel := context.WithTimeout(stream.Context(), timeout)
		defer cancel()
		return handler(srv, &grpcContextServerStream{ServerStream: stream, ctx: timeoutCtx})
	}
}

func grpcAuthUnaryInterceptor(application *Application, limiter *grpcLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		authCtx, err := authenticateAndAuthorizeGRPC(ctx, application, limiter, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(authCtx, req)
	}
}

func grpcAuthStreamInterceptor(application *Application, limiter *grpcLimiter) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		authCtx, err := authenticateAndAuthorizeGRPC(stream.Context(), application, limiter, info.FullMethod)
		if err != nil {
			return err
		}
		return handler(srv, &grpcContextServerStream{ServerStream: stream, ctx: authCtx})
	}
}

func authenticateAndAuthorizeGRPC(ctx context.Context, application *Application, limiter *grpcLimiter, method string) (context.Context, error) {
	// Bootstrap/setup RPCs authenticate with a pre-shared token rather than a JWT:
	// at first boot no accounts or service principals exist yet. Core presents
	// SETUP_BOOTSTRAP_TOKEN to provision a system auth (system tenant + admin +
	// control service). The setup service layer additionally locks these once the
	// system tenant is active (ensureSetupOpen).
	if _, isBootstrap := grpcBootstrapMethods[method]; isBootstrap {
		return authorizeSetupBootstrap(ctx, limiter)
	}

	permission, protected := grpcServicePermissions[method]
	if !protected {
		// Default-deny for the application RPC surface: any maintainerd.auth.v1
		// method not explicitly classified is rejected (fail closed). Infrastructure
		// services (health, reflection) are not in the app package and pass through.
		if strings.HasPrefix(method, grpcAppServicePrefix) {
			return ctx, status.Error(codes.PermissionDenied, "method not permitted")
		}
		return ctx, nil
	}

	claims, err := grpcJWTClaims(ctx)
	if err != nil {
		return ctx, err
	}
	identity := grpcPrincipalKey(claims)
	if !limiter.Allow(identity, time.Now()) {
		return ctx, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	if claims.Service == "" && claims.SubjectType != "service" {
		return ctx, status.Error(codes.PermissionDenied, "service account token required")
	}
	if _, requiresStepUp := grpcStepUpMethods[method]; requiresStepUp && claims.ACR != jwt.ACRLevel2 {
		return ctx, status.Error(codes.PermissionDenied, "step-up authentication required")
	}
	if permission != "" && application.AuthorizationService != nil {
		decision := application.AuthorizationService.Authorize(ctx, iam.AuthzRequest{
			Principal: claims.Service,
			Action:    permission,
			Resource:  method,
		})
		if !decision.Allowed {
			return ctx, status.Error(codes.PermissionDenied, decision.Reason)
		}
	}

	return middleware.ContextWithJWTClaims(ctx, claims), nil
}

// authorizeSetupBootstrap gates the gRPC SetupService with the pre-shared
// SETUP_BOOTSTRAP_TOKEN. When the token is unset the whole gRPC setup surface is
// disabled (standalone instances bootstrap via the REST wizard). The token is
// compared in constant time and its use is rate-limited.
func authorizeSetupBootstrap(ctx context.Context, limiter *grpcLimiter) (context.Context, error) {
	expected := config.SetupBootstrapToken
	if expected == "" {
		return ctx, status.Error(codes.PermissionDenied, "gRPC setup is disabled")
	}
	md, _ := metadata.FromIncomingContext(ctx)
	provided := ""
	if values := md.Get(grpcSetupTokenKey); len(values) > 0 {
		provided = strings.TrimSpace(values[0])
	}
	if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return ctx, status.Error(codes.Unauthenticated, "invalid setup bootstrap token")
	}
	if !limiter.Allow("setup-bootstrap", time.Now()) {
		return ctx, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	return ctx, nil
}

func grpcJWTClaims(ctx context.Context) (*middleware.JWTClaims, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization metadata required")
	}
	parts := strings.SplitN(values[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		return nil, status.Error(codes.Unauthenticated, "bearer token required")
	}

	rawClaims, err := jwt.ValidateTokenWithContext(ctx, parts[1])
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	// DPoP binds a proof to an HTTP method and URL (RFC 9449 §4.2), so a
	// sender-constrained token cannot be proven over gRPC. Accepting it as a bearer
	// token here would leave a bypass for the one token type that is supposed to be
	// theft-resistant, so it is refused outright.
	if middleware.IsSenderConstrainedToken(rawClaims) {
		return nil, status.Error(codes.Unauthenticated,
			"this access token is bound to a DPoP key and cannot be used over gRPC")
	}

	sub, _ := rawClaims["sub"].(string)
	userUUID, _ := uuid.Parse(sub)
	return &middleware.JWTClaims{
		Sub:         sub,
		UserUUID:    userUUID,
		Service:     stringClaim(rawClaims, "svc"),
		SubjectType: stringClaim(rawClaims, "sub_type"),
		Scope:       stringClaim(rawClaims, "scope"),
		Audience:    stringClaim(rawClaims, "aud"),
		Issuer:      stringClaim(rawClaims, "iss"),
		JTI:         stringClaim(rawClaims, "jti"),
		ClientID:    stringClaim(rawClaims, "client_id"),
		ProviderID:  stringClaim(rawClaims, "provider_id"),
		SessionID:   stringClaim(rawClaims, "sid"),
		ACR:         stringClaim(rawClaims, "acr"),
		AMR:         stringSliceClaim(rawClaims["amr"]),
	}, nil
}

func stringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}

func stringSliceClaim(raw any) []string {
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func grpcPrincipalKey(claims *middleware.JWTClaims) string {
	if claims.Service != "" {
		return "svc:" + claims.Service
	}
	if claims.ClientID != "" {
		return "client:" + claims.ClientID
	}
	return "sub:" + claims.Sub
}

func logGRPC(ctx context.Context, method string, start time.Time, err error) {
	code := status.Code(err)
	addr := ""
	if p, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			addr = tcpAddr.IP.String()
		} else if p.Addr != nil {
			addr = p.Addr.String()
		}
	}
	slog.Info("gRPC request", "method", method, "code", code.String(), "duration_ms", time.Since(start).Milliseconds(), "peer", addr, "request_id", grpcRequestIDFromContext(ctx))
}

func grpcContextWithRequestID(ctx context.Context) (context.Context, string) {
	requestID := grpcRequestIDFromMetadata(ctx)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return context.WithValue(ctx, grpcRequestIDContextKey{}, requestID), requestID
}

func grpcRequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(grpcRequestIDContextKey{}).(string)
	return requestID
}

func grpcRequestIDFromMetadata(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, key := range []string{grpcRequestIDKey, "request-id", "x-correlation-id", "correlation-id"} {
		if values := md.Get(key); len(values) > 0 && strings.TrimSpace(values[0]) != "" {
			return strings.TrimSpace(values[0])
		}
	}
	return ""
}

type grpcContextServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *grpcContextServerStream) Context() context.Context {
	return s.ctx
}
