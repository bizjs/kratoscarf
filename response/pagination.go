package response

// PageRequest represents offset-based pagination input.
type PageRequest struct {
	Page     int    `json:"page" form:"page"`         // 1-based page number
	PageSize int    `json:"pageSize" form:"pageSize"` // items per page
	Sort     string `json:"sort" form:"sort"`         // e.g. "created_at desc"
}

// Normalize fills in defaults and clamps values.
func (p *PageRequest) Normalize(defaultSize, maxSize int) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = defaultSize
	}
	if p.PageSize > maxSize {
		p.PageSize = maxSize
	}
}

// Offset returns the SQL offset value.
func (p *PageRequest) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

// PageResponse wraps a paginated result set.
type PageResponse[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// NewPageResponse creates a PageResponse from items, total count, and the original request.
func NewPageResponse[T any](items []T, total int64, req PageRequest) *PageResponse[T] {
	if items == nil {
		items = []T{}
	}
	return &PageResponse[T]{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
}

// CursorRequest represents cursor-based pagination input.
type CursorRequest struct {
	Cursor string `json:"cursor" form:"cursor"`
	Limit  int    `json:"limit" form:"limit"`
}

// CursorResponse wraps a cursor-paginated result set.
type CursorResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

// NewCursorResponse creates a CursorResponse.
func NewCursorResponse[T any](items []T, nextCursor string, hasMore bool) *CursorResponse[T] {
	if items == nil {
		items = []T{}
	}
	return &CursorResponse[T]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
