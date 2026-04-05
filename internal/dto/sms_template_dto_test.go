package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func validSMSCreate() SMSTemplateCreateRequestDto {
	return SMSTemplateCreateRequestDto{
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

	t.Run("sender_id too long", func(t *testing.T) {
		d := validSMSCreate()
		long := string(make([]byte, 21))
		d.SenderID = &long
		require.Error(t, d.Validate())
	})

	t.Run("valid sender_id within limit", func(t *testing.T) {
		d := validSMSCreate()
		s := "MYAPP"
		d.SenderID = &s
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validSMSCreate()
		bad := "pending"
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid inactive status", func(t *testing.T) {
		d := validSMSCreate()
		s := model.StatusInactive
		d.Status = &s
		assert.NoError(t, d.Validate())
	})
}

func TestSMSTemplateUpdateRequestDto_Validate(t *testing.T) {
	d := SMSTemplateUpdateRequestDto{
		Name:    "Updated SMS",
		Message: "Your code is {{code}}",
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestSMSTemplateUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, SMSTemplateUpdateStatusRequestDto{Status: model.StatusActive}.Validate())
	assert.NoError(t, SMSTemplateUpdateStatusRequestDto{Status: model.StatusInactive}.Validate())
	require.Error(t, SMSTemplateUpdateStatusRequestDto{Status: ""}.Validate())
	require.Error(t, SMSTemplateUpdateStatusRequestDto{Status: "bad"}.Validate())
}

func TestSMSTemplateFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := SMSTemplateFilterDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := SMSTemplateFilterDto{
			PaginationRequestDto: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})

	t.Run("valid status list", func(t *testing.T) {
		f := SMSTemplateFilterDto{
			PaginationRequestDto: validPagination(),
			Status:               []string{model.StatusActive},
		}
		assert.NoError(t, f.Validate())
	})
}

