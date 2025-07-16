package validator

type ValidationErrors map[string]string

func (ve ValidationErrors) Error() string {
	return "validation error"
}
