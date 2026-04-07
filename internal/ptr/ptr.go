package ptr

import "time"

// PtrOrNil returns a pointer to s, or nil if s is empty.
func PtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Ptr returns a pointer to the given string.
func Ptr(s string) *string {
	return &s
}

// TimePtr returns a pointer to the given time value.
func TimePtr(t time.Time) *time.Time {
	return &t
}
