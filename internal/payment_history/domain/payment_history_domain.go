package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreatePaymentHistoryRequest struct {
	CompanyID       uuid.UUID `json:"company_id"`
	CustomerID      uuid.UUID `json:"customer_id"`
	SaleID          uuid.UUID `json:"sale_id"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	UserID          uuid.UUID `json:"user_id"`
	AmountPaid      float64   `json:"amount_paid"`
	Notes           string    `json:"notes"`
}

type GetPaymentsByCustomerRequest struct {
	CompanyID  uuid.UUID `json:"company_id" form:"company_id"`
	CustomerID uuid.UUID `json:"customer_id" form:"customer_id"`
}

type GetPaymentsBySaleRequest struct {
	CompanyID uuid.UUID `json:"company_id" form:"company_id"`
	SaleID    uuid.UUID `json:"sale_id" form:"sale_id"`
}

type GetTotalReceivedByPeriodRequest struct {
	CompanyID     uuid.UUID `json:"company_id" form:"company_id"`
	PaymentDate   time.Time `json:"payment_date" form:"payment_date"`
	PaymentDate_2 time.Time `json:"payment_date_2" form:"payment_date_2"`
}

type ListPaymentHistoryRow struct {
	ID                uuid.UUID `json:"id"`
	AmountPaid        float64   `json:"amount_paid"`
	PaymentDate       time.Time `json:"payment_date"`
	Notes             string    `json:"notes"`
	CustomerName      string    `json:"customer_name"`
	UserName          string    `json:"user_name"`
	PaymentMethodName string    `json:"payment_method_name"`
	SaleID            uuid.UUID `json:"sale_id"`
}

type PaymentHistoryResponse struct {
	ID              uuid.UUID `json:"id"`
	CompanyID       uuid.UUID `json:"company_id"`
	CustomerID      uuid.UUID `json:"customer_id"`
	SaleID          uuid.UUID `json:"sale_id"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	UserID          uuid.UUID `json:"user_id"`
	AmountPaid      float64   `json:"amount_paid"`
	PaymentDate     time.Time `json:"payment_date"`
	Notes           string    `json:"notes"`
}

type GetPaymentsHistoryReportResponse struct {
	AmountPaid        float64   `json:"amount_paid" excel:"Valor Pago"`
	PaymentDate       time.Time `json:"payment_date" excel:"Data do Pagamento"`
	Notes             string    `json:"notes" excel:"Observações"`
	CustomerName      string    `json:"customer_name" excel:"Nome do Cliente"`
	UserName          string    `json:"user_name" excel:"Operador"`
	PaymentMethodName string    `json:"payment_method_name" excel:"Método de Pagamento"`
}
