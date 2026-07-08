package domain

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
)

type CreateBillPayableRequest struct {
	VendorID        uuid.UUID `json:"vendor_id"`
	CategoryID      uuid.UUID `json:"category_id"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	Amount          float64   `json:"amount"`
	DueDate         string    `json:"due_date"`
	Status          string    `json:"status"`
	Description     string    `json:"description"`
	Notes           string    `json:"notes"`
}

type GetBillsByStatusRequest struct {
	CompanyID uuid.UUID   `json:"company_id"`
	Status    interface{} `json:"status"`
}

type ListBillsPayableRow struct {
	ID                uuid.UUID `json:"id"`
	CompanyID         uuid.UUID `json:"company_id"`
	VendorID          uuid.UUID `json:"vendor_id"`
	CategoryID        uuid.UUID `json:"category_id"`
	PaymentMethodID   uuid.UUID `json:"payment_method_id"`
	Amount            float64   `json:"amount"`
	DueDate           string    `json:"due_date"`
	Status            string    `json:"status"`
	Description       string    `json:"description"`
	ScheduledDate     string    `json:"scheduled_date"`
	PaymentDate       string    `json:"payment_date"`
	AmountPaid        float64   `json:"amount_paid"`
	Notes             string    `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	VendorName        string    `json:"vendor_name"`
	CategoryName      string    `json:"category_name"`
	PaymentMethodName string    `json:"payment_method_name"`
}

type PayBillRequest struct {
	ID              uuid.UUID `json:"id"`
	CompanyID       uuid.UUID `json:"company_id"`
	PaymentDate     string    `json:"payment_date"`
	AmountPaid      float64   `json:"amount_paid"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
}

type UpdateBillPayableRequest struct {
	ID              uuid.UUID   `json:"id"`
	CompanyID       uuid.UUID   `json:"company_id"`
	VendorID        uuid.UUID   `json:"vendor_id"`
	CategoryID      uuid.UUID   `json:"category_id"`
	PaymentMethodID uuid.UUID   `json:"payment_method_id"`
	Amount          float64     `json:"amount"`
	DueDate         string      `json:"due_date"`
	Status          interface{} `json:"status"`
	Description     string      `json:"description"`
	Notes           string      `json:"notes"`
}

type BillsPayableResponse struct {
	ID              uuid.UUID `json:"id"`
	CompanyID       uuid.UUID `json:"company_id"`
	VendorID        uuid.UUID `json:"vendor_id"`
	CategoryID      uuid.UUID `json:"category_id"`
	PaymentMethodID uuid.UUID `json:"payment_method_id"`
	Amount          float64   `json:"amount"`
	DueDate         string    `json:"due_date"`
	Status          string    `json:"status"`
	Description     string    `json:"description"`
	ScheduledDate   string    `json:"scheduled_date"`
	PaymentDate     string    `json:"payment_date"`
	AmountPaid      float64   `json:"amount_paid"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type GetBillsByIdRequest struct {
	CompanyID uuid.UUID `json:"company_id"`
	ID        uuid.UUID `json:"id"`
}

type ScheduleBillRequest struct {
	ID            uuid.UUID `json:"id"`
	CompanyID     uuid.UUID `json:"company_id"`
	ScheduledDate string    `json:"scheduled_date"`
}

type GetBillsPayableSummaryResponse struct {
	TotalQuantity  int32   `json:"total_quantity"`
	TotalToPay     float64 `json:"total_to_pay"`
	TotalOverdue   float64 `json:"total_overdue"`
	TotalScheduled float64 `json:"total_scheduled"`
	GeneralStatus  string  `json:"general_status"`
}

func ApplyUpdateBillPayableParams(req UpdateBillPayableRequest, arg *db.UpdateBillPayableParams) {
	if req.VendorID != uuid.Nil {
		arg.VendorID = pgconv.ParseUUIDToPgType(req.VendorID)
	}

	if req.CategoryID != uuid.Nil {
		arg.CategoryID = pgconv.ParseUUIDToPgType(req.CategoryID)
	}

	if req.PaymentMethodID != uuid.Nil {
		arg.PaymentMethodID = pgconv.ParseUUIDToPgType(req.PaymentMethodID)
	}

	if req.Amount != 0 {
		arg.Amount = pgconv.Float64ToPgNumeric(req.Amount)
	}

	if req.DueDate != "" {
		arg.DueDate = pgconv.StringToPgDate(req.DueDate)
	}

	if req.Status != nil {
		arg.Status = req.Status
	}

	if req.Description != "" {
		arg.Description = pgconv.ParseStringToPgText(req.Description)
	}

	if req.Notes != "" {
		arg.Notes = pgconv.ParseStringToPgText(req.Notes)
	}
}
