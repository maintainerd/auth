package validator

/**
 * allows custom validation rules using a function. Useful for ad-hoc or inline validation logic.
 */
func By(f func(value any) error) FieldRule {
	return FieldRule{
		rule: f,
	}
}
