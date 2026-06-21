package branding

import (
	"testing"

	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validSMSCreate() SMSTemplateCreateRequestDTO {
	return SMSTemplateCreateRequestDTO{
		Name:    "OTP SMS",
		Message: "Your OTP is {{otp}}",
	}
}

func TestSMSTemplateCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validSMSCreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validSMSCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := validSMSCreate()
		d.Name = string(make([]byte, 101))
		require.Error(t, d.Validate())
	})

	t.Run("missing message", func(t *testing.T) {
		d := validSMSCreate()
		d.Message = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validSMSCreate()
		bad := "pending"
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid inactive status", func(t *testing.T) {
		d := validSMSCreate()
		s := shared.StatusInactive
		d.Status = &s
		assert.NoError(t, d.Validate())
	})
}

func TestSMSTemplateUpdateRequestDto_Validate(t *testing.T) {
	d := SMSTemplateUpdateRequestDTO{
		Message: "Your code is {{code}}",
	}
	assert.NoError(t, d.Validate())

	d.Message = ""
	require.Error(t, d.Validate())
}

func TestSMSTemplateUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, SMSTemplateUpdateStatusRequestDTO{Status: shared.StatusActive}.Validate())
	assert.NoError(t, SMSTemplateUpdateStatusRequestDTO{Status: shared.StatusInactive}.Validate())
	require.Error(t, SMSTemplateUpdateStatusRequestDTO{Status: ""}.Validate())
	require.Error(t, SMSTemplateUpdateStatusRequestDTO{Status: "bad"}.Validate())
}

func TestSMSTemplateFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := SMSTemplateFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := SMSTemplateFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})

	t.Run("valid status list", func(t *testing.T) {
		f := SMSTemplateFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{shared.StatusActive},
		}
		assert.NoError(t, f.Validate())
	})
}
