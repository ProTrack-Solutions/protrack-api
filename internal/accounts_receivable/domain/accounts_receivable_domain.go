package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateAccountReceivableRequest struct {
	CustomerID        uuid.UUID `json:"customer_id"`
	SaleID            uuid.UUID `json:"sale_id"`
	TotalAmount       float64   `json:"total_amount"`
	Balance           float64   `json:"balance"`
	DueDate           string    `json:"due_date"`
	InstallmentNumber int64     `json:"installment_number"`
	TotalInstallments int64     `json:"total_installments"`
}

type GetCustomerDebtSummaryRow struct {
	TotalCount    int32   `json:"total_count"`
	TotalBalance  float64 `json:"total_balance"`
	OldestDueDate string  `json:"oldest_due_date"`
}

type ListOverdueReceivablesRow struct {
	ID                uuid.UUID `json:"id"`
	CompanyID         uuid.UUID `json:"company_id"`
	CustomerID        uuid.UUID `json:"customer_id"`
	SaleID            uuid.UUID `json:"sale_id"`
	TotalAmount       float64   `json:"total_amount"`
	Balance           float64   `json:"balance"`
	DueDate           string    `json:"due_date"`
	InstallmentNumber int64     `json:"installment_number"`
	TotalInstallments int64     `json:"total_installments"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         uuid.UUID `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         uuid.UUID `json:"updated_by"`
	DeletedAt         time.Time `json:"deleted_at"`
	CustomerName      string    `json:"customer_name"`
	DaysOverdue       int32     `json:"days_overdue"`
}

type UpdateAccountReceivableBalanceRequest struct {
	Balance float64 `json:"balance"`
}

type AccountsReceivableResponse struct {
	ID                uuid.UUID `json:"id"`
	CompanyID         uuid.UUID `json:"company_id"`
	CustomerID        uuid.UUID `json:"customer_id"`
	SaleID            uuid.UUID `json:"sale_id"`
	TotalAmount       float64   `json:"total_amount"`
	Balance           float64   `json:"balance"`
	DueDate           string    `json:"due_date"`
	InstallmentNumber int64     `json:"installment_number"`
	TotalInstallments int64     `json:"total_installments"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         uuid.UUID `json:"created_by"`
	UpdatedAt         time.Time `json:"updated_at"`
	UpdatedBy         uuid.UUID `json:"updated_by"`
	DeletedAt         time.Time `json:"deleted_at"`
}

type GetTotalPendingAndOverdueResponse struct {
	TotalPending float64 `json:"total_pending"`
	TotalOverdue float64 `json:"total_overdue"`
}
