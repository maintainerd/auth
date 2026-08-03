package authn

import (
	"context"
	"log/slog"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	platformjwt "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/platform/middleware"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// RefreshToken exchanges a valid refresh token (issued by the internal login
// flow) for a fresh token set — a new access token, id token, and a rotated
// refresh token.
//
// It is session-aware: when the caller supplies the session id (via the
// X-Session-ID header or the sid claim of the access_token cookie) the existing
// server-side session is validated and reused, so the session's idle and
// absolute-lifetime limits keep applying across refreshes. When no session id is
// supplied a new session is created, mirroring the login flow.
//
// The consumed refresh token's JTI is added to the denylist so it cannot be
// replayed — refresh tokens are single-use (rotation).
func (s *loginService) RefreshToken(ctx context.Context, refreshToken string, sessionID string) (*LoginResponseDTO, error) {
	ctx, span := otel.Tracer("service").Start(ctx, "login.refresh_token")
	defer span.End()

	if strings.TrimSpace(refreshToken) == "" {
		return nil, apperror.NewUnauthorized("refresh token is required")
	}

	// Reuse detection MUST run before the shared validator.
	//
	// A consumed refresh token is recorded in the refresh-scoped `rtused:` key,
	// and rotation is what makes replay detectable. If the shared validator ran
	// first it would reject a replayed token as merely "invalid" — which is what
	// used to happen, because the consumed jti was also written to the generic
	// access-token denylist. That short-circuit made the family-revocation branch
	// below unreachable: a stolen sibling token stayed valid after a replay was
	// observed, contrary to RFC 6819 §5.2.1.1 / OAuth 2.1 §6.1.
	//
	// Claims are read unverified here purely to look up the jti/family; the
	// signature is still verified immediately afterwards, before anything is
	// issued.
	if err := s.rejectRefreshReuse(ctx, refreshToken); err != nil {
		span.SetStatus(codes.Error, "refresh token reuse")
		return nil, err
	}

	// Validate signature, expiry, and denylist via the shared validator.
	claims, err := platformjwt.ValidateTokenWithContext(ctx, refreshToken)
	if err != nil {
		span.SetStatus(codes.Error, "invalid refresh token")
		return nil, apperror.NewUnauthorized("invalid or expired refresh token")
	}

	// Reject access/id tokens presented at the refresh endpoint.
	if tt, _ := claims["token_type"].(string); tt != "refresh_token" {
		return nil, apperror.NewUnauthorized("provided token is not a refresh token")
	}

	sub, _ := claims["sub"].(string)
	clientIdentifier, _ := claims["client_id"].(string)
	if sub == "" || clientIdentifier == "" {
		return nil, apperror.NewUnauthorized("refresh token is missing required claims")
	}

	// Resolve the user behind the refresh token. sub is the user-identity sub and
	// clientIdentifier is the client's identifier — the same pair the token was
	// minted from.
	user, err := s.userRepo.FindBySubAndClientID(sub, clientIdentifier)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user lookup failed")
		return nil, err
	}
	if user == nil {
		return nil, apperror.NewUnauthorized("user not found for refresh token")
	}

	// Resolve the client so the new token set carries the correct issuer/audience.
	//
	// Identify by client identifier ALONE. clients.identifier is globally unique
	// (see migration 019: "an OAuth client_id is resolved without a tenant
	// predicate"), so the provider join adds no identification — it only adds a
	// way to fail. It did: provider_id is a realm value, not an
	// identity_providers.identifier, so the join matched nothing and refresh
	// broke permanently after the first rotation.
	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientIdentifier, "")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client lookup failed")
		return nil, err
	}
	if client == nil {
		return nil, apperror.NewUnauthorized("client not found for refresh token")
	}

	// Resolve session policy before any mutate operations so we can conditionally
	// skip reuse detection and denylisting when token rotation is disabled.
	policy := resolveEffectiveSessionPolicy(s.securitySettingRepo, client)
	tokenPolicy := resolveEffectiveTokenPolicy(s.securitySettingRepo, client)

	// The session comes from the SIGNED refresh token, not the caller.
	//
	// The transport-supplied value (X-Session-ID header, or the sid read out of
	// an unverified — possibly expired — access-token cookie) is only a fallback
	// for tokens minted before refresh tokens carried `sid`. Preferring the
	// signed claim means the caller cannot choose which session to attach to.
	boundSessionID, _ := claims["sid"].(string)
	if strings.TrimSpace(boundSessionID) == "" {
		boundSessionID = sessionID
	}

	resolvedSessionID, err := s.resolveRefreshSession(ctx, user, clientTenantID(client), boundSessionID, policy)
	if err != nil {
		span.SetStatus(codes.Error, "session resolution failed")
		return nil, err
	}

	familyID, _ := claims["rfid"].(string)

	acr, _ := claims["acr"].(string)
	if acr == "" {
		acr = platformjwt.ACRLevel1
	}
	amr, _ := claims["amr"].([]any)
	amrValues := make([]string, 0, len(amr))
	for _, v := range amr {
		if s, ok := v.(string); ok {
			amrValues = append(amrValues, s)
		}
	}
	if len(amrValues) == 0 {
		amrValues = []string{platformjwt.AMRPassword}
	}

	accessToken, idToken, newRefreshToken, err := generateTokenSetWithAuthContext(ctx, sub, user, client, tokenAuthContextWithPolicyAndRefreshFamily(amrValues, acr, resolvedSessionID, policy, tokenPolicy, familyID))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token generation failed")
		return nil, err
	}

	// Single-use rotation: deny the consumed refresh token so it cannot be replayed.
	// Skip denylisting when rotation is disabled — the token remains reusable.
	if policy.RotateRefreshTokens {
		s.denylistConsumedRefreshToken(ctx, claims, policy)
	}

	resp := buildLoginTokenResponse(accessToken, idToken, newRefreshToken, time.Now().Unix())
	applyLoginCookiePolicy(resp, policy)
	if policy.AccessTokenTTLSeconds > 0 {
		resp.ExpiresIn = int64(policy.AccessTokenTTLSeconds)
	}
	resp.RequirePasswordChange = user.ForcePasswordChange
	if resolvedSessionID != "" {
		resp.SessionID = &resolvedSessionID
	}

	span.SetStatus(codes.Ok, "")
	return resp, nil
}

