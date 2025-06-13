package validator

import "fmt"

func In(set ...any) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			for _, s := range set {
				if value == s {
					return nil
				}
			}
			return fmt.Errorf("must be one of %v", set)
		},
	}
}
