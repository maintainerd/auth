package validator

import (
	"errors"

	"github.com/google/uuid"
)

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
