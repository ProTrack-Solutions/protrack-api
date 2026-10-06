package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/validate"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"

	"github.com/google/uuid"
)

type PaginationParams struct {
	globalDomain.PaginationParams
	SaleStatus       string `form:"saleStatus" validate:"omitempty,oneof=pending paid overdue scheduled canceled partial"`
	PaymentMethod    string `form:"paymentMethod" validate:"omitempty,oneof=cash credit_card debit_card pix bank_transfer installments other"`
	PaymentStartDate string `form:"paymentStartDate" validate:"omitempty,datetime=2006-01-02"`
	PaymentEndDate   string `form:"paymentEndDate" validate:"omitempty,datetime=2006-01-02"`
	SaleStartDate    string `form:"saleStartDate" validate:"omitempty,datetime=2006-01-02"`
	SaleEndDate      string `form:"saleEndDate" validate:"omitempty,datetime=2006-01-02"`
	SortBy           string `form:"sortBy" validate:"omitempty,oneof=sale_at created_at"`
	OrderBy          string `form:"orderBy" validate:"omitempty,oneof=asc desc"`
}

type CreateSaleRequest struct {
	CustomerID        uuid.UUID               `json:"customer_id"`
	BuyerDocument     string                  `json:"buyer_document"`
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

var (
	ErrSaleNotFound        = errors.New("venda não encontrada")
	ErrSaleAlreadyCanceled = errors.New("venda já foi cancelada")
	ErrSaleUpdateExpired   = errors.New("a venda só pode ser alterada até 2 horas depois de ser realizada")
	ErrSaleHasPayments     = errors.New("não é possível alterar uma venda que já possui parcelas pagas")
	ErrInvalidSaleUpdate   = errors.New("dados inválidos para atualizar a venda")

	ErrInvalidSale          = errors.New("dados inválidos para a venda")
	ErrSaleCustomerNotFound = errors.New("cliente não encontrado")
	ErrSaleProductNotFound  = errors.New("produto não encontrado")
	ErrInsufficientStock    = errors.New("estoque insuficiente")
)

// MaxInstallments é o limite de parcelas de uma venda a prazo (o mesmo oferecido na tela).
const MaxInstallments = 24

// SaleUpdateWindow é o tempo, a partir da criação, em que a venda ainda pode ser alterada.
const SaleUpdateWindow = 2 * time.Hour

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

// UpdateSaleParams altera as condições de pagamento de uma venda.
// DiscountAmount é a porcentagem de desconto (0–100), igual ao CreateSaleRequest.
// Campos nil/zero mantêm o valor atual da venda.
type UpdateSaleParams struct {
	DiscountAmount    *float64            `json:"discount_amount"`
	DueDays           int32               `json:"due_days"`
	PaymentMethod     enums.PaymentMethod `json:"payment_method"`
	InstallmentsCount int32               `json:"installments_count"`
	Prohibited        *float64            `json:"prohibited"`
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

type OverdueSalesResult struct {
	WhatsAppEvents     []events.WhatsApp
	AnnouncementEvents []events.Announcement
}

func ValidateCreateSaleRequest(req CreateSaleRequest) error {
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidSale, fmt.Sprintf(format, args...))
	}

	if !req.PaymentMethod.IsValid() {
		return invalid("forma de pagamento inválida")
	}

	// Venda a prazo (parcelada) exige cliente cadastrado, pois gera saldo devedor
	// e contas a receber vinculados a ele. Venda avulsa (paga na hora) não exige.
	if req.PaymentMethod == enums.PaymentMethodInstallments {
		if req.CustomerID == uuid.Nil {
			return invalid("venda a prazo exige um cliente")
		}
		if req.InstallmentsCount < 1 || req.InstallmentsCount > MaxInstallments {
			return invalid("a quantidade de parcelas deve estar entre 1 e %d", MaxInstallments)
		}
		if req.DueDays < 1 || req.DueDays > 31 {
			return invalid("informe um dia de vencimento entre 1 e 31")
		}
		if req.Prohibited < 0 {
			return invalid("a entrada não pode ser negativa")
		}
	}

	if req.BuyerDocument != "" {
		if _, err := validate.ValidateDocument(req.BuyerDocument); err != nil {
			return invalid("documento do comprador inválido: %v", err)
		}
	}

	if req.DiscountAmount < 0 || req.DiscountAmount > 100 {
		return invalid("o desconto deve estar entre 0%% e 100%%")
	}

	if len(req.Items) == 0 {
		return invalid("adicione pelo menos um produto")
	}

	seen := make(map[uuid.UUID]bool, len(req.Items))
	for i, item := range req.Items {
		if item.ProductID == uuid.Nil {
			return invalid("selecione o produto do item %d", i+1)
		}
		if item.Quantity <= 0 {
			return invalid("a quantidade do item %d deve ser maior que zero", i+1)
		}
		if seen[item.ProductID] {
			return invalid("o mesmo produto foi adicionado mais de uma vez (item %d)", i+1)
		}
		seen[item.ProductID] = true
	}

	return nil
}
