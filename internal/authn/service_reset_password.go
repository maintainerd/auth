package authn

import (
	"context"
	"fmt"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

type ResetPasswordService interface {
	ResetPassword(ctx context.Context, token, newPassword string, clientID, tenantID *string) (*ResetPasswordResponseDTO, error)
}

type resetPasswordService struct {
	db                  *gorm.DB
	userRepo            UserRepository
	userTokenRepo       UserTokenRepository
	clientRepo          ClientRepository
	securitySettingRepo secpolicy.SecuritySettingRepository // nil → use defaults
	passwordHistoryRepo UserPasswordHistoryRepository       // nil → skip history
	// sessionRepo is the CANONICAL session store (user_sessions), the one
	// UserContextMiddleware validates every request against. Revoking anywhere
	// else does not end a login.
	sessionRepo UserSessionRepository
}

func NewResetPasswordService(
	db *gorm.DB,
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	clientRepo ClientRepository,
	securitySettingRepo secpolicy.SecuritySettingRepository,
	passwordHistoryRepo UserPasswordHistoryRepository,
	sessionRepo UserSessionRepository,
) ResetPasswordService {
	return &resetPasswordService{
		db:                  db,
		userRepo:            userRepo,
		userTokenRepo:       userTokenRepo,
		clientRepo:          clientRepo,
		securitySettingRepo: securitySettingRepo,
		passwordHistoryRepo: passwordHistoryRepo,
		sessionRepo:         sessionRepo,
	}
}

var resetHashPasswordWithPolicy = security.HashPasswordWithPolicy

func (s *resetPasswordService) ResetPassword(ctx context.Context, token, newPassword string, clientID, tenantID *string) (*ResetPasswordResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "password.reset")
	defer span.End()

	var user *User
	var userToken *UserToken

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txUserTokenRepo := s.userTokenRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)

		// Validate auth client first
		var Client *Client
		var txErr error
		switch {
		case clientID != nil && tenantID != nil:
			// Compatibility for already-issued legacy links whose second value was
			// the provider identifier.
			Client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(*clientID, *tenantID)
		case publicAuthSurfaceFromContext(ctx):
			Client, txErr = resolvePublicClient(ctx, txClientRepo, clientID, tenantID)
		case clientID != nil:
			Client, txErr = txClientRepo.FindByIdentifier(*clientID)
		case tenantID != nil:
			Client, txErr = resolveClient(txClientRepo, nil, tenantID)
		default:
			Client, txErr = txClientRepo.FindSystem()
		}
		if txErr != nil {
			return apperror.NewInternal("failed to find auth client", txErr)
		}
		if Client == nil {
			return apperror.NewUnauthorized("invalid client credentials")
		}

		// Find the reset token by searching all password reset tokens
		// Note: This is not the most efficient approach, but works with current repository methods
		var foundToken *UserToken

		// We need to find all password reset tokens and check which one matches our token
		// This is a security consideration - we don't want to reveal if a token exists
		allTokens := []UserToken{}
		txErr = tx.Where("token_type = ? AND token = ? AND is_revoked = false", shared.TokenTypePasswordReset, hashUserBearerToken(token)).Find(&allTokens).Error
		if txErr != nil {
			return apperror.NewInternal("failed to find reset token", txErr)
		}

		if len(allTokens) == 0 {
			return apperror.NewUnauthorized("invalid or expired reset token")
		}

		foundToken = &allTokens[0]
		userToken = foundToken

		// Check if token is expired
		if userToken.ExpiresAt != nil && time.Now().After(*userToken.ExpiresAt) {
			return apperror.NewUnauthorized("reset token has expired")
		}

		// Check if token is revoked
		if userToken.IsRevoked {
			return apperror.NewUnauthorized("reset token has been revoked")
		}

		// Find the user
		user, txErr = txUserRepo.FindByID(userToken.UserID)
		if txErr != nil {
			return apperror.NewInternal("failed to find user", txErr)
		}
		if user == nil {
			return apperror.NewNotFound("user not found")
		}

		if user.TenantID != clientTenantID(Client) {
			return apperror.NewUnauthorized("invalid or expired reset token")
		}

		// Check if user is active
		if user.Status != shared.StatusActive {
			return apperror.NewUnauthorized("user account is not active")
		}

		// Validate password against tenant policy
		tenantID := clientTenantID(Client)
		policy := secpolicy.LoadPasswordPolicy(s.securitySettingRepo, tenantID)
		if err := security.ValidatePasswordPolicyWithContext(ctx, newPassword, policy); err != nil {
			return apperror.NewValidation(err.Error())
		}

		// Check password history
		if err := checkPasswordHistory(s.passwordHistoryRepo, user.UserID, policy.HistoryCount, newPassword); err != nil {
			return apperror.NewValidation(err.Error())
		}

		// Hash the new password
		hashedPassword, txErr := resetHashPasswordWithPolicy(ctx, []byte(newPassword), policy)
		if txErr != nil {
			return apperror.NewInternal("failed to hash password", txErr)
		}

		now := time.Now()
		// Update user password using the base repository method
		_, txErr = txUserRepo.UpdateByID(user.UserID, map[string]any{
			"password":                      string(hashedPassword),
			"force_password_change":         false,
			"password_changed_at":           now,
			"temporary_password_expires_at": nil,
		})
		if txErr != nil {
			return apperror.NewInternal("failed to update password", txErr)
		}

		// Record new hash in history, transaction-scoped and fail-closed. A
		// dropped entry is invisible and permanent: the row that was supposed
		// to stop the user cycling straight back to this password would simply
		// not exist, and nothing would ever report it.
		historyRepo := s.passwordHistoryRepo
		if historyRepo != nil {
			historyRepo = historyRepo.WithTx(tx)
		}
		if txErr = secpolicy.RecordPasswordHistory(historyRepo, user.UserID, policy.HistoryCount, string(hashedPassword)); txErr != nil {
			return apperror.NewInternal("failed to record password history", txErr)
		}

		// Revoke every session so a password reset actually ends the attacker's
		// access, unless the tenant has opted out via
		// revoke_sessions_on_password_change=false.
		//
		// This used to call txUserTokenRepo.RevokeAllSessionsByUserID, which
		// updates user_tokens rows of token_type 'user:session'. The login flow
		// stopped writing those rows when sessions moved to the user_sessions
		// table, so nothing matched the UPDATE and a reset logged NOBODY out —
		// the exact opposite of what the control exists for, and silent because
		// updating zero rows is not an error. It now targets the canonical
		// session store that UserContextMiddleware actually validates against.
		if secpolicy.ShouldRevokeSessionsOnPasswordChange(s.securitySettingRepo, tenantID) {
			if s.sessionRepo == nil {
				return apperror.NewInternal("cannot revoke sessions on password reset: no session repository configured", nil)
			}
			if txErr = s.sessionRepo.WithTx(tx).RevokeAllByUserID(user.UserID, "password reset"); txErr != nil {
				return apperror.NewInternal("failed to revoke sessions on password reset", txErr)
			}
		}

		// Revoke the reset token
		txErr = txUserTokenRepo.RevokeByUUID(userToken.UserTokenUUID)
		if txErr != nil {
			return apperror.NewInternal("failed to revoke reset token", txErr)
		}

		// Revoke all other password reset tokens for this user
		existingTokens, txErr := txUserTokenRepo.FindByUserIDAndTokenType(user.UserID, shared.TokenTypePasswordReset)
		if txErr != nil {
			return apperror.NewInternal("failed to find existing tokens", txErr)
		}
		for _, existingToken := range existingTokens {
			if existingToken.UserTokenUUID != userToken.UserTokenUUID {
				if txErr := txUserTokenRepo.RevokeByUUID(existingToken.UserTokenUUID); txErr != nil {
					return apperror.NewInternal("failed to revoke existing token", txErr)
				}
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "reset password failed")
		// authevent.Log security event for failed password reset
		// NEVER log the raw token. It is a bearer credential, and several failure
		// paths here leave it VALID and unused — a policy rejection, a history
		// rejection, or an inactive account. Writing it to the application log handed
		// anyone with log or SIEM access a working account-takeover token for the
		// remainder of its TTL. The PII redactor cannot help: it matches on key
		// names, and this was logged under "user_id".
		failureSubject := "unknown"
		if user != nil {
			failureSubject = user.UserUUID.String()
		}
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "password_reset_failure",
			UserID:    failureSubject,
			Details:   fmt.Sprintf("Password reset failed: %v", err),
			Severity:  "HIGH",
			Timestamp: time.Now(),
		})
		return nil, err
	}

	// authevent.Log successful password reset
	security.LogSecurityEvent(security.SecurityEvent{
		EventType: "password_reset_success",
		UserID:    user.UserUUID.String(),
		Details:   "Password reset completed successfully",
		Severity:  "INFO",
		Timestamp: time.Now(),
	})

	// Reset failed login attempts for this user
	security.ResetFailedAttempts(user.Email)

	span.SetStatus(codes.Ok, "")
	return &ResetPasswordResponseDTO{
		Message: "Password has been reset successfully. You can now log in with your new password.",
		Success: true,
	}, nil
}
