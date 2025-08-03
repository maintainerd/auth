package validator

import (
	"errors"
	"reflect"
	"time"
)

/**
 * validates that a string can be parsed as a date using the provided layout.
 */
func Date(layout string) FieldRule {
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
			_, err := time.Parse(layout, s)
			if err != nil {
				return errors.New("must match date format " + layout)
			}
			return nil
		},
	}
}
