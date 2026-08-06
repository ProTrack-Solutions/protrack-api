package globaldomain

import "math"

// PaginationParams defines the query parameters accepted in the headers for paginated list endpoints.
type PaginationParams struct {
	// Page is the page number to retrieve.
	Page int32 `header:"Page,default=1" example:"1" minimum:"1"`
	// PerPage is the number of items per page.
	PerPage int32 `header:"Perpage,default=10" example:"10" minimum:"1"`
	// Search filters the list by a text term.
	Search string `header:"Search" example:"joao"`
	// OrderBy defines the ordering direction for the results.
	OrderBy string `json:"order_by" header:"OrderBy" example:"asc" validate:"oneof=asc desc"`
	// StartDate filters records created from this date.
	StartDate string `header:"StartDate" example:"2024-01-01" validate:"omitempty,datetime=2006-01-02"`
	// EndDate filters records created until this date.
	EndDate string `header:"EndDate" example:"2024-12-31" validate:"omitempty,datetime=2006-01-02"`
}

type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	Page       int32 `json:"page"`
	PerPage    int32 `json:"per_page"`
	TotalRows  int64 `json:"total_rows"`
	TotalPages int32 `json:"total_pages"`
}

func NewPaginatedResponse[T any](data []T, total int64, params PaginationParams) PaginatedResponse[T] {
	totalPages := int32(math.Ceil(float64(total) / float64(params.PerPage)))
	return PaginatedResponse[T]{
		Data:       data,
		TotalRows:  total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}
}
