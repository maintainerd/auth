package validator

/**
 * associates a pointer to a field with a list of validation rules.
 */
type FieldSet struct {
	Ptr   any
	Rules []FieldRule
}

/**
 * constructor for creating a FieldSet.
 */
func Field(ptr any, rules ...FieldRule) FieldSet {
	return FieldSet{
		Ptr:   ptr,
		Rules: rules,
	}
}
