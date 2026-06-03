package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiPermissionModel(t *testing.T) {
	assert.Equal(t, "api_permissions", ApiPermission{}.TableName())
}
