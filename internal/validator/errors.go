package validator

type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []ValidationErrorDetail

func (ve ValidationErrors) Error() string {
	return "validation error"
}
