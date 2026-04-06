package util

import "time"

// TimePtr returns a pointer to the given time value.
func TimePtr(t time.Time) *time.Time {
	return &t
}
