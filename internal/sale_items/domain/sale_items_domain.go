package domain

import (
	"github.com/google/uuid"
)

type CreateSaleItemRequest struct {
	SaleID    uuid.UUID `json:"sale_id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int32     `json:"quantity"`
	UnitPrice float64   `json:"unit_price"`
	Discount  float64   `json:"discount"`
}

type ListItemsFromPendingSaleRow struct {
	ID          uuid.UUID `json:"id"`
	SaleID      uuid.UUID `json:"sale_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int32     `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Discount    float64   `json:"discount"`
	ProductName string    `json:"product_name"`
}

type ListItemsByCompanyResponse struct {
	ID          uuid.UUID `json:"id"`
	SaleID      uuid.UUID `json:"sale_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int32     `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Discount    float64   `json:"discount"`
	ProductName string    `json:"product_name"`
}

type ListItemsByDateResponse struct {
	ID          uuid.UUID `json:"id"`
	SaleID      uuid.UUID `json:"sale_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Quantity    int32     `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Discount    float64   `json:"discount"`
	ProductName string    `json:"product_name"`
}
