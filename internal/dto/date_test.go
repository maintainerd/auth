package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantZero  bool
		wantError bool
	}{
		{
			name:     "valid date",
			input:    `"2024-06-15"`,
			wantZero: false,
		},
		{
			name:     "null value",
			input:    `null`,
			wantZero: true,
		},
		{
			name:     "empty string",
			input:    `""`,
			wantZero: true,
		},
		{
			name:      "invalid format (wrong separator)",
			input:     `"15/06/2024"`,
			wantError: true,
		},
		{
			name:      "invalid format (datetime)",
			input:     `"2024-06-15T00:00:00Z"`,
			wantError: true,
		},
		{
			name:      "invalid date (month out of range)",
			input:     `"2024-13-01"`,
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Date
			err := json.Unmarshal([]byte(tc.input), &d)
			if tc.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.wantZero {
				assert.True(t, d.Time.IsZero())
			} else {
				assert.False(t, d.Time.IsZero())
				assert.Equal(t, 2024, d.Time.Year())
				assert.Equal(t, 6, int(d.Time.Month()))
				assert.Equal(t, 15, d.Time.Day())
			}
		})
	}
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string // JSON to unmarshal first
		wantJSON string
	}{
		{
			name:     "valid date marshals to YYYY-MM-DD",
			input:    `"1990-01-25"`,
			wantJSON: `"1990-01-25"`,
		},
		{
			name:     "zero date marshals to null",
			input:    `null`,
			wantJSON: `null`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var d Date
			require.NoError(t, json.Unmarshal([]byte(tc.input), &d))

			out, err := json.Marshal(d)
			require.NoError(t, err)
			assert.Equal(t, tc.wantJSON, string(out))
		})
	}
}

func TestDate_RoundTrip(t *testing.T) {
	type wrapper struct {
		DOB Date `json:"dob"`
	}

	original := `{"dob":"2000-12-31"}`
	var w wrapper
	require.NoError(t, json.Unmarshal([]byte(original), &w))

	out, err := json.Marshal(w)
	require.NoError(t, err)
	assert.JSONEq(t, original, string(out))
}

