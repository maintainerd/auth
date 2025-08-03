package validator

/**
 * skips validation if the field is nil, empty string, or empty array. Acts as a bypass rule.
 */
func Optional() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			if value == nil {
				return errSkipValidation
			}
			switch v := value.(type) {
			case string:
				if v == "" {
					return errSkipValidation
				}
			case []any:
				if len(v) == 0 {
					return errSkipValidation
				}
			}
			return nil
		},
	}
}
