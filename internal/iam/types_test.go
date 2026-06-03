package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTypesDTOs(t *testing.T) {
	permissionID := uuid.New()
	role := RoleResponseDTO{
		RoleUUID:    uuid.New(),
		Name:        "admin",
		Description: "Admin",
		Permissions: &[]PermissionResponseDTO{
			{PermissionUUID: permissionID, Name: "read"},
		},
		Status: "active",
	}
	service := ServiceResponseDTO{
		ServiceUUID: uuid.New(),
		Name:        "iam",
		APICount:    2,
		PolicyCount: 3,
	}

	assert.Equal(t, "admin", role.Name)
	assert.Equal(t, permissionID, (*role.Permissions)[0].PermissionUUID)
	assert.Equal(t, int64(2), service.APICount)
	assert.Equal(t, int64(3), service.PolicyCount)
}
