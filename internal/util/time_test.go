package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimePtr(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
	}{
		{"zero time", time.Time{}},
		{"now", time.Now()},
		{"specific UTC time", time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)},
		{"far future", time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)},
		{"far past", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TimePtr(tc.input)
			require.NotNil(t, got)
			assert.True(t, tc.input.Equal(*got), "pointed-to value should equal input")
		})
	}
}

func TestTimePtrReturnsDistinctPointers(t *testing.T) {
	now := time.Now()
	p1 := TimePtr(now)
	p2 := TimePtr(now)
	assert.NotSame(t, p1, p2, "each call should return a distinct pointer")
	assert.True(t, p1.Equal(*p2))
}

func TestTimePtrMutationDoesNotAffectOriginal(t *testing.T) {
	original := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	ptr := TimePtr(original)
	*ptr = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	// original variable is a value type – it must remain unchanged
	assert.Equal(t, 2024, original.Year(), "original should not be mutated via pointer")
}

