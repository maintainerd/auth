package validator

type FieldRule func(value any) error

type FieldSet struct {
	Value any
	Rules []FieldRule
}

func Field(value any, rules ...FieldRule) FieldSet {
	return FieldSet{
		Value: value,
		Rules: rules,
	}
}
