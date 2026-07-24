package authn

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// registrationContextFlowPattern mirrors idp.registrationFlowNamePattern. A
// selector that could never match a stored name is rejected before it reaches
// the database, so probing with junk is cheap to spot and cheap to refuse.
var registrationContextFlowPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9]*([-_][a-z0-9]+)*$`)

func (q RegistrationContextQueryDTO) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.RegistrationFlow,
			validation.When(q.RegistrationFlow != "",
				validation.Length(1, 100).Error("registration_flow must be between 1 and 100 characters"),
				validation.Match(registrationContextFlowPattern).Error("registration_flow must be a valid flow name"),
			),
		),
	)
}
