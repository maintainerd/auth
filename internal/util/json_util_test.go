package util

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name  string
		input any
		check func(t *testing.T, got string)
	}{
		{
			name:  "nil input",
			input: nil,
			check: func(t *testing.T, got string) { assert.Equal(t, "null", got) },
		},
		{
			name:  "string value",
			input: "hello",
			check: func(t *testing.T, got string) { assert.Equal(t, `"hello"`, got) },
		},
		{
			name:  "integer value",
			input: 42,
			check: func(t *testing.T, got string) { assert.Equal(t, "42", got) },
		},
		{
			name:  "boolean true",
			input: true,
			check: func(t *testing.T, got string) { assert.Equal(t, "true", got) },
		},
		{
			name:  "empty map",
			input: map[string]any{},
			check: func(t *testing.T, got string) { assert.Equal(t, "{}", got) },
		},
		{
			name:  "map with values",
			input: map[string]any{"key": "value", "num": 1},
			check: func(t *testing.T, got string) {
				var out map[string]any
				require.NoError(t, json.Unmarshal([]byte(got), &out))
				assert.Equal(t, "value", out["key"])
				assert.InDelta(t, float64(1), out["num"], 0)
			},
		},
		{
			name:  "struct",
			input: struct{ Name string }{Name: "Alice"},
			check: func(t *testing.T, got string) {
				assert.JSONEq(t, `{"Name":"Alice"}`, got)
			},
		},
		{
			name:  "slice of strings",
			input: []string{"a", "b", "c"},
			check: func(t *testing.T, got string) {
				assert.JSONEq(t, `["a","b","c"]`, got)
			},
		},
		{
			name:  "returns valid JSON string (not empty)",
			input: struct{ X int }{X: 99},
			check: func(t *testing.T, got string) {
				assert.NotEmpty(t, got)
				assert.True(t, json.Valid([]byte(got)))
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ToJSON(tc.input)
			tc.check(t, got)
		})
	}
}

