package authn

import (
	"errors"
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"github.com/maintainerd/auth/internal/platform/valid"
)

// SendEmailVerificationRequestDTO represents the request payload to (re)send an email verification code.
type SendEmailVerificationRequestDTO struct {
	Email string `json:"email"`
}

func (r *SendEmailVerificationRequestDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
	)
}

// SendEmailVerificationResponseDTO represents the response for a send-verification request.
type SendEmailVerificationResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// VerifyEmailRequestDTO represents the request payload to consume a verification code.
type VerifyEmailRequestDTO struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (r *VerifyEmailRequestDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)
	r.OTP = security.SanitizeInput(r.OTP)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
		validation.Field(&r.OTP,
			validation.Required.Error("Verification code is required"),
			validation.Length(4, 12).Error("Verification code must be between 4 and 12 characters"),
		),
	)
}

// VerifyEmailResponseDTO represents the response after a successful verification.
type VerifyEmailResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// ForgotPasswordRequestDTO represents the request payload for forgot password
type ForgotPasswordRequestDTO struct {
	Email string `json:"email"`
}

func (r *ForgotPasswordRequestDTO) Validate() error {
	// Sanitize inputs first
	r.Email = security.SanitizeInput(r.Email)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
	)
}

// ForgotPasswordResponseDTO represents the response for forgot password request
type ForgotPasswordResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// Login request payload structure
type LoginRequestDTO struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *LoginRequestDTO) Validate() error {
	// Sanitize inputs first
	r.Username = security.SanitizeInput(r.Username)
	r.Password = security.SanitizeInput(r.Password)

	return validation.ValidateStruct(r,
		validation.Field(&r.Username,
			validation.Required.Error("Username is required"),
			validation.Length(1, 255).Error("Username must not exceed 255 characters"),
		),
		validation.Field(&r.Password,
			validation.Required.Error("Password is required"),
			validation.Length(1, 128).Error("Password must not exceed 128 characters"),
		),
	)
}

// Login query parameters structure
type LoginQueryDTO struct {
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (q *LoginQueryDTO) Validate() error {
	// Sanitize inputs first
	q.ClientID = security.SanitizeInput(q.ClientID)
	q.ProviderID = security.SanitizeInput(q.ProviderID)

	return validation.ValidateStruct(q,
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 255).Error("Client ID must not exceed 255 characters"),
		),
		validation.Field(&q.ProviderID,
			validation.Required.Error("Provider ID is required"),
			validation.Length(1, 255).Error("Provider ID must not exceed 255 characters"),
		),
	)
}

// ValidateSignedURL validates signed URL parameters for login
func (q *LoginQueryDTO) ValidateSignedURL(values url.Values) error {
	// Extract and validate signed URL parameters
	if _, err := signedurl.ValidateSignedURL(values); err != nil {
		return err
	}
	return nil
}

// LoginResponseDTO is the response structure for login operations
type LoginResponseDTO struct {
	AccessToken           string  `json:"access_token"`
	IDToken               string  `json:"id_token"`
	RefreshToken          string  `json:"refresh_token,omitempty"`
	ExpiresIn             int64   `json:"expires_in"`
	TokenType             string  `json:"token_type"`
	IssuedAt              int64   `json:"issued_at"`
	RequirePasswordChange bool    `json:"require_password_change,omitempty"`
	SessionID             *string `json:"session_id,omitempty"`
}

// SendMagicLinkRequestDTO represents the request payload to send a passwordless
// magic-link login email.
type SendMagicLinkRequestDTO struct {
	Email string `json:"email"`
}

func (r *SendMagicLinkRequestDTO) Validate() error {
	r.Email = security.SanitizeInput(r.Email)

	return validation.ValidateStruct(r,
		validation.Field(&r.Email,
			validation.Required.Error("Email is required"),
			is.Email.Error("Email must be a valid email address"),
			validation.Length(1, 255).Error("Email must not exceed 255 characters"),
		),
	)
}

// SendMagicLinkResponseDTO represents the response for a send-magic-link request.
type SendMagicLinkResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// VerifyMagicLinkRequestDTO represents the request payload to consume a magic-link
// token and exchange it for a session.
type VerifyMagicLinkRequestDTO struct {
	Token string `json:"token"`
}

