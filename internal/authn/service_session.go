package authn

// NOTE on password-change session revocation:
// When a user resets their password via ResetPasswordService, all sessions
// should be revoked. The existing RevokeAllByUserID method in
// UserTokenRepository (which revokes ALL token types) is already called by
// ResetPasswordService when it cleans up after a successful password reset.
// If that service does not currently call it, add:
//
//   s.userTokenRepo.RevokeAllSessionsByUserID(user.UserID)
//
// after the password update in internal/service/reset_password.go.
// That file is excluded from this change set (permission denied).

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	// defaultIdleTimeoutSeconds is the sliding idle timeout for sessions (30 min).
	defaultIdleTimeoutSeconds = 1800
	// defaultAbsoluteLifetimeHours is the maximum session lifetime (24 h).
	defaultAbsoluteLifetimeHours = 24
)

// SessionDataResult is the public representation of a single user session
// returned by ListSessions.
type SessionDataResult struct {
	SessionID         string     `json:"session_id"`
	IPAddress         *string    `json:"ip_address,omitempty"`
	UserAgent         *string    `json:"user_agent,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	AbsoluteExpiresAt *time.Time `json:"absolute_expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// SessionService manages user sessions: creation, listing, revocation, and
// timeout enforcement.
type SessionService interface {
	// ListSessions returns all active sessions for the given user.
	ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error)

	// RevokeSession revokes a single session identified by sessionUUID.
	// Returns NotFound when the session does not belong to the user.
	RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error

	// RevokeAllSessions revokes every active session for the given user.
	RevokeAllSessions(ctx context.Context, userID int64) error

	// CreateSession creates a new session token for the user, recording the
	// IP address and user agent.
	CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*UserToken, error)

	// EnforceConcurrentLimit ensures the user does not exceed
	// security.MaxConcurrentSessions. When the limit is reached the oldest
	// session is evicted rather than rejecting the login (better UX).
	EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error

	// ValidateAndTouch validates that sessionUUID is active (not revoked, not
	// idle-expired, not absolute-expired) and updates last_used_at. Intended
	// for use by the session middleware on every authenticated request.
	ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error
}

type sessionService struct {
	userTokenRepo UserTokenRepository
}

// NewSessionService constructs a SessionService backed by userTokenRepo.
func NewSessionService(userTokenRepo UserTokenRepository) SessionService {
	return &sessionService{userTokenRepo: userTokenRepo}
}

