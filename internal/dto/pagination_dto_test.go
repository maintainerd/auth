package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared test helpers (used across all _test.go files in this package)
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func validPagination() PaginationRequestDto {
	return PaginationRequestDto{Page: 1, Limit: 10}
}

// ---------------------------------------------------------------------------
// PaginationRequestDto tests
// ---------------------------------------------------------------------------

func TestPaginationRequestDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     PaginationRequestDto
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal",
			dto:  PaginationRequestDto{Page: 1, Limit: 10},
		},
		{
			name: "valid with sort",
			dto:  PaginationRequestDto{Page: 2, Limit: 25, SortBy: "name", SortOrder: SortOrderAsc},
		},
		{
			name: "valid sort desc",
			dto:  PaginationRequestDto{Page: 1, Limit: 50, SortOrder: SortOrderDesc},
		},
		{
			name:    "missing page",
			dto:     PaginationRequestDto{Page: 0, Limit: 10},
			wantErr: true,
		},
		{
			name:    "negative page",
			dto:     PaginationRequestDto{Page: -1, Limit: 10},
			wantErr: true,
		},
		{
			name:    "missing limit",
			dto:     PaginationRequestDto{Page: 1, Limit: 0},
			wantErr: true,
		},
		{
			name:    "negative limit",
			dto:     PaginationRequestDto{Page: 1, Limit: -5},
			wantErr: true,
		},
		{
			name:    "invalid sort order",
			dto:     PaginationRequestDto{Page: 1, Limit: 10, SortOrder: "random"},
			wantErr: true,
		},
		{
			name:    "sort_by exceeds 50 chars",
			dto:     PaginationRequestDto{Page: 1, Limit: 10, SortBy: string(make([]byte, 51))},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.dto.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSuccessResponseDto_Fields(t *testing.T) {
	dto := SuccessResponseDto{Message: "ok"}
	assert.Equal(t, "ok", dto.Message)
}

func TestPaginatedResponseDto_Fields(t *testing.T) {
	resp := PaginatedResponseDto[string]{
		Rows:       []string{"a", "b"},
		Total:      2,
		Page:       1,
		Limit:      10,
		TotalPages: 1,
	}
	assert.Len(t, resp.Rows, 2)
	assert.Equal(t, int64(2), resp.Total)
}
