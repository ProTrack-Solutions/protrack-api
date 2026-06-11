package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreatePaymentMethodRequest struct {
	CompanyID uuid.UUID `json:"company_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
}

type PaymentMethodResponse struct {
	ID        uuid.UUID   `json:"id"`
	CompanyID uuid.UUID   `json:"company_id"`
	Name      string      `json:"name"`
	Type      interface{} `json:"type"`
	IsActive  bool        `json:"is_active"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type TogglePaymentMethodActiveRequest struct {
	ID       uuid.UUID `json:"id"`
	IsActive bool      `json:"is_active"`
}

type GetPaymentMethodsStatsResponse struct {
	PaymentMethod    string  `json:"payment_method"`
	PercentageMethod float64 `json:"percentage_method"`
}
