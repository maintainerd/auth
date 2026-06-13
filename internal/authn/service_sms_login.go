package authn

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/notifier"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/sms"
	"github.com/maintainerd/auth/internal/secpolicy"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const smsOTPLength = 6
const smsOTPTTL = 10 * time.Minute
const smsOTPMaxFailedAttempts = 3

var generateSMSOTP = crypto.GenerateOTP
var newSMSProvider = sms.NewProviderFromDB

// SMSLoginService handles SMS one-time-code login flows.
type SMSLoginService interface {
	SendOTP(ctx context.Context, req SMSLoginSendDTO) error
	VerifyOTP(ctx context.Context, req SMSLoginVerifyDTO) (*LoginResponseDTO, error)
}

type smsLoginService struct {
	db                   *gorm.DB
	userRepo             UserRepository
	smsOtpRepo           notifier.UserOTPRepository
	clientRepo           ClientRepository
	userIdentityRepo     UserIdentityRepository
	identityProviderRepo IdentityProviderRepository
	authEventService     authevent.AuthEventService
	sessionService       SessionService
	securitySettingRepo  secpolicy.SecuritySettingRepository
}

func NewSMSLoginService(
	db *gorm.DB,
	userRepo UserRepository,
	smsOtpRepo notifier.UserOTPRepository,
	clientRepo ClientRepository,
	userIdentityRepo UserIdentityRepository,
	identityProviderRepo IdentityProviderRepository,
	authEventService authevent.AuthEventService,
	options ...any,
) SMSLoginService {
	var sessions SessionService
	var settings secpolicy.SecuritySettingRepository
	for _, option := range options {
		switch v := option.(type) {
		case SessionService:
			sessions = v
		case secpolicy.SecuritySettingRepository:
			settings = v
		}
	}
	return &smsLoginService{
		db:                   db,
		userRepo:             userRepo,
		smsOtpRepo:           smsOtpRepo,
		clientRepo:           clientRepo,
		userIdentityRepo:     userIdentityRepo,
		identityProviderRepo: identityProviderRepo,
		authEventService:     authEventService,
		sessionService:       sessions,
		securitySettingRepo:  settings,
	}
}

// SendOTP looks up the user by phone, generates a 6-digit OTP, stores its hash,
// and logs it (real SMS provider integration is a future TODO).
func (s *smsLoginService) SendOTP(ctx context.Context, req SMSLoginSendDTO) error {
	_, span := otel.Tracer("service").Start(ctx, "smsLogin.sendOTP")
	defer span.End()

	if err := security.CheckRateLimit("sms-otp:send:" + req.Phone); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "sms_otp_rate_limited",
			UserID:    req.Phone,
			Timestamp: time.Now(),
			Details:   err.Error(),
		})
		return err
	}

	// Look up user by phone — respond generically so we don't leak user existence.
	user, err := s.userRepo.FindByPhone(req.Phone)
	if err != nil {
		return apperror.NewInternal("failed to look up user", err)
	}
	if user == nil || user.Status != shared.StatusActive {
		// Still return success to avoid phone enumeration.
		span.SetStatus(codes.Ok, "")
		return nil
	}

	// Threat check before sending SMS — block if velocity threshold is breached.
	tenantID := userTenantID(ctx, s.db, user.UserID)
	threatPolicy := secpolicy.LoadThreatPolicy(s.securitySettingRepo, tenantID)
	threatDecision := security.AssessLoginThreat(ctx, tenantID, middleware.ClientIPFromContext(ctx), "", threatPolicy)
	if threatDecision.Blocked {
		span.SetStatus(codes.Ok, "")
		return nil // fail silently to avoid enumeration
	}

	if err := security.CheckAndRecordSMSDailyBudget(ctx, "global", smsDailySendLimit(s.db, 0)); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "sms_otp_budget_exceeded",
			UserID:    req.Phone,
			Timestamp: time.Now(),
			Details:   err.Error(),
		})
		return apperror.NewValidation("SMS send budget exceeded")
	}

	// Generate OTP and hash it for storage.
	otp, err := generateSMSOTP(smsOTPLength)
	if err != nil {
		return apperror.NewInternal("failed to generate OTP", err)
	}
	otpHash := crypto.HashAuthorizationCode(otp)

	expiresAt := time.Now().Add(smsOTPTTL)
	record := &notifier.UserOTP{
		UserID:    user.UserID,
		Channel:   "sms",
		Recipient: req.Phone,
		OTPHash:   otpHash,
		ExpiresAt: expiresAt,
	}
	if _, err := s.smsOtpRepo.Create(record); err != nil {
		return apperror.NewInternal("failed to store SMS OTP", err)
	}

	provider, smsErr := newSMSProvider(ctx, s.db, tenantID)
	if smsErr != nil {
		slog.Warn("SMS provider init failed — logging OTP for dev", "err", smsErr, "phone", req.Phone, "otp", otp)
	} else if provider != nil {
		if sendErr := provider.Send(ctx, req.Phone, fmt.Sprintf("Your verification code is: %s", otp)); sendErr != nil {
			slog.Error("SMS send failed", "err", sendErr)
		}
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// VerifyOTP validates the submitted OTP and issues tokens on success.
func (s *smsLoginService) VerifyOTP(ctx context.Context, req SMSLoginVerifyDTO) (*LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsLogin.verifyOTP")
	defer span.End()

	if err := security.CheckRateLimit("sms-otp:verify:" + req.Phone); err != nil {
		security.LogSecurityEvent(security.SecurityEvent{
			EventType: "sms_otp_verify_rate_limited",
			UserID:    req.Phone,
			Timestamp: time.Now(),
			Details:   err.Error(),
		})
		return nil, err
	}

	var user *User
	var client *Client
	var userIdentitySub string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txUserRepo := s.userRepo.WithTx(tx)
		txClientRepo := s.clientRepo.WithTx(tx)
		txIdpRepo := s.identityProviderRepo.WithTx(tx)
		txUserIdentityRepo := s.userIdentityRepo.WithTx(tx)
		txSmsOtpRepo := s.smsOtpRepo.WithTx(tx)

		// Validate identity provider and client.
		idp, txErr := txIdpRepo.FindByIdentifier(req.ProviderID)
		if txErr != nil || idp == nil {
			return apperror.NewUnauthorized("authentication failed")
		}

		client, txErr = txClientRepo.FindByClientIDAndIdentityProvider(req.ClientID, req.ProviderID)
		if txErr != nil || client == nil ||
			client.Status != shared.StatusActive ||
			client.Domain == nil || *client.Domain == "" {
			return apperror.NewUnauthorized("authentication failed")
		}

		// Find user by phone.
		user, txErr = txUserRepo.FindByPhone(req.Phone)
		if txErr != nil {
			return apperror.NewInternal("failed to look up user", txErr)
		}
		if user == nil || user.Status != shared.StatusActive {
			return apperror.NewUnauthorized("invalid phone or OTP")
		}

		// Find a valid (unused, not expired) OTP for this phone.
		otpRecord, txErr := txSmsOtpRepo.FindValid("sms", req.Phone)
		if txErr != nil {
			return apperror.NewInternal("failed to look up OTP", txErr)
		}
		if otpRecord == nil {
			return apperror.NewUnauthorized("invalid or expired OTP")
		}

		// Verify hash.
		expectedHash := crypto.HashAuthorizationCode(req.OTP)
		if subtle.ConstantTimeCompare([]byte(otpRecord.OTPHash), []byte(expectedHash)) != 1 {
			if txErr := txSmsOtpRepo.RecordFailure(otpRecord.UserOTPID, smsOTPMaxFailedAttempts); txErr != nil {
				return apperror.NewInternal("failed to record OTP attempt", txErr)
			}
			return apperror.NewUnauthorized("invalid or expired OTP")
		}

		// Mark OTP as used (single-use).
		if txErr := txSmsOtpRepo.MarkUsed(otpRecord.UserOTPID); txErr != nil {
			return apperror.NewInternal("failed to invalidate OTP", txErr)
		}

		// Resolve user identity sub for token issuance.
		userIdentity, txErr := txUserIdentityRepo.FindByUserIDAndClientID(user.UserID, client.ClientID)
		if txErr != nil || userIdentity == nil {
			return apperror.NewUnauthorized("authentication failed")
		}
		userIdentitySub = userIdentity.Sub

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "SMS OTP verification failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	// Record threat success for SMS login — marks device/last-login for future assessments.
	security.RecordLoginThreatSuccess(ctx, client.IdentityProvider.TenantID, user.UserID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), secpolicy.LoadThreatPolicy(s.securitySettingRepo, client.IdentityProvider.TenantID))
	return s.generateSMSTokenResponse(ctx, userIdentitySub, user, client)
}

