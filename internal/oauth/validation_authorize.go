package oauth

import (
	"errors"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/security"
)

// Validate sanitises inputs and checks required OAuth parameters.
func (r *OAuthAuthorizeRequestDTO) Validate() error {
	r.ResponseType = security.SanitizeInput(r.ResponseType)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.RedirectURI = security.SanitizeInput(r.RedirectURI)
	r.Scope = security.SanitizeInput(r.Scope)
	r.State = security.SanitizeInput(r.State)
	r.Nonce = security.SanitizeInput(r.Nonce)
	r.IdpHint = security.SanitizeInput(r.IdpHint)
	r.CodeChallenge = security.SanitizeInput(r.CodeChallenge)
	r.CodeChallengeMethod = security.SanitizeInput(r.CodeChallengeMethod)

	if r.ClientID == "" {
		return errors.New("client_id is required")
	}

	return validation.ValidateStruct(r,
		validation.Field(&r.ResponseType,
			validation.Required.Error("response_type is required"),
			validation.In("code").Error("response_type must be 'code'"),
		),
		validation.Field(&r.ClientID,
			validation.Length(0, 255).Error("client_id must not exceed 255 characters"),
		),
		validation.Field(&r.RedirectURI,
			validation.Required.Error("redirect_uri is required"),
			validation.Length(1, 2048).Error("redirect_uri must not exceed 2048 characters"),
		),
		validation.Field(&r.CodeChallenge,
			validation.When(r.CodeChallenge != "",
				validation.Length(43, 128).Error("code_challenge must be between 43 and 128 characters"),
			),
		),
		validation.Field(&r.CodeChallengeMethod,
			validation.When(r.CodeChallengeMethod != "",
				validation.In("S256").Error("code_challenge_method must be 'S256'"),
			),
		),
		validation.Field(&r.State,
			validation.Length(0, 512).Error("state must not exceed 512 characters"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
		validation.Field(&r.Nonce,
			validation.Length(0, 512).Error("nonce must not exceed 512 characters"),
		),
		validation.Field(&r.IdpHint,
			validation.Length(0, 255).Error("idp_hint must not exceed 255 characters"),
		),
	)
}

// Validate sanitises inputs and checks that the challenge ID is a valid UUID.
func (r *OAuthConsentDecisionDTO) Validate() error {
	r.ChallengeID = security.SanitizeInput(r.ChallengeID)

	return validation.ValidateStruct(r,
		validation.Field(&r.ChallengeID,
			validation.Required.Error("challenge_id is required"),
			validation.By(func(value any) error {
				s, _ := value.(string)
				if _, err := uuid.Parse(s); err != nil {
					return validation.NewError("validation_uuid", "challenge_id must be a valid UUID")
				}
				return nil
			}),
		),
	)
}
