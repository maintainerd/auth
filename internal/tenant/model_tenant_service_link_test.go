package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantServiceLinkTableName(t *testing.T) {
	assert.Equal(t, "tenant_services", TenantServiceLink{}.TableName())
}
