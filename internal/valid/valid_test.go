package valid_test

import (
	"testing"

	"github.com/maintainerd/auth/internal/valid"
	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid simple", "user@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"empty", "", false},
		{"missing at", "userexample.com", false},
		{"missing domain", "user@", false},
		{"missing tld", "user@example", false},
		{"too long", string(make([]byte, 255)), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, valid.IsValidEmail(tc.email))
		})
	}
}

func TestIsValidPhoneNumber(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"valid international", "+12345678901", true},
		{"valid local", "1234567", true},
		{"too short", "123", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, valid.IsValidPhoneNumber(tc.phone))
		})
	}
}
