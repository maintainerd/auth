package validator

import "reflect"

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

		// Apply rules
		for _, rule := range f.Rules {
			if err := rule.Apply(ptrVal.Elem().Interface()); err != nil {
				errs = append(errs, ValidationErrorDetail{
					Field:   fieldName,
					Message: err.Error(),
				})
				break
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
