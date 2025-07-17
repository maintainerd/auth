package validator

type FieldRule struct {
	rule    func(value any) error
	message string
}

func (fr FieldRule) Apply(value any) error {
	if err := fr.rule(value); err != nil {
		if fr.message != "" {
			return ValidationError(fr.message)
		}
		return err
	}
	return nil
}

func (fr FieldRule) Error(msg string) FieldRule {
	fr.message = msg
	return fr
}

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}
