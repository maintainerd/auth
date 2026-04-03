package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPtrOrNil(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		want    string
	}{
		{"empty string returns nil", "", true, ""},
		{"non-empty string returns pointer", "hello", false, "hello"},
		{"whitespace-only is non-empty", " ", false, " "},
		{"single char", "x", false, "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := PtrOrNil(tc.input)
			if tc.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tc.want, *got)
			}
		})
	}
}

func TestPtr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string still returns pointer", "", ""},
		{"non-empty string", "hello", "hello"},
		{"unicode string", "日本語", "日本語"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Ptr(tc.input)
			require.NotNil(t, got)
			assert.Equal(t, tc.want, *got)
		})
	}
}

func TestPtrReturnsDistinctPointers(t *testing.T) {
	s := "same"
	p1 := Ptr(s)
	p2 := Ptr(s)
	assert.NotSame(t, p1, p2, "each call should return a distinct pointer")
	assert.Equal(t, *p1, *p2)
}

