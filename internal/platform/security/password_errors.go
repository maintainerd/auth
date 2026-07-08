package security

import "errors"

// PasswordStrengthError wraps a password-policy violation so callers can
// detect strength failures via errors.As without matching message strings.
type PasswordStrengthError struct {
	Err error
}

func (e *PasswordStrengthError) Error() string { return e.Err.Error() }
func (e *PasswordStrengthError) Unwrap() error { return e.Err }

// IsPasswordStrengthError reports whether err originated from ValidatePasswordStrength.
func IsPasswordStrengthError(err error) bool {
	var pse *PasswordStrengthError
	return errors.As(err, &pse)
}
