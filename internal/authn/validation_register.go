package authn

import (
	"errors"
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/platform/security"
	"github.com/maintainerd/auth/internal/platform/signedurl"
	"github.com/maintainerd/auth/internal/platform/valid"
)

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
			validation.Length(0, 255).Error("Fullname must not exceed 255 characters"),
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

func (q *RegisterQueryDTO) Validate() error {
	// Sanitize inputs first
	q.ClientID = security.SanitizeInput(q.ClientID)
	q.TenantID = security.SanitizeInput(q.TenantID)

	return validation.ValidateStruct(q,
		validation.Field(&q.ClientID,
			validation.Length(0, 255).Error("Client ID must not exceed 255 characters"),
		),
		validation.Field(&q.TenantID,
			validation.Length(0, 255).Error("Tenant ID must not exceed 255 characters"),
		),
	)
}

func (q *RegisterInviteQueryDTO) Validate() error {
	// Sanitize inputs first
	q.ClientID = security.SanitizeInput(q.ClientID)
	q.TenantID = security.SanitizeInput(q.TenantID)
	q.InviteToken = security.SanitizeInput(q.InviteToken)
	q.Expires = security.SanitizeInput(q.Expires)
	q.Sig = security.SanitizeInput(q.Sig)
	q.AuthFlow = security.SanitizeInput(q.AuthFlow)

	return validation.ValidateStruct(q,
		validation.Field(&q.ClientID,
			validation.Length(0, 255).Error("Client ID must not exceed 255 characters"),
		),
		validation.Field(&q.TenantID,
			validation.Length(0, 255).Error("Tenant ID must not exceed 255 characters"),
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
		validation.Field(&q.AuthFlow,
			validation.Length(0, 255).Error("Auth flow identifier must not exceed 255 characters"),
		),
	)
}

// ValidateInternal validates the query DTO for internal use (client_id/tenant_id optional).
func (q *RegisterInviteQueryDTO) ValidateInternal() error {
	q.InviteToken = security.SanitizeInput(q.InviteToken)
	q.Expires = security.SanitizeInput(q.Expires)
	q.Sig = security.SanitizeInput(q.Sig)
	q.AuthFlow = security.SanitizeInput(q.AuthFlow)
	q.ClientID = security.SanitizeInput(q.ClientID)
	q.TenantID = security.SanitizeInput(q.TenantID)

	return validation.ValidateStruct(q,
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
		validation.Field(&q.AuthFlow,
			validation.Length(0, 255).Error("Auth flow identifier must not exceed 255 characters"),
		),
		validation.Field(&q.ClientID,
			validation.Length(0, 255).Error("Client ID must not exceed 255 characters"),
		),
		validation.Field(&q.TenantID,
			validation.Length(0, 255).Error("Tenant ID must not exceed 255 characters"),
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
