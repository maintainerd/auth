package runner

import "errors"

var errAlwaysFails = errors.New("rotation failed")

// swapRotateForTest replaces the rotation seam and returns a restore func.
func swapRotateForTest(fn func() error) func() {
	prev := rotateKeys
	rotateKeys = fn
	return func() { rotateKeys = prev }
}
