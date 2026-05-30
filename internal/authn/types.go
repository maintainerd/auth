package authn

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
	Username string `json:"username"`
	Password string `json:"password"`
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
	Username string  `json:"username"`
	Fullname string  `json:"fullname"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Password string  `json:"password"`
}

// Register query parameters structure
type RegisterQueryDTO struct {
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
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

// SMSLoginSendDTO is the request to send a one-time SMS code.
type SMSLoginSendDTO struct {
	Phone      string `json:"phone"`
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}
