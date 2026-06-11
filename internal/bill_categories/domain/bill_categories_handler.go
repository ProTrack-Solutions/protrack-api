package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateBillCategoriesRequest struct {
	CompanyID   uuid.UUID `json:"company_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type ToggleBillCategoriesActiveRequest struct {
	ID       uuid.UUID `json:"id"`
	IsActive bool      `json:"is_active"`
}

type BillCategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	CompanyID   uuid.UUID `json:"company_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}
