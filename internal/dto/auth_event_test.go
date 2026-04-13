package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validAuthEventFilter() AuthEventFilterDTO {
	cat := "AUTHN"
	sev := "INFO"
	res := "success"
	return AuthEventFilterDTO{
		Category: &cat,
		Severity: &sev,
		Result:   &res,
		PaginationRequestDTO: PaginationRequestDTO{
			Page:  1,
			Limit: 10,
		},
	}
}

func TestAuthEventFilterDTO_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		f := AuthEventFilterDTO{
			PaginationRequestDTO: PaginationRequestDTO{Page: 1, Limit: 10},
		}
		assert.NoError(t, f.Validate())
	})

	t.Run("valid full", func(t *testing.T) {
		assert.NoError(t, validAuthEventFilter().Validate())
	})

	t.Run("invalid category", func(t *testing.T) {
		f := validAuthEventFilter()
		bad := "INVALID"
		f.Category = &bad
		require.Error(t, f.Validate())
	})

	t.Run("valid categories", func(t *testing.T) {
		for _, cat := range []string{"AUTHN", "AUTHZ", "SESSION", "USER", "SYSTEM"} {
			f := validAuthEventFilter()
			c := cat
			f.Category = &c
			assert.NoError(t, f.Validate(), "category %s should be valid", cat)
		}
	})

	t.Run("invalid severity", func(t *testing.T) {
		f := validAuthEventFilter()
		bad := "LOW"
		f.Severity = &bad
		require.Error(t, f.Validate())
	})

	t.Run("valid severities", func(t *testing.T) {
		for _, sev := range []string{"INFO", "WARN", "CRITICAL"} {
			f := validAuthEventFilter()
			s := sev
			f.Severity = &s
			assert.NoError(t, f.Validate(), "severity %s should be valid", sev)
		}
	})

	t.Run("invalid result", func(t *testing.T) {
		f := validAuthEventFilter()
		bad := "maybe"
		f.Result = &bad
		require.Error(t, f.Validate())
	})

	t.Run("valid results", func(t *testing.T) {
		for _, r := range []string{"success", "failure"} {
			f := validAuthEventFilter()
			v := r
			f.Result = &v
			assert.NoError(t, f.Validate(), "result %s should be valid", r)
		}
	})

	t.Run("event_type too long", func(t *testing.T) {
		f := validAuthEventFilter()
		long := strings.Repeat("a", 61)
		f.EventType = &long
		require.Error(t, f.Validate())
	})

	t.Run("page zero", func(t *testing.T) {
		f := validAuthEventFilter()
		f.Page = 0
		require.Error(t, f.Validate())
	})

	t.Run("limit zero", func(t *testing.T) {
		f := validAuthEventFilter()
		f.Limit = 0
		require.Error(t, f.Validate())
	})

	t.Run("limit exceeds max", func(t *testing.T) {
		f := validAuthEventFilter()
		f.Limit = 101
		require.Error(t, f.Validate())
	})

	t.Run("sort_by too long", func(t *testing.T) {
		f := validAuthEventFilter()
		f.SortBy = strings.Repeat("x", 51)
		require.Error(t, f.Validate())
	})

	t.Run("invalid sort_order", func(t *testing.T) {
		f := validAuthEventFilter()
		f.SortOrder = "random"
		require.Error(t, f.Validate())
	})

	t.Run("valid sort_order asc", func(t *testing.T) {
		f := validAuthEventFilter()
		f.SortOrder = "asc"
		assert.NoError(t, f.Validate())
	})

	t.Run("valid sort_order desc", func(t *testing.T) {
		f := validAuthEventFilter()
		f.SortOrder = "desc"
		assert.NoError(t, f.Validate())
	})

	t.Run("nil category is valid", func(t *testing.T) {
		f := validAuthEventFilter()
		f.Category = nil
		assert.NoError(t, f.Validate())
	})

	t.Run("nil severity is valid", func(t *testing.T) {
		f := validAuthEventFilter()
		f.Severity = nil
		assert.NoError(t, f.Validate())
	})

	t.Run("nil result is valid", func(t *testing.T) {
		f := validAuthEventFilter()
		f.Result = nil
		assert.NoError(t, f.Validate())
	})
}
