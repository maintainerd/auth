package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func validIPRuleCreate() IPRestrictionRuleCreateRequestDto {
	return IPRestrictionRuleCreateRequestDto{
		Type:      model.IPRuleTypeAllow,
		IPAddress: "192.168.1.1",
	}
}

func TestIPRestrictionRuleCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validIPRuleCreate().Validate())
	})

	t.Run("valid all types", func(t *testing.T) {
		types := []string{
			model.IPRuleTypeAllow,
			model.IPRuleTypeDeny,
			model.IPRuleTypeWhitelist,
			model.IPRuleTypeBlacklist,
		}
		for _, ruleType := range types {
			d := validIPRuleCreate()
			d.Type = ruleType
			assert.NoError(t, d.Validate(), "type: %s", ruleType)
		}
	})

	t.Run("description too long", func(t *testing.T) {
		d := validIPRuleCreate()
		d.Description = string(make([]byte, 501))
		require.Error(t, d.Validate())
	})

	t.Run("missing type", func(t *testing.T) {
		d := validIPRuleCreate()
		d.Type = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid type", func(t *testing.T) {
		d := validIPRuleCreate()
		d.Type = "block"
		require.Error(t, d.Validate())
	})

	t.Run("missing ip_address", func(t *testing.T) {
		d := validIPRuleCreate()
		d.IPAddress = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid ip_address format", func(t *testing.T) {
		d := validIPRuleCreate()
		d.IPAddress = "not-an-ip"
		require.Error(t, d.Validate())
	})

	t.Run("ipv6 address rejected", func(t *testing.T) {
		d := validIPRuleCreate()
		d.IPAddress = "2001:db8::1"
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validIPRuleCreate()
		bad := "pending"
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid inactive status", func(t *testing.T) {
		d := validIPRuleCreate()
		s := model.StatusInactive
		d.Status = &s
		assert.NoError(t, d.Validate())
	})
}

func TestIPRestrictionRuleUpdateRequestDto_Validate(t *testing.T) {
	d := IPRestrictionRuleUpdateRequestDto{
		Type:      model.IPRuleTypeDeny,
		IPAddress: "10.0.0.1",
	}
	assert.NoError(t, d.Validate())

	d.Type = ""
	require.Error(t, d.Validate())
}

func TestIPRestrictionRuleUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, IPRestrictionRuleUpdateStatusRequestDto{Status: model.StatusActive}.Validate())
	assert.NoError(t, IPRestrictionRuleUpdateStatusRequestDto{Status: model.StatusInactive}.Validate())
	require.Error(t, IPRestrictionRuleUpdateStatusRequestDto{Status: ""}.Validate())
	require.Error(t, IPRestrictionRuleUpdateStatusRequestDto{Status: "bad"}.Validate())
}

func TestIPRestrictionRuleFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := IPRestrictionRuleFilterDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, f.Validate())
	})
}

