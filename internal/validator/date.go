package validator

import (
	"errors"
	"time"
)

func Date(layout string) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			s, ok := value.(string)
			if !ok {
				return errors.New("must be a date string")
			}
			if _, err := time.Parse(layout, s); err != nil {
				return errors.New("invalid date format")
			}
			return nil
		},
	}
}
