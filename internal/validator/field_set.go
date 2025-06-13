package validator

type FieldSet struct {
	Ptr   any
	Rules []FieldRule
}

func Field(ptr any, rules ...FieldRule) FieldSet {
	return FieldSet{
		Ptr:   ptr,
		Rules: rules,
	}
}
