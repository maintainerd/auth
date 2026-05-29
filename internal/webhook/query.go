package webhook

import (
	"net/http"
	"strconv"
)

func parsePaginationQuery(r *http.Request) PaginationRequestDTO {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 10
	}
	return PaginationRequestDTO{
		Page:      page,
		Limit:     limit,
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
	}
}
