package authevent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthEventConstants(t *testing.T) {
	assert.Equal(t, "AUTHN", AuthEventCategoryAuthn)
	assert.Equal(t, "AUTHZ", AuthEventCategoryAuthz)
	assert.Equal(t, "SESSION", AuthEventCategorySession)
	assert.Equal(t, "USER", AuthEventCategoryUser)
	assert.Equal(t, "SYSTEM", AuthEventCategorySystem)
	assert.Equal(t, "INFO", AuthEventSeverityInfo)
	assert.Equal(t, "WARN", AuthEventSeverityWarn)
	assert.Equal(t, "CRITICAL", AuthEventSeverityCritical)
	assert.Equal(t, "success", AuthEventResultSuccess)
	assert.Equal(t, "failure", AuthEventResultFailure)
	assert.Equal(t, "authn_login_success", AuthEventTypeLoginSuccess)
	assert.Equal(t, "privilege_permissions_changed", AuthEventTypePrivilegePermissionsChanged)
	assert.Equal(t, "sys_crash", AuthEventTypeSystemCrash)
}
