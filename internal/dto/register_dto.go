package dto

import (
	"errors"
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/util"
)

// Register request payload structure
type RegisterRequestDto struct {
	Username string  `json:"username"`
	Fullname string  `json:"fullname"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Password string  `json:"password"`
}

func (r *RegisterRequestDto) Validate() error {
	// Sanitize inputs first
	r.Username = util.SanitizeInput(r.Username)
	r.Fullname = util.SanitizeInput(r.Fullname)
	r.Password = util.SanitizeInput(r.Password)
	if r.Email != nil {
		*r.Email = util.SanitizeInput(*r.Email)
	}
	if r.Phone != nil {
		*r.Phone = util.SanitizeInput(*r.Phone)
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
						if !util.IsValidEmail(*email) {
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
						if !util.IsValidPhoneNumber(*phone) {
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
func (r *RegisterRequestDto) ValidateForRegistration() error {
	// First do standard validation (includes sanitization)
	if err := r.Validate(); err != nil {
		return err
	}

	// Additional password strength validation for registration
	if err := util.ValidatePasswordStrength(r.Password); err != nil {
		return err
	}

	return nil
}

// Register query parameters structure
type RegisterQueryDto struct {
	ClientID   string `json:"client_id"`
	ProviderID string `json:"provider_id"`
}

func (q *RegisterQueryDto) Validate() error {
	// Sanitize inputs first
	q.ClientID = util.SanitizeInput(q.ClientID)
	q.ProviderID = util.SanitizeInput(q.ProviderID)

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
type RegisterInviteQueryDto struct {
	ClientID    string `json:"client_id"`
	ProviderID  string `json:"provider_id"`
	InviteToken string `json:"invite_token"`
	Expires     string `json:"expires"`
	Sig         string `json:"sig"`
}

func (q *RegisterInviteQueryDto) Validate() error {
	// Sanitize inputs first
	q.ClientID = util.SanitizeInput(q.ClientID)
	q.ProviderID = util.SanitizeInput(q.ProviderID)
	q.InviteToken = util.SanitizeInput(q.InviteToken)
	q.Expires = util.SanitizeInput(q.Expires)
	q.Sig = util.SanitizeInput(q.Sig)

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
func (q *RegisterInviteQueryDto) ValidateSignedURL(values url.Values) error {
	// Extract and validate signed URL parameters
	if _, err := util.ValidateSignedURL(values); err != nil {
		return err
	}
	return nil
}

// RegisterResponseDto is the response structure for registration operations
type RegisterResponseDto struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IssuedAt     int64  `json:"issued_at"`
}
