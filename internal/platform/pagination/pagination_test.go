package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// PaginationRequestDTO tests
// ---------------------------------------------------------------------------

func TestPaginationRequestDto_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dto     PaginationRequestDTO
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal",
			dto:  PaginationRequestDTO{Page: 1, Limit: 10},
		},
		{
			name: "valid with sort",
			dto:  PaginationRequestDTO{Page: 2, Limit: 25, SortBy: "name", SortOrder: SortOrderAsc},
		},
		{
			name: "valid sort desc",
			dto:  PaginationRequestDTO{Page: 1, Limit: 50, SortOrder: SortOrderDesc},
		},
		{
			name:    "missing page",
			dto:     PaginationRequestDTO{Page: 0, Limit: 10},
			wantErr: true,
		},
		{
			name:    "negative page",
			dto:     PaginationRequestDTO{Page: -1, Limit: 10},
			wantErr: true,
		},
		{
			name:    "missing limit",
			dto:     PaginationRequestDTO{Page: 1, Limit: 0},
			wantErr: true,
		},
		{
			name:    "negative limit",
			dto:     PaginationRequestDTO{Page: 1, Limit: -5},
			wantErr: true,
		},
		{
			name:    "invalid sort order",
			dto:     PaginationRequestDTO{Page: 1, Limit: 10, SortOrder: "random"},
			wantErr: true,
		},
		{
			name:    "sort_by exceeds 50 chars",
			dto:     PaginationRequestDTO{Page: 1, Limit: 10, SortBy: string(make([]byte, 51))},
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
	dto := SuccessResponseDTO{Message: "ok"}
	assert.Equal(t, "ok", dto.Message)
}

func TestPaginatedResponseDto_Fields(t *testing.T) {
	resp := PaginatedResponseDTO[string]{
		Rows:       []string{"a", "b"},
		Total:      2,
		Page:       1,
		Limit:      10,
		TotalPages: 1,
	}
	assert.Len(t, resp.Rows, 2)
	assert.Equal(t, int64(2), resp.Total)
}

// ---------------------------------------------------------------------------
// ParseQuery tests
// ---------------------------------------------------------------------------

func TestParseQuery_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	dto := ParseQuery(r)
	assert.Equal(t, 1, dto.Page)
	assert.Equal(t, DefaultPageSize, dto.Limit)
	assert.Equal(t, "", dto.SortBy)
	assert.Equal(t, "", dto.SortOrder)
}

func TestParseQuery_ValidParams(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=3&limit=50&sort_by=name&sort_order=asc", nil)
	dto := ParseQuery(r)
	assert.Equal(t, 3, dto.Page)
	assert.Equal(t, 50, dto.Limit)
	assert.Equal(t, "name", dto.SortBy)
	assert.Equal(t, "asc", dto.SortOrder)
}

func TestParseQuery_InvalidPageDefaultsTo1(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=abc&limit=10", nil)
	dto := ParseQuery(r)
	assert.Equal(t, 1, dto.Page)
	assert.Equal(t, 10, dto.Limit)
}

func TestParseQuery_NegativePageDefaultsTo1(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=-1&limit=10", nil)
	dto := ParseQuery(r)
	assert.Equal(t, 1, dto.Page)
}

func TestParseQuery_InvalidLimitDefaultsToDefault(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=1&limit=abc", nil)
	dto := ParseQuery(r)
	assert.Equal(t, 1, dto.Page)
	assert.Equal(t, DefaultPageSize, dto.Limit)
}

func TestParseQuery_ZeroLimitDefaultsToDefault(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=1&limit=0", nil)
	dto := ParseQuery(r)
	assert.Equal(t, DefaultPageSize, dto.Limit)
}

func TestParseQuery_NegativeLimitDefaultsToDefault(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=1&limit=-5", nil)
	dto := ParseQuery(r)
	assert.Equal(t, DefaultPageSize, dto.Limit)
}

func TestParseQuery_SortDesc(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=1&limit=25&sort_by=email&sort_order=desc", nil)
	dto := ParseQuery(r)
	assert.Equal(t, "email", dto.SortBy)
	assert.Equal(t, "desc", dto.SortOrder)
}
