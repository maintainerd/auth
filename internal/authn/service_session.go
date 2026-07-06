package authn

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	defaultIdleTimeoutSeconds    = 1800
	defaultAbsoluteLifetimeHours = 24
)

type SessionDataResult struct {
	SessionID    string     `json:"session_id"`
	IPAddress    *string    `json:"ip_address,omitempty"`
	UserAgent    *string    `json:"user_agent,omitempty"`
	LastActiveAt *time.Time `json:"last_active_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type SessionService interface {
	ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error)
	RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error
	RevokeAllSessions(ctx context.Context, userID int64) error
	CreateSession(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string) (*UserSession, error)
	EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error
	ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error
}

type sessionService struct {
	sessionRepo UserSessionRepository
}

func NewSessionService(sessionRepo UserSessionRepository) SessionService {
	return &sessionService{sessionRepo: sessionRepo}
}

func (s *sessionService) ListSessions(ctx context.Context, userID int64) ([]*SessionDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "session.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	sessions, err := s.sessionRepo.FindActiveByUserID(userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list sessions failed")
		return nil, err
	}

	result := make([]*SessionDataResult, len(sessions))
	for i := range sessions {
		result[i] = toSessionDataResult(&sessions[i])
	}
	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (s *sessionService) RevokeSession(ctx context.Context, userID int64, sessionUUID uuid.UUID) error {
	_, span := otel.Tracer("service").Start(ctx, "session.revoke")
	defer span.End()

	sess, err := s.sessionRepo.FindActiveByUUID(userID, sessionUUID)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if sess == nil {
		return apperror.NewNotFound("session not found")
	}

	if err := s.sessionRepo.RevokeByUUID(userID, sessionUUID, "logout"); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *sessionService) RevokeAllSessions(ctx context.Context, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "session.revokeAll")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	if err := s.sessionRepo.RevokeAllByUserID(userID); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *sessionService) CreateSession(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string) (*UserSession, error) {
	return s.CreateSessionWithPolicy(ctx, userID, tenantID, ipAddress, userAgent, defaultEffectiveSessionPolicy())
}

func (s *sessionService) CreateSessionWithPolicy(ctx context.Context, userID, tenantID int64, ipAddress, userAgent string, policy secpolicy.EffectiveSessionPolicy) (*UserSession, error) {
	_, span := otel.Tracer("service").Start(ctx, "session.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	now := time.Now()
	if policy.IdleTimeoutSeconds <= 0 {
		policy.IdleTimeoutSeconds = defaultIdleTimeoutSeconds
	}
	if policy.AbsoluteTimeoutSeconds <= 0 {
		policy.AbsoluteTimeoutSeconds = int((defaultAbsoluteLifetimeHours * time.Hour).Seconds())
	}

	var ipPtr *string
	if ipAddress != "" {
		ipPtr = &ipAddress
	}
	var uaPtr *string
	if userAgent != "" {
		uaPtr = &userAgent
	}

	session := &UserSession{
		UserID:             userID,
		TenantID:           tenantID,
		AuthTime:           now,
		IPAddress:          ipPtr,
		UserAgent:          uaPtr,
		ACR:                "1",
		IdleTimeoutSeconds: policy.IdleTimeoutSeconds,
		LastActiveAt:       now,
		ExpiresAt:          now.Add(time.Duration(policy.AbsoluteTimeoutSeconds) * time.Second),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return session, nil
}

func (s *sessionService) EnforceConcurrentLimit(ctx context.Context, userUUID uuid.UUID, userID int64) error {
	return s.EnforceConcurrentLimitWithPolicy(ctx, userUUID, userID, defaultEffectiveSessionPolicy())
}

func (s *sessionService) EnforceConcurrentLimitWithPolicy(ctx context.Context, userUUID uuid.UUID, userID int64, policy secpolicy.EffectiveSessionPolicy) error {
	_, span := otel.Tracer("service").Start(ctx, "session.enforceConcurrentLimit")
	defer span.End()

	count, err := s.sessionRepo.CountActive(userID)
	if err != nil {
		span.RecordError(err)
		return err
	}

	if policy.MaxConcurrentSessions <= 0 || count < int64(policy.MaxConcurrentSessions) {
		return nil
	}

	sessions, err := s.sessionRepo.FindActiveByUserID(userID)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if len(sessions) == 0 {
		return nil
	}

	oldest := sessions[0]
	if err := s.sessionRepo.RevokeByUUID(userID, oldest.UserSessionUUID, "concurrent_limit"); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *sessionService) ValidateAndTouch(ctx context.Context, sessionUUID uuid.UUID, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "session.validateAndTouch")
	defer span.End()

	sess, err := s.sessionRepo.FindActiveByUUID(userID, sessionUUID)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if sess == nil {
		return apperror.NewUnauthorized("session not found or has been revoked")
	}

	now := time.Now()

	if !sess.ExpiresAt.IsZero() && now.After(sess.ExpiresAt) {
		_ = s.sessionRepo.RevokeByUUID(userID, sessionUUID, "session_expired")
		return apperror.NewUnauthorized("session has expired")
	}

	idleSince := now.Sub(sess.LastActiveAt)
	if idleSince > time.Duration(sess.IdleTimeoutSeconds)*time.Second {
		_ = s.sessionRepo.RevokeByUUID(userID, sessionUUID, "session_expired")
		return apperror.NewUnauthorized("session has expired due to inactivity")
	}

	if err := s.sessionRepo.Touch(sess.UserSessionID, now); err != nil {
		span.RecordError(err)
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

func toSessionDataResult(s *UserSession) *SessionDataResult {
	return &SessionDataResult{
		SessionID:    s.UserSessionUUID.String(),
		IPAddress:    s.IPAddress,
		UserAgent:    s.UserAgent,
		LastActiveAt: &s.LastActiveAt,
		ExpiresAt:    &s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
	}
}
