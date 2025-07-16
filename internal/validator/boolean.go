package validator

import "errors"

func Boolean() FieldRule {
	return func(value any) error {
		if _, ok := value.(bool); !ok {
			return errors.New("must be a boolean")
		}
		return nil
	}
}
