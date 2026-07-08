package globaldomain

import "math"

type PaginationParams struct {
	Page    int32 `header:"page,default=1"`
	PerPage int32 `header:"per_page,default=10"`
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