func (r *VerifyMagicLinkRequestDTO) Validate() error {
	r.Token = security.SanitizeInput(r.Token)

	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required.Error("Token is required"),
			validation.Length(16, 256).Error("Token has an invalid length"),
		),
	)
}

// Register request payload structure
type RegisterRequestDTO struct {
	Username string  `json:"username"`
	Fullname string  `json:"fullname"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Password string  `json:"password"`
}

func (r *RegisterRequestDTO) Validate() error {
	// Sanitize inputs first
	r.Username = security.SanitizeInput(r.Username)
	r.Fullname = security.SanitizeInput(r.Fullname)
	r.Password = security.SanitizeInput(r.Password)
	if r.Email != nil {
		*r.Email = security.SanitizeInput(*r.Email)
	}
	if r.Phone != nil {
		*r.Phone = security.SanitizeInput(*r.Phone)
	}

	return validation.ValidateStruct(r,
		validation.Field(&r.Username,
			validation.Required.Error("Username is required"),
			validation.Length(1, 255).Error("Username must not exceed 255 characters"),
		),
		validation.Field(&r.Fullname,
			validation.Required.Error("Fullname is required"),
			validation.Length(1, 255).Error("Fullname must not exceed 255 characters"),
		),
		validation.Field(&r.Email,
			validation.When(r.Email != nil,
				validation.By(func(value interface{}) error {
					if email := value.(*string); email != nil && *email != "" {
						if !valid.IsValidEmail(*email) {
							return errors.New("email must be a valid email address")
						}
					}
					return nil
				}),
			),
		),
		validation.Field(&r.Phone,
			validation.When(r.Phone != nil,
				validation.By(func(value interface{}) error {
					if phone := value.(*string); phone != nil && *phone != "" {
						if !valid.IsValidPhoneNumber(*phone) {
							return errors.New("phone must be a valid phone number")
						}
					}
					return nil
				}),
			),
		),
		validation.Field(&r.Password,
			validation.Required.Error("Password is required"),
			validation.Length(8, 128).Error("Password must be between 8 and 128 characters"),
		),
	)
}

// ValidateForRegistration validates with additional password strength requirements
func (r *RegisterRequestDTO) ValidateForRegistration() error {
	// First do standard validation (includes sanitization)
	if err := r.Validate(); err != nil {
		return err
	}

	// Additional password strength validation for registration
	if err := security.ValidatePasswordStrength(r.Password); err != nil {
		return err
	}

	return nil
}

// Register query parameters structure
type RegisterQueryDTO struct {
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (q *RegisterQueryDTO) Validate() error {
	// Sanitize inputs first
	q.ClientID = security.SanitizeInput(q.ClientID)
	q.ProviderID = security.SanitizeInput(q.ProviderID)

	return validation.ValidateStruct(q,
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 255).Error("Client ID must not exceed 255 characters"),
		),
		validation.Field(&q.ProviderID,
			validation.Required.Error("Provider ID is required"),
			validation.Length(1, 255).Error("Provider ID must not exceed 255 characters"),
		),
	)
}

// Register invite query parameters structure
type RegisterInviteQueryDTO struct {
	ClientID    string `json:"client_id"`
	ProviderID  string `json:"provider_id"`
	InviteToken string `json:"invite_token"`
	Expires     string `json:"expires"`
	Sig         string `json:"sig"`
}

func (q *RegisterInviteQueryDTO) Validate() error {
	// Sanitize inputs first
	q.ClientID = security.SanitizeInput(q.ClientID)
	q.ProviderID = security.SanitizeInput(q.ProviderID)
	q.InviteToken = security.SanitizeInput(q.InviteToken)
	q.Expires = security.SanitizeInput(q.Expires)
	q.Sig = security.SanitizeInput(q.Sig)

	return validation.ValidateStruct(q,
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 255).Error("Client ID must not exceed 255 characters"),
		),
		validation.Field(&q.ProviderID,
			validation.Required.Error("Provider ID is required"),
			validation.Length(1, 255).Error("Provider ID must not exceed 255 characters"),
		),
		validation.Field(&q.InviteToken,
			validation.Required.Error("Invite token is required"),
			validation.Length(1, 500).Error("Invite token must not exceed 500 characters"),
		),
		validation.Field(&q.Expires,
			validation.Required.Error("Expires parameter is required"),
			validation.Length(1, 50).Error("Expires parameter must not exceed 50 characters"),
		),
		validation.Field(&q.Sig,
			validation.Required.Error("Signature is required"),
			validation.Length(1, 500).Error("Signature must not exceed 500 characters"),
		),
	)
}

// ValidateSignedURL validates signed URL parameters for register invite
func (q *RegisterInviteQueryDTO) ValidateSignedURL(values url.Values) error {
	// Extract and validate signed URL parameters
	if _, err := signedurl.ValidateSignedURL(values); err != nil {
		return err
	}
	return nil
}

// RegisterResponseDTO is the response structure for registration operations
type RegisterResponseDTO struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IssuedAt     int64  `json:"issued_at"`
}

// ResetPasswordRequestDTO represents the request to reset a password
// Token is always extracted from the signed URL, not from request body
type ResetPasswordRequestDTO struct {
	NewPassword string `json:"new_password" example:"NewSecurePassword123!"`
}

// Validate validates the reset password request
func (dto ResetPasswordRequestDTO) Validate() error {
	return validation.ValidateStruct(&dto,
		validation.Field(&dto.NewPassword, validation.Required.Error("New password is required")),
		// Token is optional in request body - can come from signed URL instead
	)
}

// ResetPasswordResponseDTO represents the response after password reset
type ResetPasswordResponseDTO struct {
	Message string `json:"message" example:"Password has been reset successfully"`
	Success bool   `json:"success" example:"true"`
}

// ResetPasswordQueryDTO represents query parameters for signed URL validation
type ResetPasswordQueryDTO struct {
	Token      string `json:"token"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
	Expires    string `json:"expires"`
	Sig        string `json:"sig"`
}

