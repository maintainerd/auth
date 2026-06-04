package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserPasswordHistoryTableName(t *testing.T) {
	assert.Equal(t, "user_password_history", UserPasswordHistory{}.TableName())
}
