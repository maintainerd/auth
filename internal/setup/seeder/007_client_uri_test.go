package seeder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizedHTTPSBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "keeps https URL",
			input:    "https://identity.auth.maintainerd.local",
			expected: "https://identity.auth.maintainerd.local",
		},
		{
			name:     "keeps http URL",
			input:    "http://localhost:5174/",
			expected: "http://localhost:5174",
		},
		{
			name:     "adds https to bare host",
			input:    "identity.auth.maintainerd.local",
			expected: "https://identity.auth.maintainerd.local",
		},
		{
			name:     "trims whitespace and trailing slash",
			input:    " https://identity.auth.maintainerd.local/ ",
			expected: "https://identity.auth.maintainerd.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizedHTTPSBaseURL(tt.input))
		})
	}
}
