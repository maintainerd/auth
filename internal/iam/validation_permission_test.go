package iam

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionCreateRequestDto_Validate(t *testing.T) {
	valid := PermissionCreateRequestDTO{
		Name:        "perm:read",
		Description: "Read permission for all users",
		Status:      shared.StatusActive,
		APIUUID:     uuid.New().String(),
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too short", func(t *testing.T) {
		d := valid
		d.Name = "ab"
		require.Error(t, d.Validate())
	})

	t.Run("description too short is now valid", func(t *testing.T) {
		d := valid
		d.Description = "short"
		require.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := valid
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("missing api_id", func(t *testing.T) {
		d := valid
		d.APIUUID = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid api_id uuid", func(t *testing.T) {
		d := valid
		d.APIUUID = "not-a-uuid"
		require.Error(t, d.Validate())
	})

	// Name used to be any 3-50 character string. PermissionMiddleware matches route
	// guards on the exact name, so an admin holding permission:create could mint a
	// permission literally called "tenant:delete" in their own tenant and satisfy
	// every route that requires it.
	t.Run("reserved seeded namespaces are refused", func(t *testing.T) {
		for _, name := range []string{
			"tenant:delete", "tenant:create", "user:delete", "role:update",
			"permission:create", "client:secret:read", "public:login",
			"account:user:read:self", "system:read",
		} {
			d := valid
			d.Name = name
			assert.ErrorContains(t, d.Validate(), "reserved", "%q must be refused", name)
		}
	})

	t.Run("names outside the seeded namespaces stay allowed", func(t *testing.T) {
		for _, name := range []string{
			"users:read:own", "reports:read", "billing:invoice:export",
			"widget-store:order:refund:own", "auth_events:read",
		} {
			d := valid
			d.Name = name
			assert.NoError(t, d.Validate(), "%q is a legitimate tenant permission", name)
		}
	})

	t.Run("malformed names are refused", func(t *testing.T) {
		for _, name := range []string{
			"nocolon",             // a bare word is not an action on a resource
			"Reports:Read",        // guards compare exactly, so case must not vary
			"reports:read admin",  // whitespace
			"reports::read",       // empty segment
			":read",               // empty namespace
			"reports:",            // empty action
			"a:b:c:d:e",           // deeper than any seeded name
			"reports:read;DROP--", // punctuation with no business in a guard
		} {
			d := valid
			d.Name = name
			assert.Error(t, d.Validate(), "%q must be refused", name)
		}
	})
}

func TestPermissionUpdateRequestDto_Validate(t *testing.T) {
	d := PermissionUpdateRequestDTO{
		Name:        "perm:write",
		Description: "Write permission for all resources",
		Status:      shared.StatusInactive,
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())

	// Update carries the same name rules as create. Without them the guard is
	// bypassed in two calls: create "reports:read", then rename it "tenant:delete".
	d.Name = "tenant:delete"
	assert.ErrorContains(t, d.Validate(), "reserved")
	d.Name = "nocolon"
	require.Error(t, d.Validate())
}

func TestPermissionStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, PermissionStatusUpdateDTO{Status: shared.StatusActive}.Validate())
	assert.NoError(t, PermissionStatusUpdateDTO{Status: shared.StatusInactive}.Validate())
	require.Error(t, PermissionStatusUpdateDTO{Status: ""}.Validate())
	require.Error(t, PermissionStatusUpdateDTO{Status: "bad"}.Validate())
}

func TestPermissionFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := PermissionFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		bad := "unknown"
		f := PermissionFilterDTO{PaginationRequestDTO: validPagination(), Status: &bad}
		require.Error(t, f.Validate())
	})

	t.Run("valid status filter", func(t *testing.T) {
		s := shared.StatusActive
		f := PermissionFilterDTO{PaginationRequestDTO: validPagination(), Status: &s}
		assert.NoError(t, f.Validate())
	})
}
