package pagination

import (
	"net/http"
	"strconv"
)

const DefaultPageSize = 20

// ParseQuery extracts page, limit, sort_by, and sort_order from the
// request query string. Missing or invalid page/limit values fall back to
// page=1 and DefaultPageSize so callers never receive zero values that would
// fail the PaginationRequestDTO min-1 validation.
func ParseQuery(r *http.Request) PaginationRequestDTO {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = DefaultPageSize
	}
	return PaginationRequestDTO{
		Page:      page,
		Limit:     limit,
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}
}
