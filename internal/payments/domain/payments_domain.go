package domain

import "github.com/google/uuid"

type CreatePaymentRequest struct {
	CustomerID      uuid.UUID `json:"customer_id"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	AmountPaid      float64   `json:"amount_paid"`
	Notes           string    `json:"notes"`
}
