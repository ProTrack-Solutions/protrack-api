package domain

import (
	"time"

	globaldomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/google/uuid"
)

type CreateInvoicementHistoryRequest struct {
	SubscriptionID  uuid.UUID `json:"subscription_id"`
	MpPaymentID     string    `json:"mp_payment_id"`
	AmountCents     int32     `json:"amount_cents"`
	Status          string    `json:"status"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	PaidAt          time.Time `json:"paid_at"`
}

type UpdateInvoceStatusRequest struct {
	Status string    `json:"status"`
	PaidAt time.Time `json:"paid_at"`
}

type InvoiceHistoryResponse struct {
	ID              uuid.UUID `json:"id"`
	SubscriptionID  uuid.UUID `json:"subscription_id"`
	CompanyID       uuid.UUID `json:"company_id"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	MpPaymentID     string    `json:"mp_payment_id"`
	AmountCents     int32     `json:"amount_cents"`
	Status          string    `json:"status"`
	PaidAt          time.Time `json:"paid_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type InvoiceHistoryPaginatedResponse struct {
	globaldomain.PaginatedResponse[InvoiceHistoryResponse]
}
