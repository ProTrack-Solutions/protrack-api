package domain

import (
	"errors"
	"fmt"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"

	"github.com/google/uuid"
)

type CreateSaleRequest struct {
	CustomerID        uuid.UUID               `json:"customer_id"`
	DiscountAmount    float64                 `json:"discount_amount"`
	DueDays           int32                   `json:"due_days"`
	PaymentMethod     enums.PaymentMethod     `json:"payment_method"`
	InstallmentsCount int32                   `json:"installments_count"`
	Items             []CreateSaleItemRequest `json:"items"`
	Prohibited        float64                 `json:"prohibited"`
}

type CreateSaleItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int32     `json:"quantity"`
}

type DeleteSaleRequest struct {
	DeletedBy uuid.UUID `json:"deleted_by"`
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
}

type GetSaleByIdRequest struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
}

type GetSaleByIdRow struct {
	ID             uuid.UUID   `json:"id"`
	CustomerID     uuid.UUID   `json:"customer_id"`
	CompanyID      uuid.UUID   `json:"company_id"`
	SaleAt         time.Time   `json:"sale_at"`
	DiscountAmount float64     `json:"discount_amount"`
	Subtotal       float64     `json:"subtotal"`
	TotalAmount    float64     `json:"total_amount"`
	DueDays        int32       `json:"due_days"`
	PaymentMethod  interface{} `json:"payment_method"`
	Status         interface{} `json:"status"`
	CreatedAt      time.Time   `json:"created_at"`
	CreatedBy      uuid.UUID   `json:"created_by"`
	UpdatedAt      time.Time   `json:"updated_at"`
	UpdatedBy      uuid.UUID   `json:"updated_by"`
	DeletedAt      time.Time   `json:"deleted_at"`
	DeletedBy      uuid.UUID   `json:"deleted_by"`
	CustomerName   string      `json:"customer_name"`
}

