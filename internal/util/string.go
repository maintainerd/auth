package util

func PtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Ptr returns a pointer to the given string
func Ptr(s string) *string {
	return &s
}