// resolveRefreshSession reuses the caller's session when a valid session id is
// supplied (preserving idle/absolute limits), otherwise creates a new one. It
// returns the empty string when no session service is configured.
func (s *loginService) resolveRefreshSession(ctx context.Context, user *User, tenantID int64, sessionID string, policy secpolicy.EffectiveSessionPolicy) (string, error) {
	if s.sessionService == nil {
		return "", nil
	}

	// A refresh NEVER establishes a session.
	//
	// It used to: with no session id supplied it created a fresh one. That made
	// every session-revoking control ineffective against a stolen refresh token —
	// after "sign out everywhere" or a password reset the thief simply omitted
	// the access-token cookie and was issued a brand-new session, with acr
	// silently reset to 1. The refresh token now carries `sid` (see
	// jwt.RefreshTokenOptions), so a token that presents no session is either
	// pre-dating that change or forged; both must re-authenticate.
	if strings.TrimSpace(sessionID) == "" {
		return "", apperror.NewUnauthorized("refresh token is not bound to a session")
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return "", apperror.NewUnauthorized("invalid session id")
	}
	if err := s.sessionService.ValidateAndTouch(ctx, sessionUUID, user.UserID); err != nil {
		// Session revoked or past its idle/absolute limit — require re-login.
		return "", apperror.NewUnauthorized("session is no longer valid")
	}
	return sessionID, nil
}

// denylistConsumedRefreshToken best-effort denylists the refresh token's JTI for
// its remaining lifetime so a rotated token cannot be replayed.
func (s *loginService) denylistConsumedRefreshToken(ctx context.Context, claims jwtlib.MapClaims, policy secpolicy.EffectiveSessionPolicy) {
	if s.jtiDenylist == nil {
		return
	}
	jti, _ := claims["jti"].(string)
	if strings.TrimSpace(jti) == "" {
		return
	}
	ttl := jwtClaimTTL(claims["exp"])
	if ttl <= 0 {
		return
	}
	// Deliberately NOT the bare jti: that namespace is the generic access-token
	// denylist consulted by ValidateTokenWithContext, and writing to it made a
	// replayed refresh token fail validation before reuse detection could run.
	// The refresh-scoped key below is the one rejectRefreshReuse reads.
	if err := s.jtiDenylist.DenyJTI(ctx, refreshUsedKey(jti), ttl); err != nil {
		// Fail loudly rather than silently leaving the parent replayable.
		slog.Error("refresh rotation: failed to mark token consumed",
			"jti", jti, "error", err)
	}
	if policy.RefreshTokenReuseIntervalSeconds > 0 {
		_ = s.jtiDenylist.DenyJTI(ctx, refreshGraceKey(jti), time.Duration(policy.RefreshTokenReuseIntervalSeconds)*time.Second)
	}
}

