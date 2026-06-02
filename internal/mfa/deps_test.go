package mfa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserTableName(t *testing.T) {
	assert.Equal(t, "users", User{}.TableName())
}
