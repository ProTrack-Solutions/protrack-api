package domain

import (
	"time"

	globaldomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/google/uuid"
)

type PaginationParams struct {
	globaldomain.PaginationParams
	SaleID       string `form:"saleId"       validate:"omitempty,uuid"`
	Status       string `form:"status"       validate:"omitempty,oneof=pending paid overdue partial cancelled"`
	StartDueDate string `form:"startDueDate" validate:"omitempty,datetime=2006-01-02"`
	EndDueDate   string `form:"endDueDate"   validate:"omitempty,datetime=2006-01-02"`
	OrderField   string `form:"orderField"   validate:"oneof=due_date created_at"`
}

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

type ListAccountsReceivables struct {
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
	DaysOverdue       int64     `json:"days_overdue"`
}

type ListAccountsReceivablesResponse struct {
	globaldomain.PaginatedResponse[ListAccountsReceivables]
	AmountOverdue float64 `json:"amount_overdue"`
	Amount        float64 `json:"amount"`
}