func (s *loginService) rejectRefreshReuse(ctx context.Context, refreshToken string) error {
	if s.jtiDenylist == nil {
		return nil
	}
	claims := unverifiedRefreshClaims(refreshToken)
	if len(claims) == 0 {
		return nil
	}
	jti, _ := claims["jti"].(string)
	familyID, _ := claims["rfid"].(string)
	if familyID != "" {
		if denied, _ := s.jtiDenylist.IsJTIDenied(ctx, refreshFamilyKey(familyID)); denied {
			return apperror.NewUnauthorized("refresh token family has been revoked")
		}
	}
	if strings.TrimSpace(jti) == "" {
		return nil
	}
	used, _ := s.jtiDenylist.IsJTIDenied(ctx, refreshUsedKey(jti))
	if !used {
		used, _ = s.jtiDenylist.IsJTIDenied(ctx, jti)
	}
	if !used {
		return nil
	}

	// The grace window may soften the ERROR SHAPE for a client that legitimately
	// retried, but it must never suppress the RESPONSE TO COMPROMISE. Revoking
	// the family happens first, unconditionally: an attacker replays a stolen
	// token immediately, which is squarely inside the window, so returning early
	// here disabled detection at exactly the highest-risk moment.
	inGrace, _ := s.jtiDenylist.IsJTIDenied(ctx, refreshGraceKey(jti))

	if familyID != "" {
		ttl := jwtClaimTTL(claims["exp"])
		if ttl <= 0 {
			ttl = platformjwt.RefreshTokenTTL
		}
		_ = s.jtiDenylist.DenyJTI(ctx, refreshFamilyKey(familyID), ttl)
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "refresh_token_reuse",
			UserID:    stringClaim(claims, "sub"),
			ClientID:  stringClaim(claims, "client_id"),
			ClientIP:  middleware.ClientIPFromContext(ctx),
			UserAgent: middleware.UserAgentFromContext(ctx),
			Timestamp: time.Now(),
			Details:   "Refresh token reuse detected; token family revoked",
			Severity:  "HIGH",
		})
	}
	if inGrace {
		// Distinct message for an in-window replay (most often a client that
		// retried), but the family is already revoked above either way.
		return apperror.NewUnauthorized("refresh token was already consumed")
	}
	return apperror.NewUnauthorized("refresh token reuse detected")
}

func unverifiedRefreshClaims(tokenString string) jwtlib.MapClaims {
	token, _, err := jwtlib.NewParser().ParseUnverified(tokenString, jwtlib.MapClaims{})
	if err != nil {
		return nil
	}
	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		return nil
	}
	return claims
}

func refreshUsedKey(jti string) string { return "rtused:" + strings.TrimSpace(jti) }

func refreshGraceKey(jti string) string { return "rtgrace:" + strings.TrimSpace(jti) }

func refreshFamilyKey(familyID string) string { return "rtfam:" + strings.TrimSpace(familyID) }

func stringClaim(claims jwtlib.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}

// sessionIDFromAccessToken extracts the sid claim from a (typically expired)
// access token without verifying its signature or expiry. This lets
// cookie-based clients preserve session continuity on refresh without sending
// an explicit X-Session-ID header.
func sessionIDFromAccessToken(accessToken string) string {
	if strings.TrimSpace(accessToken) == "" {
		return ""
	}
	token, _, err := jwtlib.NewParser().ParseUnverified(accessToken, jwtlib.MapClaims{})
	if err != nil {
		return ""
	}
	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		return ""
	}
	sid, _ := claims["sid"].(string)
	return sid
}