// Validate validates the reset password query parameters
func (q ResetPasswordQueryDTO) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.Token,
			validation.Required.Error("Token is required"),
			validation.Length(1, 500).Error("Token must not exceed 500 characters"),
		),
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 100).Error("Client ID must not exceed 100 characters"),
		),
		validation.Field(&q.ProviderID,
			validation.Required.Error("Provider ID is required"),
			validation.Length(1, 100).Error("Provider ID must not exceed 100 characters"),
		),
		validation.Field(&q.Expires,
			validation.Required.Error("Expires is required"),
			validation.Length(1, 50).Error("Expires must not exceed 50 characters"),
		),
		validation.Field(&q.Sig,
			validation.Required.Error("Signature is required"),
			validation.Length(1, 500).Error("Signature must not exceed 500 characters"),
		),
	)
}

// ValidateSignedURL validates signed URL parameters for reset password
func (q *ResetPasswordQueryDTO) ValidateSignedURL(values url.Values) error {
	// Extract and validate signed URL parameters
	if _, err := signedurl.ValidateSignedURL(values); err != nil {
		return err
	}
	return nil
}

// SMSLoginSendDTO is the request to send a one-time SMS code.
type SMSLoginSendDTO struct {
	Phone      string `json:"phone"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (r *SMSLoginSendDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.Length(1, 20)),
		validation.Field(&r.ClientID, validation.Required),
		validation.Field(&r.ProviderID, validation.Required),
	)
}

// SMSLoginVerifyDTO is the request to verify an SMS OTP and obtain tokens.
type SMSLoginVerifyDTO struct {
	Phone      string `json:"phone"`
	OTP        string `json:"otp"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (r *SMSLoginVerifyDTO) Validate() error {
	r.Phone = security.SanitizeInput(r.Phone)
	return validation.ValidateStruct(r,
		validation.Field(&r.Phone, validation.Required, validation.Length(1, 20)),
		validation.Field(&r.OTP, validation.Required, validation.Length(6, 6)),
		validation.Field(&r.ClientID, validation.Required),
		validation.Field(&r.ProviderID, validation.Required),
	)
}
