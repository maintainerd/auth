package branding

import (
	"testing"

	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validLoginTemplateCreate() LoginTemplateCreateRequestDTO {
	return LoginTemplateCreateRequestDTO{
		Name:     "Default Login",
		Template: shared.LoginTemplateModern,
	}
}

func TestLoginTemplateCreateRequestDto_Validate(t *testing.T) {
	t.Run("valid modern", func(t *testing.T) {
		assert.NoError(t, validLoginTemplateCreate().Validate())
	})

	t.Run("valid all template types", func(t *testing.T) {
		templates := []string{
			shared.LoginTemplateModern,
			shared.LoginTemplateClassic,
			shared.LoginTemplateMinimal,
			shared.LoginTemplateCorporate,
			shared.LoginTemplateCreative,
			shared.LoginTemplateCustom,
		}
		for _, tmpl := range templates {
			d := validLoginTemplateCreate()
			d.Template = tmpl
			assert.NoError(t, d.Validate(), "template: %s", tmpl)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		d := validLoginTemplateCreate()
		d.Name = ""
		require.Error(t, d.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		d := validLoginTemplateCreate()
		d.Name = string(make([]byte, 101))
		require.Error(t, d.Validate())
	})

	t.Run("missing template", func(t *testing.T) {
		d := validLoginTemplateCreate()
		d.Template = ""
		require.Error(t, d.Validate())
	})

	t.Run("invalid template value", func(t *testing.T) {
		d := validLoginTemplateCreate()
		d.Template = "invalid-template"
		require.Error(t, d.Validate())
	})

	t.Run("invalid status", func(t *testing.T) {
		d := validLoginTemplateCreate()
		bad := "pending"
		d.Status = &bad
		require.Error(t, d.Validate())
	})

	t.Run("valid active status", func(t *testing.T) {
		d := validLoginTemplateCreate()
		s := shared.StatusActive
		d.Status = &s
		assert.NoError(t, d.Validate())
	})
}

func TestLoginTemplateUpdateRequestDto_Validate(t *testing.T) {
	d := LoginTemplateUpdateRequestDTO{
		Name:     "Updated Login",
		Template: shared.LoginTemplateClassic,
	}
	assert.NoError(t, d.Validate())

	d.Template = "bad"
	require.Error(t, d.Validate())
}

func TestLoginTemplateUpdateStatusRequestDto_Validate(t *testing.T) {
	assert.NoError(t, LoginTemplateUpdateStatusRequestDTO{Status: shared.StatusActive}.Validate())
	assert.NoError(t, LoginTemplateUpdateStatusRequestDTO{Status: shared.StatusInactive}.Validate())
	require.Error(t, LoginTemplateUpdateStatusRequestDTO{Status: ""}.Validate())
	require.Error(t, LoginTemplateUpdateStatusRequestDTO{Status: "bad"}.Validate())
}

func TestLoginTemplateFilterDto_Validate(t *testing.T) {
	t.Run("valid with pagination", func(t *testing.T) {
		f := LoginTemplateFilterDTO{PaginationRequestDTO: validPagination()}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid template filter", func(t *testing.T) {
		bad := "unknown-template"
		f := LoginTemplateFilterDTO{PaginationRequestDTO: validPagination(), Template: &bad}
		require.Error(t, f.Validate())
	})

	t.Run("valid template filter", func(t *testing.T) {
		tmpl := shared.LoginTemplateMinimal
		f := LoginTemplateFilterDTO{PaginationRequestDTO: validPagination(), Template: &tmpl}
		assert.NoError(t, f.Validate())
	})

	t.Run("invalid status in list", func(t *testing.T) {
		f := LoginTemplateFilterDTO{
			PaginationRequestDTO: validPagination(),
			Status:               []string{"bad"},
		}
		require.Error(t, f.Validate())
	})
}
