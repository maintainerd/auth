package validator

import "errors"

// Sentinel error used to short-circuit remaining rules when Optional() applies
var errSkipValidation = errors.New("__skip_validation__")

/**
 * wraps a rule function and an optional custom error message.
 */
type FieldRule struct {
	rule    func(value any) error
	message string
}

/**
 * applies the rule and returns the error (custom if set).
 */
func (fr FieldRule) Apply(value any) error {
	err := fr.rule(value)
	if err != nil {
		if err == errSkipValidation {
			return errSkipValidation
		}
		if fr.message != "" {
			return ValidationError(fr.message)
		}
		return err
	}
	return nil
}

/**
 * allows chaining a custom error message.
 */
func (fr FieldRule) Error(msg string) FieldRule {
	fr.message = msg
	return fr
}

/**
 * a wrapper type to represent a custom message explicitly.
 */
type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}
