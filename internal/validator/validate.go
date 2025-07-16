package validator

import "fmt"

func ValidateStruct(fields ...FieldSet) error {
	errors := ValidationErrors{}
	for idx, f := range fields {
		for _, rule := range f.Rules {
			if err := rule(f.Value); err != nil {
				errors[fmt.Sprintf("field_%d", idx)] = err.Error()
				break
			}
		}
	}
	if len(errors) > 0 {
		return errors
	}
	return nil
}
