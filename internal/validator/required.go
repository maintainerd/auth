package validator

import (
	"errors"
	"fmt"
	"reflect"
)

/**
 * ensures the field is not nil, empty string, or empty array.
 */
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

/**
 * for string length constraints.
 */
func MinLength(min int) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return nil
				}
				v = v.Elem()
			}

			s, ok := v.Interface().(string)
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
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return nil // Optional() should have handled nil already
				}
				v = v.Elem()
			}

			s, ok := v.Interface().(string)
			if !ok {
				return errors.New("must be a string")
			}
			if len(s) > max {
				return fmt.Errorf("must be at most %d characters", max)
			}
			return nil
		},
	}
}
