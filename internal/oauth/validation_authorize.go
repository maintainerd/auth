package oauth

import (
	"errors"
	"strconv"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
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
	r.ScreenHint = security.SanitizeInput(r.ScreenHint)
	r.RegistrationFlow = security.SanitizeInput(r.RegistrationFlow)
	r.Prompt = security.SanitizeInput(r.Prompt)
	r.CodeChallenge = security.SanitizeInput(r.CodeChallenge)
	r.CodeChallengeMethod = security.SanitizeInput(r.CodeChallengeMethod)
	r.ACRValues = security.SanitizeInput(r.ACRValues)
	r.MaxAge = security.SanitizeInput(r.MaxAge)
	r.LoginHint = security.SanitizeInput(r.LoginHint)
	r.ResponseMode = security.SanitizeInput(r.ResponseMode)
	r.UILocales = security.SanitizeInput(r.UILocales)

	// OIDC Core §6: a request object changes the request's meaning. Accepting the
	// query parameters while dropping the signed object it was supposed to
	// override would authorize something the RP did not ask for, so refuse with
	// the error code the spec defines for exactly this.
	if r.Request != "" {
		return errors.New("request_not_supported: signed request objects (JAR) are not supported; use PAR (request_uri)")
	}

	if r.ClientID == "" {
		return errors.New("client_id is required")
	}

	// Parsed rather than merely length-checked: enforcement compares it against
	// the session's auth_time, and a value that cannot be parsed must not silently
	// become "no limit".
	if r.MaxAge != "" {
		seconds, err := strconv.ParseInt(r.MaxAge, 10, 64)
		if err != nil || seconds < 0 {
			return errors.New("max_age must be a non-negative number of seconds")
		}
		r.MaxAgeSeconds = seconds
		r.MaxAgeSet = true
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
		validation.Field(&r.Prompt,
			validation.When(r.Prompt != "",
				validation.In("none").Error("only prompt=none is supported"),
			),
		),
		validation.Field(&r.ScreenHint,
			validation.When(r.ScreenHint != "",
				validation.In("signup", "login").Error("screen_hint must be 'signup' or 'login'"),
			),
		),
		validation.Field(&r.RegistrationFlow,
			validation.Length(0, 255).Error("registration_flow must not exceed 255 characters"),
		),
		validation.Field(&r.ACRValues,
			validation.Length(0, 255).Error("acr_values must not exceed 255 characters"),
		),
		validation.Field(&r.LoginHint,
			validation.Length(0, 320).Error("login_hint must not exceed 320 characters"),
		),
		validation.Field(&r.UILocales,
			validation.Length(0, 255).Error("ui_locales must not exceed 255 characters"),
		),
		validation.Field(&r.ResponseMode,
			validation.When(r.ResponseMode != "",
				// Only the query response mode is implemented. Accepting
				// fragment/form_post and then answering in query would put the code
				// somewhere the RP is not reading it — or, worse, somewhere it did not
				// intend it to be exposed.
				validation.In("query").Error("only response_mode 'query' is supported"),
			),
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
