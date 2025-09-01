package util

func PtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
