package domain

import "math"

type PaginationParams struct {
	Page    int32
	PerPage int32
}

type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int32 `json:"page"`
	PerPage    int32 `json:"per_page"`
	TotalPages int32 `json:"total_pages"`
}

func NewPaginatedResponse[T any](data []T, total int64, params PaginationParams) PaginatedResponse[T] {
	totalPages := int32(math.Ceil(float64(total) / float64(params.PerPage)))
	return PaginatedResponse[T]{
		Data:       data,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}
}
