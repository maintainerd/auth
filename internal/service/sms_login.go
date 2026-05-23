package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/crypto"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/jwt"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const smsOTPLength = 6
const smsOTPTTL = 10 * time.Minute

// SMSLoginService handles SMS one-time-code login flows.
type SMSLoginService interface {
	SendOTP(ctx context.Context, req dto.SMSLoginSendDTO) error
	VerifyOTP(ctx context.Context, req dto.SMSLoginVerifyDTO) (*dto.LoginResponseDTO, error)
}

type smsLoginService struct {
	db                   *gorm.DB
	userRepo             repository.UserRepository
	smsOtpRepo           repository.SMSOtpRepository
	clientRepo           repository.ClientRepository
	userIdentityRepo     repository.UserIdentityRepository
	identityProviderRepo repository.IdentityProviderRepository
	authEventService     AuthEventService
}

func NewSMSLoginService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	smsOtpRepo repository.SMSOtpRepository,
	clientRepo repository.ClientRepository,
	userIdentityRepo repository.UserIdentityRepository,
	identityProviderRepo repository.IdentityProviderRepository,
	authEventService AuthEventService,
) SMSLoginService {
	return &smsLoginService{
		db:                   db,
		userRepo:             userRepo,
		smsOtpRepo:           smsOtpRepo,
		clientRepo:           clientRepo,
		userIdentityRepo:     userIdentityRepo,
		identityProviderRepo: identityProviderRepo,
		authEventService:     authEventService,
	}
}

// SendOTP looks up the user by phone, generates a 6-digit OTP, stores its hash,
// and logs it (real SMS provider integration is a future TODO).
func (s *smsLoginService) SendOTP(ctx context.Context, req dto.SMSLoginSendDTO) error {
	_, span := otel.Tracer("service").Start(ctx, "smsLogin.sendOTP")
	defer span.End()

	// Look up user by phone — respond generically so we don't leak user existence.
	user, err := s.userRepo.FindByPhone(req.Phone)
	if err != nil {
		return apperror.NewInternal("failed to look up user", err)
	}
	if user == nil || user.Status != model.StatusActive {
		// Still return success to avoid phone enumeration.
		span.SetStatus(codes.Ok, "")
		return nil
	}

	// Generate OTP and hash it for storage.
	otp, err := crypto.GenerateOTP(smsOTPLength)
	if err != nil {
		return apperror.NewInternal("failed to generate OTP", err)
	}
	otpHash := crypto.HashAuthorizationCode(otp)

	expiresAt := time.Now().Add(smsOTPTTL)
	record := &model.SMSOtp{
		UserID:    user.UserID,
		Phone:     req.Phone,
		OTPHash:   otpHash,
		ExpiresAt: expiresAt,
	}
	if _, err := s.smsOtpRepo.Create(record); err != nil {
		return apperror.NewInternal("failed to store SMS OTP", err)
	}

	// TODO: replace with real SMS provider (Twilio/SNS/Vonage) when integrated.
	slog.Info("SMS OTP", "phone", req.Phone, "otp", otp)

	span.SetStatus(codes.Ok, "")
	return nil
}

// VerifyOTP validates the submitted OTP and issues tokens on success.
func (s *smsLoginService) VerifyOTP(ctx context.Context, req dto.SMSLoginVerifyDTO) (*dto.LoginResponseDTO, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsLogin.verifyOTP")
	defer span.End()

	var user *model.User
	var client *model.Client
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
			client.Status != model.StatusActive ||
			client.Domain == nil || *client.Domain == "" {
			return apperror.NewUnauthorized("authentication failed")
		}

		// Find user by phone.
		user, txErr = txUserRepo.FindByPhone(req.Phone)
		if txErr != nil {
			return apperror.NewInternal("failed to look up user", txErr)
		}
		if user == nil || user.Status != model.StatusActive {
			return apperror.NewUnauthorized("invalid phone or OTP")
		}

		// Find a valid (unused, not expired) OTP for this phone.
		otpRecord, txErr := txSmsOtpRepo.FindValidByPhone(req.Phone)
		if txErr != nil {
			return apperror.NewInternal("failed to look up OTP", txErr)
		}
		if otpRecord == nil {
			return apperror.NewUnauthorized("invalid or expired OTP")
		}

		// Verify hash.
		expectedHash := crypto.HashAuthorizationCode(req.OTP)
		if otpRecord.OTPHash != expectedHash {
			return apperror.NewUnauthorized("invalid or expired OTP")
		}

		// Mark OTP as used (single-use).
		if txErr := txSmsOtpRepo.MarkUsed(otpRecord.SMSOtpID); txErr != nil {
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
	return s.generateSMSTokenResponse(userIdentitySub, user, client)
}

func (s *smsLoginService) generateSMSTokenResponse(sub string, user *model.User, client *model.Client) (*dto.LoginResponseDTO, error) {
	accessToken, err := jwt.GenerateAccessToken(
		sub,
		"openid profile email",
		*client.Domain,
		*client.Identifier,
		*client.Identifier,
		client.IdentityProvider.Identifier,
	)
	if err != nil {
		return nil, err
	}

	profile := &jwt.UserProfile{
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		Phone:         user.Phone,
		PhoneVerified: user.IsPhoneVerified,
	}

	idToken, err := generateIDTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier, profile, "", nil)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshTokenFn(sub, *client.Domain, *client.Identifier, client.IdentityProvider.Identifier)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponseDTO{
		AccessToken:  accessToken,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		IssuedAt:     time.Now().Unix(),
	}, nil
}