type ListSalesRow struct {
	ID          uuid.UUID   `json:"id"`
	SaleAt      time.Time   `json:"sale_date"`
	TotalAmount float64     `json:"total_amount"`
	Status      interface{} `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
}

type UpdateSaleStatusRequest struct {
	Status    interface{} `json:"status"`
	UpdatedBy uuid.UUID   `json:"updated_by"`
	ID        uuid.UUID   `json:"id"`
	CompanyID uuid.UUID   `json:"company_id"`
}

type ListSalesByCompanyAndStatusRequest struct {
	CompanyID uuid.UUID `json:"company_id" form:"company_id"`
	Status    string    `json:"status" form:"status"`
}

type ListSalesByCompanyAndStatusRow struct {
	SaleID         uuid.UUID   `json:"sale_id"`
	TotalAmount    float64     `json:"total_amount"`
	DiscountAmount float64     `json:"discount_amount"`
	Status         interface{} `json:"status"`
	SaleDate       time.Time   `json:"sale_date"`
	ItemID         uuid.UUID   `json:"item_id"`
	ProductID      uuid.UUID   `json:"product_id"`
	Quantity       int32       `json:"quantity"`
	UnitPrice      float64     `json:"unit_price"`
	Discount       float64     `json:"discount"`
	ProductName    string      `json:"product_name"`
	CustomerName   string      `json:"customer_name"`
}

type GetSalesPerformanceSummaryRow struct {
	CurrentMonthCount   int64 `json:"current_month_count"`
	CurrentMonthRevenue int64 `json:"current_month_revenue"`
	LastMonthCount      int64 `json:"last_month_count"`
	LastMonthRevenue    int64 `json:"last_month_revenue"`
}

type GetTotalAmountSummaryRow struct {
	CurrentMonthSt   float64 `json:"current_month_st"`
	LastMonthSt      float64 `json:"last_month_st"`
	GrowthPercentage float64 `json:"growth_percentage"`
}

type GetTotalAmountByStatusRequest struct {
	CompanyID uuid.UUID   `json:"company_id"`
	Status    interface{} `json:"status"`
}

type UpdateOverdueSalesAndAccountsGlobalRow struct {
	SaleID     uuid.UUID `json:"sale_id"`
	CustomerID uuid.UUID `json:"customer_id"`
}

type ListSalesResponse struct {
	SaleID                 uuid.UUID   `json:"sale_id"`
	SaleAt                 time.Time   `json:"sale_at"`
	Subtotal               float64     `json:"subtotal"`
	DiscountAmount         float64     `json:"discount_amount"`
	TotalAmount            float64     `json:"total_amount"`
	InstallmentsCount      int32       `json:"installments_count"`
	PaymentMethod          interface{} `json:"payment_method"`
	SaleStatus             interface{} `json:"sale_status"`
	CustomerID             uuid.UUID   `json:"customer_id"`
	CustomerName           string      `json:"customer_name"`
	InstallmentTotalAmount float64     `json:"installment_total_amount"`
	DownPayment            float64     `json:"down_payments"`
}

type ListProductResponse struct {
	SaleItemID   uuid.UUID `json:"sale_item_id"`
	ProductID    uuid.UUID `json:"product_id"`
	Quantity     int32     `json:"quantity"`
	UnitPrice    float64   `json:"unit_price"`
	ItemDiscount float64   `json:"item_discount"`
	ProductName  string    `json:"product_name"`
}

type ListAccReceivableResponse struct {
	InstallmentID uuid.UUID `json:"installment_id"`

	InstallmentBalance float64 `json:"installment_balance"`
	DueDate            string  `json:"due_date"`
	InstallmentNumber  int     `json:"installment_number"`
	InstallmentStatus  string  `json:"installment_status"`
}

type ListSalesWithInstallmentsResponse struct {
	Sale          ListSalesResponse           `json:"sale"`
	Products      []ListProductResponse       `json:"products"`
	AccReceivable []ListAccReceivableResponse `json:"installment"`
}

type GetTop5RealProfitItemResponse struct {
	ProductsName      string  `json:"product_name"`
	ProductRealProfit float64 `json:"product_real_profit"`
	TotalSale         float64 `json:"total_sale"`
}

type GetPerformanceMonthResponse struct {
	Mount      string  `json:"mount"`
	RealProfit float64 `json:"real_profit"`
	TotalSale  float64 `json:"total_sale"`
}

type GetTotalInvestmentCategoryResponse struct {
	CategoryName    string  `json:"category_name"`
	TotalInvestment float64 `json:"total_investment"`
	Amount          int     `json:"amount"`
	StockTurnover   float64 `json:"stock_turnover"`
}

type MarginDistributionResponse struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type GetPendingSalesDetailedReportResponse struct {
	SaleID                 uuid.UUID   `json:"sale_id"`
	SaleAt                 time.Time   `json:"sale_at"`
	Subtotal               float64     `json:"subtotal"`
	DiscountAmount         float64     `json:"discount_amount"`
	TotalAmount            float64     `json:"total_amount"`
	InstallmentsCount      int32       `json:"installments_count"`
	PaymentMethod          interface{} `json:"payment_method"`
	SaleStatus             interface{} `json:"sale_status"`
	CustomerID             uuid.UUID   `json:"customer_id"`
	CustomerName           string      `json:"customer_name"`
	SaleItemID             uuid.UUID   `json:"sale_item_id"`
	ProductID              uuid.UUID   `json:"product_id"`
	Quantity               int32       `json:"quantity"`
	UnitPrice              float64     `json:"unit_price"`
	ItemDiscount           float64     `json:"item_discount"`
	ProductName            string      `json:"product_name"`
	InstallmentID          uuid.UUID   `json:"installment_id"`
	InstallmentTotalAmount float64     `json:"installment_total_amount"`
	InstallmentBalance     float64     `json:"installment_balance"`
	DueDate                string      `json:"due_date"`
	InstallmentNumber      int         `json:"installment_number"`
	InstallmentStatus      string      `json:"installment_status"`
}

type SaleResponsePaginate struct {
	globalDomain.PaginatedResponse[ListSalesWithInstallmentsResponse]
	SalesCount    int64   `json:"sales_count"`
	TotalInvoiced float64 `json:"total_invoiced"`
	TotalPending  float64 `json:"total_pending"`
	SalesCanceled int64   `json:"sales_canceled"`
}

type UpdateSaleParams struct {
	DiscountAmount    float64             `json:"discount_amount"`
	DueDays           int32               `json:"due_days"`
	PaymentMethod     enums.PaymentMethod `json:"payment_method"`
	InstallmentsCount int32               `json:"installments_count"`
	Prohibited        float64             `json:"prohibited"`
}

type GetInventoryTurnoverResponse struct {
	InventoryTurnover float64 `json:"inventory_turnover"`
}
type UpdateOverdueSalesResponse struct {
	IDSale       uuid.UUID `json:"id_sale"`
	IDCustomer   uuid.UUID `json:"id_customer"`
	CustomerName string    `json:"customer_name"`
	PhoneNumber  string    `json:"phone_number"`
	Value        float64   `json:"value"`
	DueDate      time.Time `json:"due_date"`
	InstanceName string
	Message      string
}

func ValidateCreateSaleRequest(req CreateSaleRequest) error {
	if req.CustomerID == uuid.Nil {
		return errors.New("customer_id is required")
	}
	if len(req.Items) == 0 {
		return errors.New("the sale must have at least one item")
	}

	for i, item := range req.Items {
		if item.ProductID == uuid.Nil {
			return fmt.Errorf("item[%d]: product_id is required", i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("item[%d]: quantity must be greater than zero", i)
		}

	}
	return nil
}

func ApplyUpdateSaleParams(
	req UpdateSaleParams,
	arg *db.UpdateSaleParams,
) {
	if req.PaymentMethod != "" {
		arg.PaymentMethod = req.PaymentMethod
	}

	if req.DiscountAmount != 0 {
		arg.DiscountAmount = pgconv.Float64ToPgNumeric(req.DiscountAmount)
	}

	if req.DueDays != 0 {
		arg.DueDays = pgconv.IntToPgInt4(int(req.DueDays))
	}

	if req.InstallmentsCount != 0 {
		arg.InstallmentsCount = req.InstallmentsCount
	}

	if req.Prohibited != 0 {
		arg.DownPayment = pgconv.Float64ToPgNumeric(req.Prohibited)
	}
}
