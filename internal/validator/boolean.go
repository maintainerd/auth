package validator

import "errors"

func Boolean() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			if _, ok := value.(bool); !ok {
				return errors.New("must be a boolean")
			}
			return nil
		},
	}
}
