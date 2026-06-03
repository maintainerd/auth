package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_TableName(t *testing.T) {
	assert.Equal(t, "users", User{}.TableName())
}

func TestUserIdentity_TableName(t *testing.T) {
	assert.Equal(t, "user_identities", UserIdentity{}.TableName())
}
