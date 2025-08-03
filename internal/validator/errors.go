package validator

/**
 * includes the field name and error message.
 */
type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

/**
 * a slice of those details, used as the main return type when validation fails.
 */
type ValidationErrors []ValidationErrorDetail

func (ve ValidationErrors) Error() string {
	return "validation error"
}