func (s *smsLoginService) generateSMSTokenResponse(ctx context.Context, sub string, user *User, client *Client) (*LoginResponseDTO, error) {
	var sessionID string
	policy := resolveEffectiveSessionPolicy(s.securitySettingRepo, client)
	if s.sessionService != nil {
		if err := enforceConcurrentLimitWithPolicy(ctx, s.sessionService, user.UserUUID, user.UserID, policy); err != nil {
			return nil, err
		}
		sess, err := createSessionWithPolicy(ctx, s.sessionService, user.UserID, middleware.ClientIPFromContext(ctx), middleware.UserAgentFromContext(ctx), policy)
		if err != nil {
			return nil, err
		}
		sessionID = sess.UserTokenUUID.String()
	}
	accessToken, idToken, refreshToken, err := generateTokenSetWithAuthContext(ctx, sub, user, client, tokenAuthContextWithPolicy([]string{jwt.AMRSMS}, jwt.ACRLevel1, sessionID, policy))
	if err != nil {
		return nil, err
	}
	resp := buildLoginTokenResponse(accessToken, idToken, refreshToken, time.Now().Unix())
	applyLoginCookiePolicy(resp, policy)
	if policy.AccessTokenTTLSeconds > 0 {
		resp.ExpiresIn = int64(policy.AccessTokenTTLSeconds)
	}
	if sessionID != "" {
		resp.SessionID = &sessionID
	}
	return resp, nil
}

func userTenantID(ctx context.Context, db *gorm.DB, userID int64) int64 {
	if db == nil {
		return 0
	}
	var tenantID int64
	if err := db.WithContext(ctx).
		Table("user_identities").
		Select("tenant_id").
		Where("user_id = ?", userID).
		Order("tenant_id ASC").
		Limit(1).
		Scan(&tenantID).Error; err != nil || tenantID == 0 {
		return 0
	}
	return tenantID
}

var smsDailySendLimit = smsDailySendLimitFromDB

func smsDailySendLimitFromDB(db *gorm.DB, tenantID int64) int {
	if db == nil {
		return 1000
	}
	var limit int
	if err := db.Table("sms_config").
		Select("daily_send_limit").
		Where("tenant_id = ? AND status = 'active' AND deleted_at IS NULL", tenantID).
		Scan(&limit).Error; err != nil || limit == 0 {
		return 0
	}
	return limit
}
