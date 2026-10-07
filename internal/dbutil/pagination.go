package dbutil

var DefaultPageSize = 20

// MaxPageSize caps the page size.
var MaxPageSize = 100

type Pagination struct {
	Page     int
	PageSize int
}

func (p Pagination) LimitOffset() (limit, offset int) {
	return LimitOffset(p.Page, p.PageSize)
}

// LimitOffset returns LIMIT and OFFSET parameter for SQL.
// The first page is page 1.
func LimitOffset(page, pageSize int) (limit, offset int) {
	if page <= 0 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	pageSize = min(pageSize, MaxPageSize)
	return pageSize, (page - 1) * pageSize
}
