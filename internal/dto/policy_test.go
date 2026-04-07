package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/maintainerd/auth/internal/model"
)

func validPolicyDoc() datatypes.JSON {
	return datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["user:*"],"resource":["auth:*"]}]}`)
}

func TestPolicyDocument_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		d := PolicyDocument{
			Version:   "v1",
			Statement: []PolicyStatement{{Effect: model.PolicyEffectAllow, Action: []string{"user:*"}, Resource: []string{"auth:*"}}},
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("missing version", func(t *testing.T) {
		d := PolicyDocument{Statement: []PolicyStatement{{Effect: model.PolicyEffectAllow, Action: []string{"x"}, Resource: []string{"y"}}}}
		require.Error(t, d.Validate())
	})

	t.Run("missing statement", func(t *testing.T) {
		d := PolicyDocument{Version: "v1"}
		require.Error(t, d.Validate())
	})
}

func TestPolicyStatement_Validate(t *testing.T) {
	t.Run("valid allow", func(t *testing.T) {
		s := PolicyStatement{Effect: model.PolicyEffectAllow, Action: []string{"user:*"}, Resource: []string{"auth:*"}}
		assert.NoError(t, s.Validate())
	})

	t.Run("valid deny", func(t *testing.T) {
		s := PolicyStatement{Effect: model.PolicyEffectDeny, Action: []string{"role:delete"}, Resource: []string{"auth:roles"}}
		assert.NoError(t, s.Validate())
	})

	t.Run("invalid effect", func(t *testing.T) {
		s := PolicyStatement{Effect: "maybe", Action: []string{"user:*"}, Resource: []string{"auth:*"}}
		require.Error(t, s.Validate())
	})

	t.Run("missing action", func(t *testing.T) {
		s := PolicyStatement{Effect: model.PolicyEffectAllow, Resource: []string{"auth:*"}}
		require.Error(t, s.Validate())
	})

	t.Run("missing resource", func(t *testing.T) {
		s := PolicyStatement{Effect: model.PolicyEffectAllow, Action: []string{"user:*"}}
		require.Error(t, s.Validate())
	})
}

func TestPolicyCreateRequestDto_Validate(t *testing.T) {
	valid := PolicyCreateRequestDTO{
		Name:     "auth:user:read",
		Document: validPolicyDoc(),
		Version:  "v1",
		Status:   model.StatusActive,
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid name characters", func(t *testing.T) {
		d := valid
		d.Name = "Auth Policy"
		require.Error(t, d.Validate())
	})

	t.Run("missing document", func(t *testing.T) {
		d := valid
		d.Document = nil
		require.Error(t, d.Validate())
	})

	t.Run("invalid document structure", func(t *testing.T) {
		d := valid
		d.Document = datatypes.JSON(`{"version":"v1"}`)
		require.Error(t, d.Validate())
	})

	t.Run("missing version", func(t *testing.T) {
		d := valid
		d.Version = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := valid
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})
}

func TestPolicyFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := PolicyFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		f := PolicyFilterDTO{PaginationRequestDTO: validPagination(), Status: []string{"bad"}}
		require.Error(t, f.Validate())
	})
}

func TestPolicyStatusUpdateDto_Validate(t *testing.T) {
	assert.NoError(t, PolicyStatusUpdateDTO{Status: model.StatusActive}.Validate())
	require.Error(t, PolicyStatusUpdateDTO{Status: ""}.Validate())
	require.Error(t, PolicyStatusUpdateDTO{Status: "unknown"}.Validate())
}

func TestPolicyUpdateRequestDto_Validate(t *testing.T) {
	valid := PolicyUpdateRequestDTO{
		Name:     "auth:user:read",
		Document: validPolicyDoc(),
		Version:  "v1",
		Status:   model.StatusActive,
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := valid
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := valid
		d.Status = "unknown"
		require.Error(t, d.Validate())
	})

	t.Run("missing version", func(t *testing.T) {
		d := valid
		d.Version = ""
		require.Error(t, d.Validate())
	})
}

func TestPolicyServicesFilterDto_Validate(t *testing.T) {
	t.Run("valid empty", func(t *testing.T) {
		f := PolicyServicesFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		f := PolicyServicesFilterDTO{PaginationRequestDTO: validPagination(), Name: strPtr(string(make([]byte, 151)))}
		require.Error(t, f.Validate())
	})

	t.Run("display_name too long", func(t *testing.T) {
		f := PolicyServicesFilterDTO{PaginationRequestDTO: validPagination(), DisplayName: strPtr(string(make([]byte, 151)))}
		require.Error(t, f.Validate())
	})

	t.Run("description too long", func(t *testing.T) {
		f := PolicyServicesFilterDTO{PaginationRequestDTO: validPagination(), Description: strPtr(string(make([]byte, 501)))}
		require.Error(t, f.Validate())
	})
}

func TestValidatePolicyDocumentStructure(t *testing.T) {
	t.Run("non-datatypes.JSON value returns error", func(t *testing.T) {
		err := validatePolicyDocumentStructure("just-a-string")
		require.Error(t, err)
	})

	t.Run("invalid JSON bytes returns error", func(t *testing.T) {
		err := validatePolicyDocumentStructure(datatypes.JSON([]byte("not-json")))
		require.Error(t, err)
	})

	t.Run("invalid statement returns error", func(t *testing.T) {
		// Document structure is valid but statement has bad effect → hits statement.Validate() error
		doc := datatypes.JSON(`{"version":"v1","statement":[{"effect":"bad","action":["x"],"resource":["y"]}]}`)
		err := validatePolicyDocumentStructure(doc)
		require.Error(t, err)
	})
}
