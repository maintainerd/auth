package validator

func By(f func(value any) error) FieldRule {
	return FieldRule{
		rule: f,
	}
}
