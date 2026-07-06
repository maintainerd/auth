package user

import "errors"

var (
	errConsentTypeRequired   = errors.New("consent_type is required")
	errPolicyVersionRequired = errors.New("policy_version is required")
	errInvalidConsentType    = errors.New("consent_type must be one of: terms_of_service, privacy_policy, data_processing")
)
