package idp

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationFlowCreateRequestDto_Validate(t *testing.T) {
	valid := RegistrationFlowCreateRequestDTO{
		Name:        "default-flow",
		Description: "The default registration flow",
		ClientUUID:  uuid.New().String(),
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := valid
		d.Name = strings.Repeat("x", 101)
		require.Error(t, d.Validate())
	})

	t.Run("name at the 100-character boundary is accepted", func(t *testing.T) {
		d := valid
		d.Name = strings.Repeat("x", 100)
		assert.NoError(t, d.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		d := valid
		d.Description = strings.Repeat("x", 501)
		require.Error(t, d.Validate())
	})

	t.Run("missing description", func(t *testing.T) {
		d := valid
		d.Description = ""
		require.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		bad := "unknown"
		d := valid
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid active status", func(t *testing.T) {
		s := shared.StatusActive
		d := valid
		d.Status = &s
		assert.NoError(t, d.Validate())
	})

	t.Run("missing client_uuid", func(t *testing.T) {
		d := valid
		d.ClientUUID = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid client_uuid", func(t *testing.T) {
		d := valid
		d.ClientUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})

	t.Run("valid required fields", func(t *testing.T) {
		fields := []string{"email", "fullname", "phone"}
		d := valid
		d.RequiredFields = &fields
		assert.NoError(t, d.Validate())
	})

	t.Run("unknown required field is rejected", func(t *testing.T) {
		fields := []string{"address"}
		d := valid
		d.RequiredFields = &fields
		require.Error(t, d.Validate())
	})

	t.Run("duplicate required field is rejected", func(t *testing.T) {
		fields := []string{"email", " Email "}
		d := valid
		d.RequiredFields = &fields
		err := d.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate required field")
	})

	t.Run("empty required_fields array is accepted", func(t *testing.T) {
		fields := []string{}
		d := valid
		d.RequiredFields = &fields
		assert.NoError(t, d.Validate())
	})

	// role_ids gained an Each(is.UUID) rule so a malformed id is rejected at the
	// edge rather than silently dropped by the handler's parseUUIDList.
	t.Run("valid role_ids", func(t *testing.T) {
		d := valid
		d.RoleIDs = []string{uuid.New().String(), uuid.New().String()}
		assert.NoError(t, d.Validate())
	})

	t.Run("empty role_ids array is accepted", func(t *testing.T) {
		d := valid
		d.RoleIDs = []string{}
		assert.NoError(t, d.Validate())
	})

	t.Run("malformed role id is rejected", func(t *testing.T) {
		d := valid
		d.RoleIDs = []string{"not-a-uuid"}
		err := d.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid role UUID provided")
	})

	t.Run("one malformed role id among valid ones is rejected", func(t *testing.T) {
		d := valid
		d.RoleIDs = []string{uuid.New().String(), "nope"}
		require.Error(t, d.Validate())
	})

	t.Run("verification_required accepts both explicit values", func(t *testing.T) {
		for _, v := range []bool{true, false} {
			d := valid
			d.VerificationRequired = &v
			assert.NoError(t, d.Validate())
		}
	})
}

func TestRegistrationFlowUpdateRequestDto_Validate(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	t.Run("fully omitted body is valid (omitted means unchanged)", func(t *testing.T) {
		assert.NoError(t, RegistrationFlowUpdateRequestDTO{}.Validate())
	})

	t.Run("valid name and description", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{
			Name:        strPtr("updated-flow"),
			Description: strPtr("Updated description"),
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("present but empty name is rejected", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{Name: strPtr("")}
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{Name: strPtr(strings.Repeat("x", 101))}
		require.Error(t, d.Validate())
	})

	t.Run("name at the 100-character boundary is accepted", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{Name: strPtr(strings.Repeat("x", 100))}
		assert.NoError(t, d.Validate())
	})

	t.Run("present but empty description is accepted (clears it)", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{Description: strPtr("")}
		assert.NoError(t, d.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{Description: strPtr(strings.Repeat("x", 501))}
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{Status: strPtr("unknown")}
		require.Error(t, d.Validate())
	})

	for _, status := range []string{shared.StatusActive, shared.StatusInactive} {
		t.Run("valid status "+status, func(t *testing.T) {
			d := RegistrationFlowUpdateRequestDTO{Status: strPtr(status)}
			assert.NoError(t, d.Validate())
		})
	}

	t.Run("valid role_ids", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{RoleIDs: []string{uuid.New().String()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("empty role_ids array is valid (clears membership)", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{RoleIDs: []string{}}
		assert.NoError(t, d.Validate())
	})

	t.Run("malformed role id is rejected", func(t *testing.T) {
		d := RegistrationFlowUpdateRequestDTO{RoleIDs: []string{uuid.New().String(), "not-a-uuid"}}
		err := d.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid role UUID provided")
	})

	t.Run("verification_required accepts both explicit values", func(t *testing.T) {
		for _, v := range []bool{true, false} {
			d := RegistrationFlowUpdateRequestDTO{VerificationRequired: &v}
			assert.NoError(t, d.Validate())
		}
	})

	t.Run("valid required_fields", func(t *testing.T) {
		fields := []string{"email", "fullname", "phone"}
		d := RegistrationFlowUpdateRequestDTO{RequiredFields: &fields}
		assert.NoError(t, d.Validate())
	})

	t.Run("present but empty required_fields is accepted (clears it)", func(t *testing.T) {
		fields := []string{}
		d := RegistrationFlowUpdateRequestDTO{RequiredFields: &fields}
		assert.NoError(t, d.Validate())
	})

	t.Run("unsupported required field is rejected", func(t *testing.T) {
		fields := []string{"address"}
		d := RegistrationFlowUpdateRequestDTO{RequiredFields: &fields}
		err := d.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported required field")
	})

	t.Run("duplicate required field is rejected", func(t *testing.T) {
		fields := []string{"email", "EMAIL"}
		d := RegistrationFlowUpdateRequestDTO{RequiredFields: &fields}
		err := d.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate required field")
	})
}

func TestRegistrationFlowUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, RegistrationFlowUpdateStatusRequestDTO{Status: shared.StatusActive}.Validate())
	assert.NoError(t, RegistrationFlowUpdateStatusRequestDTO{Status: shared.StatusInactive}.Validate())
	require.Error(t, RegistrationFlowUpdateStatusRequestDTO{Status: ""}.Validate())
	require.Error(t, RegistrationFlowUpdateStatusRequestDTO{Status: "bad"}.Validate())
}

func TestRegistrationFlowFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: validPagination(), Status: []string{"bad"}}
		require.Error(t, f.Validate())
	})

	t.Run("invalid client_uuid", func(t *testing.T) {
		s := "not-a-uuid"
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: validPagination(), ClientUUID: &s}
		require.Error(t, f.Validate())
	})

	t.Run("valid client_uuid", func(t *testing.T) {
		s := uuid.New().String()
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: validPagination(), ClientUUID: &s}
		assert.NoError(t, f.Validate())
	})

	t.Run("multi-value status list is valid", func(t *testing.T) {
		f := RegistrationFlowFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{shared.StatusActive, shared.StatusInactive},
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("one invalid entry in the status list is rejected", func(t *testing.T) {
		f := RegistrationFlowFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{shared.StatusActive, "bogus"},
		}
		require.Error(t, f.Validate())
	})

	t.Run("search and is_system are unconstrained free filters", func(t *testing.T) {
		search := "partner"
		isSystem := true
		f := RegistrationFlowFilterDTO{
			PaginationRequestDTO: validPagination(),
			Search:               &search,
			IsSystem:             &isSystem,
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid pagination is rejected", func(t *testing.T) {
		f := RegistrationFlowFilterDTO{PaginationRequestDTO: PaginationRequestDTO{Page: 1, Limit: 10, SortOrder: "sideways"}}
		require.Error(t, f.Validate())
	})
}

// ------ RegistrationFlowRoleDTO tests ------

func TestRegistrationFlowAssignRolesRequestDto_Validate(t *testing.T) {
	t.Run("valid single role", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: []string{uuid.New().String()}}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing role_uuids", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{}
		require.Error(t, d.Validate())
	})

	t.Run("empty role_uuids", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: []string{}}
		require.Error(t, d.Validate())
	})

	t.Run("invalid uuid in list", func(t *testing.T) {
		d := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: []string{"not-a-uuid"}}
		require.Error(t, d.Validate())
	})
}

// Unbounded list and filter inputs are cheap DoS: each role entry costs a role
// lookup plus a permission lookup inside the write transaction, and each filter
// term reaches the database as LOWER(col) LIKE '%term%'.
func TestRegistrationFlowValidation_InputBounds(t *testing.T) {
	uuids := func(n int) []string {
		out := make([]string, 0, n)
		for range n {
			out = append(out, uuid.NewString())
		}
		return out
	}
	longTerm := strings.Repeat("a", 101)

	t.Run("create caps the role list at 50", func(t *testing.T) {
		ok := RegistrationFlowCreateRequestDTO{Name: "partner-signup", ClientUUID: uuid.NewString(), RoleIDs: uuids(50)}
		assert.NoError(t, ok.Validate())

		tooMany := RegistrationFlowCreateRequestDTO{Name: "partner-signup", ClientUUID: uuid.NewString(), RoleIDs: uuids(51)}
		err := tooMany.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "At most 50 roles")
	})

	t.Run("update caps the role list at 50", func(t *testing.T) {
		name := "partner-signup"
		tooMany := RegistrationFlowUpdateRequestDTO{Name: &name, RoleIDs: uuids(51)}
		err := tooMany.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "At most 50 roles")
	})

	t.Run("assign-roles caps the list at 50 and still requires at least one", func(t *testing.T) {
		none := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: []string{}}
		assert.Error(t, none.Validate())

		ok := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: uuids(50)}
		assert.NoError(t, ok.Validate())

		tooMany := RegistrationFlowAssignRolesRequestDTO{RoleUUIDs: uuids(51)}
		err := tooMany.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Between 1 and 50")
	})

	t.Run("filter terms are length-capped", func(t *testing.T) {
		search := RegistrationFlowFilterDTO{Search: &longTerm}
		errSearch := search.Validate()
		require.Error(t, errSearch)
		assert.Contains(t, errSearch.Error(), "Search term must not exceed 100")

		nameFilter := RegistrationFlowFilterDTO{Name: &longTerm}
		errName := nameFilter.Validate()
		require.Error(t, errName)
		assert.Contains(t, errName.Error(), "Name filter must not exceed 100")
	})
}
