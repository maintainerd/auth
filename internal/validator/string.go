package validator

import (
	"errors"
	"reflect"
	"regexp"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

/**
 * checks if the value is a string.
 */
func String() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			if _, ok := value.(string); !ok {
				return errors.New("must be a string")
			}
			return nil
		},
	}
}

/**
 * validates proper email format.
 */
func Email() FieldRule {
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
			if !emailRegex.MatchString(s) {
				return errors.New("must be a valid email")
			}
			return nil
		},
	}
}

/**
 * validates the string against a regular expression.
 */
func Match(pattern *regexp.Regexp) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			s, ok := value.(string)
			if !ok {
				return errors.New("must be a string")
			}
			if !pattern.MatchString(s) {
				return errors.New("invalid format")
			}
			return nil
		},
	}
}
