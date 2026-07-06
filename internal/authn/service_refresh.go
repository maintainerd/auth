package authn

import (
	"context"
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
	providerIdentifier, _ := claims["provider_id"].(string)
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
	client, err := s.clientRepo.FindByClientIDAndIdentityProvider(clientIdentifier, providerIdentifier)
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

	if policy.RotateRefreshTokens {
		if err := s.rejectRefreshReuse(ctx, refreshToken); err != nil {
			return nil, err
		}
	}

	// Bind the new tokens to a session (reuse the caller's, or start a new one).
	resolvedSessionID, err := s.resolveRefreshSession(ctx, user, sessionID, policy)
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
func (s *loginService) resolveRefreshSession(ctx context.Context, user *User, sessionID string, policy secpolicy.EffectiveSessionPolicy) (string, error) {
	if s.sessionService == nil {
		return "", nil
	}

	if strings.TrimSpace(sessionID) != "" {
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

	// No session supplied — establish a new one, mirroring the login flow.
	if err := enforceConcurrentLimitWithPolicy(ctx, s.sessionService, user.UserUUID, user.UserID, policy); err != nil {
		return "", err
	}
	sess, err := createSessionWithPolicy(ctx, s.sessionService, user.UserID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), policy)
	if err != nil {
		return "", err
	}
	return sess.UserSessionUUID.String(), nil
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
	_ = s.jtiDenylist.DenyJTI(ctx, jti, ttl)
	_ = s.jtiDenylist.DenyJTI(ctx, refreshUsedKey(jti), ttl)
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
	if inGrace, _ := s.jtiDenylist.IsJTIDenied(ctx, refreshGraceKey(jti)); inGrace {
		return apperror.NewUnauthorized("refresh token was already consumed")
	}
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
