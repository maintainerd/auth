package validator

import (
	"errors"
	"fmt"
)

func Required() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			if value == nil {
				return errors.New("is required")
			}
			switch v := value.(type) {
			case string:
				if v == "" {
					return errors.New("is required")
				}
			case []any:
				if len(v) == 0 {
					return errors.New("is required")
				}
			}
			return nil
		},
	}
}

func MinLength(min int) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			s, ok := value.(string)
			if !ok {
				return errors.New("must be a string")
			}
			if len(s) < min {
				return fmt.Errorf("must have at least %d characters", min)
			}
			return nil
		},
	}
}

func MaxLength(max int) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			s, ok := value.(string)
			if !ok {
				return errors.New("must be a string")
			}
			if len(s) > max {
				return fmt.Errorf("must have at most %d characters", max)
			}
			return nil
		},
	}
}
