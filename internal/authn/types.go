package authn

import "encoding/json"

// MFALoginVerifyRequestDTO completes the login MFA second step. `code` carries
// the typed proof (totp/sms/backup_code); `assertion` carries the raw WebAuthn
// assertion JSON (passkey). The challenge token comes from the login response.
type MFALoginVerifyRequestDTO struct {
	ChallengeToken string          `json:"mfa_challenge_token"`
	Method         string          `json:"method"`
	Code           string          `json:"code,omitempty"`
	Assertion      json.RawMessage `json:"assertion,omitempty"`
	RememberDevice bool            `json:"remember_device,omitempty"`
}

// MFALoginChallengeRequestDTO carries just the login MFA challenge token, used
// by the send-SMS and WebAuthn-begin steps.
type MFALoginChallengeRequestDTO struct {
	ChallengeToken string `json:"mfa_challenge_token"`
}

// SendEmailVerificationRequestDTO represents the request payload to (re)send an email verification code.
type SendEmailVerificationRequestDTO struct {
	Email string `json:"email"`
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

// VerifyEmailResponseDTO represents the response after a successful verification.
type VerifyEmailResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// ForgotPasswordRequestDTO represents the request payload for forgot password
type ForgotPasswordRequestDTO struct {
	Email string `json:"email"`
}

// ForgotPasswordResponseDTO represents the response for forgot password request
type ForgotPasswordResponseDTO struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// Login request payload structure
type LoginRequestDTO struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	TrustedDeviceToken string `json:"trusted_device_token,omitempty"`
}

// LoginQueryDTO holds login query parameters.
type LoginQueryDTO struct {
	ClientID string `json:"client_id"`
	TenantID string `json:"tenant_id"`
}

// RefreshTokenRequestDTO is the request body for the token refresh endpoint.
// The refresh token may instead be supplied via the refresh-token cookie, in
// which case the body is optional.
type RefreshTokenRequestDTO struct {
	RefreshToken string `json:"refresh_token"`
}

// LoginResponseDTO is the response structure for login operations
type LoginResponseDTO struct {
	AccessToken             string   `json:"access_token,omitempty"`
	IDToken                 string   `json:"id_token,omitempty"`
	RefreshToken            string   `json:"refresh_token,omitempty"`
	ExpiresIn               int64    `json:"expires_in,omitempty"`
	TokenType               string   `json:"token_type,omitempty"`
	IssuedAt                int64    `json:"issued_at,omitempty"`
	RequirePasswordChange   bool     `json:"require_password_change,omitempty"`
	SessionID               *string  `json:"session_id,omitempty"`
	MFARequired             bool     `json:"mfa_required,omitempty"`
	MFAChallengeToken       *string  `json:"mfa_challenge_token,omitempty"`
	MFAAllowedMethods       []string `json:"mfa_allowed_methods,omitempty"`
	TrustedDeviceToken      string   `json:"trusted_device_token,omitempty"`
	TrustedDeviceMaxAge     int      `json:"-"`
	CookieSecure            *bool    `json:"-"`
	CookieHTTPOnly          *bool    `json:"-"`
	CookieSameSite          string   `json:"-"`
	RefreshTokenMaxAge      int      `json:"-"`
	AccessTokenCookieMaxAge int64    `json:"-"`
}

// SendMagicLinkRequestDTO represents the request payload to send a passwordless
// magic-link login email.
type SendMagicLinkRequestDTO struct {
	Email string `json:"email"`
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

// Register request payload structure
type RegisterRequestDTO struct {
	Username     string  `json:"username"`
	Fullname     string  `json:"fullname"`
	Email        *string `json:"email,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	Password     string  `json:"password"`
	CaptchaToken string  `json:"captcha_token,omitempty"`
}

// Register query parameters structure
type RegisterQueryDTO struct {
	ClientID         string `json:"client_id"`
	TenantID         string `json:"tenant_id"`
	RegistrationFlow string `json:"registration_flow"`
}

// RegisterInviteQueryDTO holds signed invite registration query parameters.
type RegisterInviteQueryDTO struct {
	ClientID    string `json:"client_id"`
	TenantID    string `json:"tenant_id"`
	InviteToken string `json:"invite_token"`
	Expires     string `json:"expires"`
	Sig         string `json:"sig"`
}

// RegisterResponseDTO is the response structure for registration operations
type RegisterResponseDTO struct {
	AccessToken             string `json:"access_token"`
	IDToken                 string `json:"id_token"`
	RefreshToken            string `json:"refresh_token,omitempty"`
	ExpiresIn               int64  `json:"expires_in"`
	TokenType               string `json:"token_type"`
	IssuedAt                int64  `json:"issued_at"`
	CookieSecure            *bool  `json:"-"`
	CookieHTTPOnly          *bool  `json:"-"`
	CookieSameSite          string `json:"-"`
	RefreshTokenMaxAge      int    `json:"-"`
	AccessTokenCookieMaxAge int64  `json:"-"`
}

// ResetPasswordRequestDTO represents the request to reset a password
// Token is always extracted from the signed URL, not from request body
type ResetPasswordRequestDTO struct {
	NewPassword string `json:"new_password" example:"NewSecurePassword123!"`
}

// ResetPasswordResponseDTO represents the response after password reset.
type ResetPasswordResponseDTO struct {
	Message string `json:"message" example:"Password has been reset successfully"`
	Success bool   `json:"success" example:"true"`
}

// ResetPasswordQueryDTO represents query parameters for signed URL validation.
type ResetPasswordQueryDTO struct {
	Token      string `json:"token"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
	Expires    string `json:"expires"`
	Sig        string `json:"sig"`
}

// SMSLoginSendDTO is the request to send a one-time SMS code.
// client_id/tenant_id are passed as query parameters, not in the body.
type SMSLoginSendDTO struct {
	Phone string `json:"phone"`
}

// SMSLoginVerifyDTO is the request to verify an SMS OTP and obtain tokens.
// client_id/tenant_id are passed as query parameters, not in the body.
type SMSLoginVerifyDTO struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}
