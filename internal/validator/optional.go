package validator

func Optional() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			if value == nil {
				return nil
			}
			switch v := value.(type) {
			case string:
				if v == "" {
					return nil
				}
			case []any:
				if len(v) == 0 {
					return nil
				}
			}
			return nil
		},
	}
}