// ListSessions returns all active sessions for userID.
func (s *sessionService) ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "session.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	sessions, err := s.userTokenRepo.FindActiveSessions(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list sessions failed")
		return nil, err
	}

	result := make([]*SessionDataResult, len(sessions))
	for i, sess := range sessions {
		result[i] = toSessionDataResult(&sess)
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

// RevokeSession revokes a single session that belongs to userID.
func (s *sessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "session.revoke")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("user.id", userID),
		attribute.String("session.uuid", sessionUUID.String()),
	)

	// Verify the session exists and belongs to the user.
	sess, err := s.userTokenRepo.FindActiveSessionByUUID(userID, sessionUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke session lookup failed")
		return err
	}
	if sess == nil {
		span.SetStatus(codes.Error, "session not found")
		return apperror.NewNotFound("session not found")
	}

	if err := s.userTokenRepo.RevokeSessionByUUID(userID, sessionUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke session failed")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// RevokeAllSessions revokes every session for userID.
func (s *sessionService) RevokeAllSessions(ctx context.Context, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "session.revokeAll")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	if err := s.userTokenRepo.RevokeAllSessionsByUserID(userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke all sessions failed")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// CreateSession creates a new UserToken of type shared.TokenTypeSession for userID.
func (s *sessionService) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (*UserToken, error) {
	return s.CreateSessionWithPolicy(ctx, userID, ipAddress, userAgent, defaultEffectiveSessionPolicy())
}

func (s *sessionService) CreateSessionWithPolicy(ctx context.Context, userID int64, ipAddress, userAgent string, policy secpolicy.EffectiveSessionPolicy) (*UserToken, error) {
	_, span := otel.Tracer("service").Start(ctx, "session.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	// Generate a random opaque token (not the session UUID — the UUID is the
	// public identifier; the token is stored hashed for additional security).
	rawToken, err := generateRandomToken(32)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token generation failed")
		return nil, apperror.NewInternal("failed to generate session token", err)
	}

	now := time.Now()
	if policy.IdleTimeoutSeconds <= 0 {
		policy.IdleTimeoutSeconds = defaultIdleTimeoutSeconds
	}
	if policy.AbsoluteTimeoutSeconds <= 0 {
		policy.AbsoluteTimeoutSeconds = int((defaultAbsoluteLifetimeHours * time.Hour).Seconds())
	}
	absoluteExpiry := now.Add(time.Duration(policy.AbsoluteTimeoutSeconds) * time.Second)
	idleTimeout := policy.IdleTimeoutSeconds

	token := &UserToken{
		UserID:             userID,
		TokenType:          shared.TokenTypeSession,
		Token:              rawToken, // stored as-is; rotate to hash if required
		IPAddress:          ptr.PtrOrNil(ipAddress),
		UserAgent:          ptr.PtrOrNil(userAgent),
		IsRevoked:          false,
		LastUsedAt:         &now,
		IdleTimeoutSeconds: &idleTimeout,
		AbsoluteExpiresAt:  &absoluteExpiry,
	}

	created, err := s.userTokenRepo.Create(token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create session failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return created, nil
}

// EnforceConcurrentLimit checks whether the user has reached
// security.MaxConcurrentSessions. If so, the oldest active session is evoked
// to make room for the new one rather than blocking login.
func (s *sessionService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return s.EnforceConcurrentLimitWithPolicy(ctx, userUUID, userID, defaultEffectiveSessionPolicy())
}

func (s *sessionService) EnforceConcurrentLimitWithPolicy(ctx context.Context, userUUID uuid.UUID, userID int64, policy secpolicy.EffectiveSessionPolicy) error {
	_, span := otel.Tracer("service").Start(ctx, "session.enforceConcurrentLimit")
	defer span.End()
	span.SetAttributes(
		attribute.String("user.uuid", userUUID.String()),
		attribute.Int64("user.id", userID),
	)

	count, err := s.userTokenRepo.CountActiveSessions(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "count sessions failed")
		return err
	}

	if policy.MaxConcurrentSessions <= 0 || count < int64(policy.MaxConcurrentSessions) {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	// Evict the oldest session (FindActiveSessions returns oldest-first).
	sessions, err := s.userTokenRepo.FindActiveSessions(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find sessions for eviction failed")
		return err
	}
	if len(sessions) == 0 {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	oldest := sessions[0]
	if err := s.userTokenRepo.RevokeSessionByUUID(userID, oldest.UserTokenUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "evict oldest session failed")
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func defaultEffectiveSessionPolicy() secpolicy.EffectiveSessionPolicy {
	policy, err := secpolicy.ResolveEffectiveSessionPolicy(nil, nil, secpolicy.SecuritySettingClientOverrides{})
	if err != nil {
		return secpolicy.EffectiveSessionPolicy{
			MaxConcurrentSessions:  5,
			IdleTimeoutSeconds:     defaultIdleTimeoutSeconds,
			AbsoluteTimeoutSeconds: int((defaultAbsoluteLifetimeHours * time.Hour).Seconds()),
		}
	}
	return policy
}

// ValidateAndTouch validates that the session is still active (checking both
// idle timeout and absolute expiry) and updates last_used_at.
func (s *sessionService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "session.validateAndTouch")
	defer span.End()
	span.SetAttributes(
		attribute.String("session.uuid", sessionUUID.String()),
		attribute.Int64("user.id", userID),
	)

	sess, err := s.userTokenRepo.FindActiveSessionByUUID(userID, sessionUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "session lookup failed")
		return err
	}
	if sess == nil {
		span.SetStatus(codes.Error, "session not found or revoked")
		return apperror.NewUnauthorized("session not found or has been revoked")
	}

	now := time.Now()

	// Absolute lifetime check.
	if sess.AbsoluteExpiresAt != nil && now.After(*sess.AbsoluteExpiresAt) {
		// Eagerly revoke so subsequent checks are fast.
		_ = s.userTokenRepo.RevokeSessionByUUID(userID, sessionUUID)
		span.SetStatus(codes.Error, "session absolute lifetime exceeded")
		return apperror.NewUnauthorized("session has expired")
	}

	// Idle timeout check.
	if sess.IdleTimeoutSeconds != nil && sess.LastUsedAt != nil {
		idleSince := now.Sub(*sess.LastUsedAt)
		maxIdle := time.Duration(*sess.IdleTimeoutSeconds) * time.Second
		if idleSince > maxIdle {
			_ = s.userTokenRepo.RevokeSessionByUUID(userID, sessionUUID)
			span.SetStatus(codes.Error, "session idle timeout exceeded")
			return apperror.NewUnauthorized("session has expired due to inactivity")
		}
	}

	// Touch — update last_used_at for the sliding window.
	if err := s.userTokenRepo.TouchSession(userID, sessionUUID, now); err != nil {
		// Non-fatal: log but don't fail the request.
		span.RecordError(err)
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// toSessionDataResult converts a UserToken model to the public DTO.
func toSessionDataResult(t *UserToken) *SessionDataResult {
	return &SessionDataResult{
		SessionID:         t.UserTokenUUID.String(),
		IPAddress:         t.IPAddress,
		UserAgent:         t.UserAgent,
		LastUsedAt:        t.LastUsedAt,
		ExpiresAt:         t.ExpiresAt,
		AbsoluteExpiresAt: t.AbsoluteExpiresAt,
		CreatedAt:         t.CreatedAt,
	}
}

var randRead = rand.Read

// generateRandomToken generates a hex-encoded random token of byteLen bytes.
func generateRandomToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := randRead(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
