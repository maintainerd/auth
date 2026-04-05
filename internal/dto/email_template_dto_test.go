package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maintainerd/auth/internal/model"
)

func validEmailCreate() EmailTemplateCreateRequestDto {
	return EmailTemplateCreateRequestDto{
		Name:     "Welcome Email",
		Subject:  "Welcome to our platform",
		BodyHTML: "<h1>Welcome!</h1>",
	}
}

func TestEmailTemplateCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, validEmailCreate().Validate())
	})

	t.Run("missing name", func(t *testing.T) {
		d := validEmailCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := validEmailCreate()
		d.Name = string(make([]byte, 101))
		require.Error(t, d.Validate())
	})

	t.Run("missing subject", func(t *testing.T) {
		d := validEmailCreate()
		d.Subject = ""
		require.Error(t, d.Validate())
	})

	t.Run("missing body_html", func(t *testing.T) {
		d := validEmailCreate()
		d.BodyHTML = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validEmailCreate()
		bad := "pending"
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid active status", func(t *testing.T) {
		d := validEmailCreate()
		s := model.StatusActive
		d.Status = &s
		assert.NoError(t, d.Validate())
	})
}

func TestEmailTemplateUpdateRequestDto_Validate(t *testing.T) {
	d := EmailTemplateUpdateRequestDto{
		Name:     "Updated Email",
		Subject:  "Updated subject",
		BodyHTML: "<h1>Updated!</h1>",
	}
	assert.NoError(t, d.Validate())

	d.Name = ""
	require.Error(t, d.Validate())
}

func TestEmailTemplateUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, EmailTemplateUpdateStatusRequestDto{Status: model.StatusActive}.Validate())
	assert.NoError(t, EmailTemplateUpdateStatusRequestDto{Status: model.StatusInactive}.Validate())
	require.Error(t, EmailTemplateUpdateStatusRequestDto{Status: ""}.Validate())
	require.Error(t, EmailTemplateUpdateStatusRequestDto{Status: "bad"}.Validate())
}

func TestEmailTemplateFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := EmailTemplateFilterDto{PaginationRequestDto: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := EmailTemplateFilterDto{
			PaginationRequestDto: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})

	t.Run("valid status list", func(t *testing.T) {
		f := EmailTemplateFilterDto{
			PaginationRequestDto: validPagination(),
			Status:               []string{model.StatusActive, model.StatusInactive},
		}
		assert.NoError(t, f.Validate())
	})
}

