package authn

import (
	"testing"

	"github.com/maintainerd/auth/internal/platform/pagination"
	"github.com/stretchr/testify/assert"
)

func TestFoundationAliasesAndConstants(t *testing.T) {
	assert.Equal(t, pagination.SortOrderAsc, SortOrderAsc)
	assert.Equal(t, pagination.SortOrderDesc, SortOrderDesc)

	result := PaginationResult[User]{Page: 1, Limit: 10}
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
}
