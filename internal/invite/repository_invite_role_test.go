package invite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInviteRoleRepository(t *testing.T) {
	db, _ := newMockGormDB(t)
	repo := NewInviteRoleRepository(db)
	assert.NotNil(t, repo)
}
