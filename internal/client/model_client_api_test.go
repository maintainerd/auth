package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientAPI_TableName(t *testing.T) {
	assert.Equal(t, "client_apis", ClientAPI{}.TableName())
}
