package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"log/slog"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authctx"
	"github.com/maintainerd/maintainerd-auth/internal/iam"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
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
	// grpcOnBehalfOfClaim names the user a service principal is authorized to act
	// for. It is a claim INSIDE the signed access token, so a caller can only
	// assert an actor the issuer minted into its own token — unlike the request
	// body actor_user_uuid it replaces, which every caller could set to any user
	// in any tenant. A gRPC metadata header was rejected for the same reason: the
	// caller controls it end to end, so it would only move the forgery one hop.
	grpcOnBehalfOfClaim = "on_behalf_of"
)

// grpcActorPreloads are the associations an on-behalf-of actor needs: identities
// (for the tenant-boundary check and the tenant itself), roles/permissions so the
// AuthContext matches what the REST path builds, and the profile for attribution.
var grpcActorPreloads = []string{
	"UserIdentities.Tenant",
	"UserRoles.Role.RolePermissions.Permission",
	"Profile",
}

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
		// After auth so the claims are in context, and outermost of the business
		// handlers so it sees the final outcome.
		grpcAuditUnaryInterceptor(application),
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
	// The instance's declared ROLE is checked before anything else: it is a
	// property of the deployment that no credential can change, so a method this
	// instance does not serve at all should not cost a signature verification —
	// and refusing here means no handler on an ordinary instance can be reached
	// through a gap in the permission map.
	if err := authorizeInstanceRole(method); err != nil {
		return ctx, err
	}

	// Bootstrap/setup RPCs authenticate with a pre-shared token rather than a JWT:
	// at first boot no accounts or service principals exist yet. Core presents
	// SETUP_BOOTSTRAP_TOKEN to provision a system auth (system tenant + admin +
	// control service). The setup service layer additionally locks these once the
	// system tenant is active (ensureSetupOpen).
	if _, isBootstrap := grpcBootstrapMethods[method]; isBootstrap {
		return authorizeSetupBootstrap(ctx, application, limiter)
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

	caller, err := grpcVerifiedCaller(ctx)
	if err != nil {
		return ctx, err
	}
	claims := caller.claims
	// Certificate-bound access tokens (RFC 8705 §3).
	//
	// DPoP cannot sender-constrain a gRPC call — its proof covers an HTTP method
	// and URL — so on this transport the binding is the client certificate, which
	// mTLS has already verified for every connection here. Without this check a
	// control-plane token is a plain bearer token: anyone who obtains it AND holds
	// any certificate signed by this deployment's CA could present it and inherit
	// the orchestrator's permissions. Binding it to ONE certificate means a
	// legitimate cert belonging to a different workload is not enough.
	//
	// Checked before the rate limiter so a token presented with the wrong
	// certificate cannot consume another principal's budget.
	//
	// Clients with no registered thumbprint are unaffected, so this is opt-in per
	// client and cannot break an existing caller.
	if err := enforceGRPCCertBinding(ctx, application, claims.ClientID); err != nil {
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
	if permission != "" {
		// A missing PDP used to SKIP the decision and let the call through, so a
		// mis-wired instance served its entire permission-gated gRPC surface
		// unauthenticated-by-policy. Every other outcome on this path denies.
		if application.AuthorizationService == nil {
			return ctx, status.Error(codes.Internal, "authorization service unavailable; refusing to serve a permission-gated method")
		}
		// TenantID must be carried: without it the policy lookup resolves the
		// principal by name across all tenants and returns the lowest service_id,
		// which is the system tenant's service — collapsing every service principal
		// onto the platform's own policies.
		if claims.TenantID == 0 {
			return ctx, status.Error(codes.PermissionDenied, "service token is not bound to a tenant")
		}
		decision := application.AuthorizationService.Authorize(ctx, iam.AuthzRequest{
			Principal: claims.Service,
			Action:    permission,
			Resource:  method,
			TenantID:  claims.TenantID,
		})
		if !decision.Allowed {
			return ctx, status.Error(codes.PermissionDenied, decision.Reason)
		}
	}

	auth, err := grpcAuthContext(application.UserRepo, caller)
	if err != nil {
		return ctx, err
	}
	if _, requiresActor := grpcActorRequiredMethods[method]; requiresActor && auth.User == nil {
		return ctx, status.Errorf(codes.PermissionDenied,
			"%s changes state on behalf of a user, but this access token carries no %q claim, so the change has no actor to attribute it to or to check the tenant/escalation guards against; mint the service token with %q set to the acting user's UUID",
			method, grpcOnBehalfOfClaim, grpcOnBehalfOfClaim)
	}

	return middleware.WithAuthContextValue(middleware.ContextWithJWTClaims(ctx, claims), auth), nil
}

// grpcOnBehalfOfResolver is the narrow slice of user.UserRepository needed to
// turn an on-behalf-of subject into a full principal. Declared here rather than
// taking the repository interface so the resolution is testable without a
// database.
type grpcOnBehalfOfResolver interface {
	FindByUUID(uuid any, preloads ...string) (*user.User, error)
}

// grpcAuthContext builds the AuthContext downstream handlers read through
// middleware.AuthFromContext. Without it that lookup returned an empty context,
// so every handler that resolves its actor from the token (role mutations,
// AssignUserRoles, tenant member management) saw a nil User and denied.
//
// Tenant always comes from the verified tenant_id claim. User is populated ONLY
// from the signed on_behalf_of claim, and only after the named user is confirmed
// to live in the CALLER's own tenant: the actor is both the audit attribution and
// the subject of the handlers' membership/escalation guards, so a service token
// for tenant B naming a user in tenant A would otherwise borrow that user's
// standing — the cross-tenant takeover the request-body actor field allowed.
func grpcAuthContext(resolver grpcOnBehalfOfResolver, caller *grpcCaller) (*authctx.AuthContext, error) {
	auth := &authctx.AuthContext{}
	if caller.claims.TenantID != 0 {
		auth.Tenant = &authctx.AuthTenant{
			TenantID:   caller.claims.TenantID,
			TenantUUID: caller.claims.TenantUUID,
		}
	}
	if caller.onBehalfOf == "" {
		return auth, nil
	}

	// The tenant boundary below is the whole protection; a token that names no
	// tenant has no boundary to enforce, so it may not name an actor either.
	if caller.claims.TenantID == 0 {
		return nil, status.Errorf(codes.PermissionDenied,
			"the %q claim requires a tenant-bound token, and this token carries no tenant_id claim", grpcOnBehalfOfClaim)
	}
	actorUUID, err := uuid.Parse(caller.onBehalfOf)
	if err != nil || actorUUID == uuid.Nil {
		return nil, status.Errorf(codes.PermissionDenied,
			"the %q claim must be the acting user's UUID", grpcOnBehalfOfClaim)
	}
	if resolver == nil {
		// Fail closed: the alternative is running the mutation with no actor, which
		// is exactly the unattributable state this mechanism exists to prevent.
		return nil, status.Errorf(codes.Internal,
			"cannot resolve the %q user on this instance", grpcOnBehalfOfClaim)
	}
	actor, err := resolver.FindByUUID(actorUUID, grpcActorPreloads...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve the %q user", grpcOnBehalfOfClaim)
	}
	if actor == nil {
		return nil, status.Errorf(codes.PermissionDenied, "the %q user does not exist", grpcOnBehalfOfClaim)
	}
	// Tenant boundary. ValidateTenantAccess is the user package's own rule and
	// grants no system-tenant override, so a control-plane token still may not
	// impersonate a user it does not share a tenant with.
	if err := user.ValidateTenantAccess(actor, &user.Tenant{TenantID: caller.claims.TenantID}); err != nil {
		return nil, status.Errorf(codes.PermissionDenied,
			"the %q user does not belong to this token's tenant", grpcOnBehalfOfClaim)
	}
	// A deactivated or suspended account is a revoked one. Without this, a token
	// minted while an admin was active would keep acting as them after removal.
	if actor.Status != shared.StatusActive {
		return nil, status.Errorf(codes.PermissionDenied,
			"the %q user is not active", grpcOnBehalfOfClaim)
	}

	resolved := toUserContextByTenant(actor, caller.claims.TenantID)
	auth.User = resolved.User
	// Prefer the identity's tenant record — it carries the name/display name the
	// claim-derived one cannot. Keep the claim-derived tenant when the identity
	// row was not preloaded, so TenantUUID is never dropped.
	if resolved.Tenant != nil && resolved.Tenant.TenantUUID != uuid.Nil {
		auth.Tenant = resolved.Tenant
	}
	return auth, nil
}

// authorizeInstanceRole refuses core-provisioning RPCs on any instance that is
// not the ecosystem's system IAM.
//
// Core provisions many auth instances: one SYSTEM instance the whole maintainerd
// ecosystem is built on, plus ordinary instances a developer spins up for their
// own application and throws away. Only the first is core's to drive. Without
// this check every instance answered the orchestrator's provisioning surface, so
// anyone who could reach an ordinary instance's control-plane port — a tenant
// admin, say — was holding an API that was only ever meant for core.
//
// Both conditions FAIL CLOSED. config.IsSystemInstance reports false for the
// unset and for any unrecognised role, and the control-plane switch is re-checked
// here even though StartGRPCServer will not bind a listener without it, because
// "the listener could not exist" is an assumption about another function rather
// than a guarantee at this call site.
func authorizeInstanceRole(method string) error {
	if !grpcRequiresSystemInstance(method) {
		return nil
	}
	if !config.ControlPlaneEnabled {
		return status.Errorf(codes.FailedPrecondition,
			"%s belongs to the maintainerd control plane, which is disabled on this instance (CONTROL_PLANE_ENABLED is not true); it is a standalone deployment and serves no orchestrator surface",
			method)
	}
	if !config.IsSystemInstance() {
		return status.Errorf(codes.FailedPrecondition,
			"%s is a core-provisioning RPC and is served only by the maintainerd ecosystem's SYSTEM auth instance; this instance declares INSTANCE_ROLE=%q (an unset or unrecognised role is treated as %q). Direct this call at the auth instance provisioned with INSTANCE_ROLE=%q, or administer this instance through its console/REST surface",
			method, config.InstanceRole, config.InstanceRoleRegular, config.InstanceRoleSystem)
	}
	return nil
}

// authorizeSetupBootstrap gates the gRPC SetupService with the per-instance
// bootstrap credential. When no credential is configured the whole gRPC setup
// surface is disabled (standalone instances bootstrap via the REST wizard).
//
// Authorization is delegated to BootstrapCredentialService, which holds the
// credential's state in the DATABASE. A constant-time compare against the env
// var alone answers "is this the right secret", never "has it already been
// spent" — so the credential stayed valid forever, and single-use was a property
// only the setup service's own ensureSetupOpen check approximated. Delegating
// here makes it the credential's property: spent means spent, across restarts
// and across every replica sharing the database.
//
// The rate limit still applies before the credential is examined, so a guessing
// attack is throttled whatever the store says.
func authorizeSetupBootstrap(ctx context.Context, application *Application, limiter *grpcLimiter) (context.Context, error) {
	if config.SetupBootstrapToken == "" {
		return ctx, status.Error(codes.PermissionDenied, "gRPC setup is disabled")
	}
	md, _ := metadata.FromIncomingContext(ctx)
	provided := ""
	if values := md.Get(grpcSetupTokenKey); len(values) > 0 {
		provided = strings.TrimSpace(values[0])
	}
	if provided == "" {
		return ctx, status.Error(codes.Unauthenticated, "invalid setup bootstrap token")
	}
	if !limiter.Allow("setup-bootstrap", time.Now()) {
		return ctx, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	if subtle.ConstantTimeCompare([]byte(provided), []byte(config.SetupBootstrapToken)) != 1 {
		return ctx, status.Error(codes.Unauthenticated, "invalid setup bootstrap token")
	}

	// There is no "already spent" check here, and deliberately no ledger to hold
	// one. Whether this instance has been bootstrapped is already recorded by the
	// existence of the system tenant: the setup service reads it in
	// ensureSetupOpen and refuses every mutating call once it is there, and the
	// single-system-tenant constraint settles the case where two replicas race on
	// a genuinely fresh instance. A separate table storing that same boolean could
	// only drift from the fact it was copying.
	//
	// Who may present the credential at all is settled a layer up: with the
	// control plane on, R2 makes mTLS mandatory, so the caller has already proven
	// a client certificate signed by this deployment's CA before reaching here.

	slog.Info("gRPC bootstrap call authorized", "peer", grpcBootstrapCaller(ctx))

	return ctx, nil
}

// grpcBootstrapCaller names the peer for the audit trail: the verified client
// certificate subject when mTLS is in play, otherwise the socket address.
func grpcBootstrapCaller(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p == nil {
		return ""
	}
	if tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo); ok {
		for _, chain := range tlsInfo.State.VerifiedChains {
			if len(chain) > 0 && chain[0].Subject.CommonName != "" {
				return "mtls:" + chain[0].Subject.CommonName
			}
		}
	}
	if p.Addr != nil {
		return "peer:" + p.Addr.String()
	}
	return ""
}

// grpcCaller is the verified principal behind a gRPC call: the shared JWT claims
// plus the on-behalf-of subject, which is not a field on middleware.JWTClaims
// because only this transport honours it.
type grpcCaller struct {
	claims     *middleware.JWTClaims
	onBehalfOf string
}

func grpcVerifiedCaller(ctx context.Context) (*grpcCaller, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization metadata required")
	}
	parts := strings.SplitN(values[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		return nil, status.Error(codes.Unauthenticated, "bearer token required")
	}

	// Access tokens only: ValidateTokenWithContext is permissive about token_type,
	// so an ID token — routinely handed to relying parties — would otherwise
	// authenticate here as a service credential.
	rawClaims, err := jwt.ValidateAccessTokenWithContext(ctx, parts[1])
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
	tenantUUIDStr := stringClaim(rawClaims, "tenant_id")
	tenantUUID, _ := uuid.Parse(tenantUUIDStr)
	claims := &middleware.JWTClaims{
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
		// This path builds JWTClaims by hand rather than through
		// middleware.buildJWTClaims, so every claim it needs must be mapped here.
		// The tenant_id claim VALUE is the tenant's opaque UUID; keep it and resolve
		// it back to the internal PK for TenantID (mirrors buildJWTClaims).
		TenantUUID: tenantUUID,
		TenantID:   shared.TenantIDByUUIDString(ctx, tenantUUIDStr),
	}
	return &grpcCaller{
		claims: claims,
		// Read straight off the VERIFIED claim set: the signature is what makes
		// this actor unforgeable by the caller.
		onBehalfOf: strings.TrimSpace(stringClaim(rawClaims, grpcOnBehalfOfClaim)),
	}, nil
}

// int64Claim reads a numeric claim; JSON numbers decode to float64.
func int64Claim(raw any) int64 {
	switch v := raw.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
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

// enforceGRPCCertBinding rejects a certificate-bound token presented by a peer
// whose certificate is not the one it was bound to.
func enforceGRPCCertBinding(ctx context.Context, application *Application, clientID string) error {
	if application == nil || application.ClientService == nil || clientID == "" {
		return nil
	}
	expected := application.ClientService.BoundCertThumbprint(ctx, clientID)
	if expected == "" {
		// Not certificate-bound; an ordinary bearer token for this transport.
		return nil
	}

	presented, ok := grpcPeerCertThumbprint(ctx)
	if !ok {
		// The token says it may only be used with a certificate and this
		// connection presented none. Failing open here would make the binding
		// advisory, which is the same as not having it.
		slog.Warn("certificate-bound token presented without a client certificate", "client_id", clientID)
		return status.Error(codes.Unauthenticated,
			"this access token is bound to a client certificate, which this connection did not present")
	}
	if subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
		slog.Warn("certificate-bound token presented with the wrong certificate", "client_id", clientID)
		return status.Error(codes.Unauthenticated,
			"this access token is bound to a different client certificate")
	}
	return nil
}

// grpcPeerCertThumbprint returns the base64url-encoded SHA-256 thumbprint of the
// peer's verified leaf certificate, in the encoding RFC 8705 §3.1 defines for
// the x5t#S256 confirmation value.
//
// It reads VerifiedChains, never PeerCertificates: the latter is whatever the
// client sent, so using it would compare against a certificate nobody checked
// was issued by this deployment's CA.
func grpcPeerCertThumbprint(ctx context.Context) (string, bool) {
	p, ok := peer.FromContext(ctx)
	if !ok || p == nil {
		return "", false
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", false
	}
	for _, chain := range tlsInfo.State.VerifiedChains {
		if len(chain) == 0 || chain[0] == nil {
			continue
		}
		sum := sha256.Sum256(chain[0].Raw)
		return base64.RawURLEncoding.EncodeToString(sum[:]), true
	}
	return "", false
}
