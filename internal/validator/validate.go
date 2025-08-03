package validator

import "reflect"

/**
 * The validation entry point:
 * ValidateStruct(structPtr, fields...) — performs validation over multiple fields.
 * - It reflects over the struct to find the field name (using json tag if available).
 * - Applies rules using FieldRule.Apply().
 * - Aggregates all validation errors into ValidationErrors.
 * - xReturns nil if all validations pass; otherwise returns the error slice.
 */
func ValidateStruct(structPtr any, fields ...FieldSet) error {
	v := reflect.ValueOf(structPtr)
	if v.Kind() != reflect.Ptr {
		panic("validator: ValidateStruct requires a pointer to struct")
	}
	v = v.Elem()
	t := v.Type()

	var errs ValidationErrors

	for _, f := range fields {
		ptrVal := reflect.ValueOf(f.Ptr)
		if ptrVal.Kind() != reflect.Ptr {
			panic("validator: Field requires a pointer to struct field")
		}

		fieldName := ""
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.Addr().Pointer() == ptrVal.Pointer() {
				fieldName = t.Field(i).Tag.Get("json")
				if fieldName == "" {
					fieldName = t.Field(i).Name
				}
				break
			}
		}

		if fieldName == "" {
			panic("validator: could not find matching struct field")
		}

		// Apply rules in order
		for _, rule := range f.Rules {
			err := rule.Apply(ptrVal.Elem().Interface())
			if err != nil {
				if err == errSkipValidation {
					break // stop checking further rules for this field
				}
				errs = append(errs, ValidationErrorDetail{
					Field:   fieldName,
					Message: err.Error(),
				})
				break // stop at first actual validation error for this field
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
