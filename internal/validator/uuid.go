package validator

import (
	"errors"

	"github.com/google/uuid"
)

/**
 * validates if a string is a valid UUID using the google/uuid package.
 */
func UUID() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			s, ok := value.(string)
			if !ok {
				return errors.New("must be a string UUID")
			}
			if _, err := uuid.Parse(s); err != nil {
				return errors.New("invalid UUID")
			}
			return nil
		},
	}
}
