package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantServiceLink_TableName(t *testing.T) {
	assert.Equal(t, "tenant_services", TenantServiceLink{}.TableName())
}
