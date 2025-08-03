package validator

import (
	"errors"
	"fmt"
	"reflect"
)

/**
 * validates whether the value is one of a predefined set of allowed values.
 */
func In(choices ...string) FieldRule {
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
			for _, c := range choices {
				if s == c {
					return nil
				}
			}
			return fmt.Errorf("must be one of: %v", choices)
		},
	}
}
