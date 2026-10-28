package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// DefaultReturnWindowDays é o prazo de troca quando a empresa não configurou
// return_window_days. Zero na configuração = sem prazo.
const DefaultReturnWindowDays = 30

// MaxReturnWindowDays limita o prazo configurável (um ano)
const MaxReturnWindowDays = 365

const (
	KindReturn   = "return"
	KindExchange = "exchange"

	DestinationCredit = "credit"
	DestinationRefund = "refund"

	// Forma de pagamento da venda de troca quando o valor devolvido cobre tudo
	PaymentStoreCredit = "store_credit"
)

// RefundMethods são as formas de devolver dinheiro ou receber a diferença da troca
var RefundMethods = map[string]bool{
	"cash":          true,
	"pix":           true,
	"credit_card":   true,
	"debit_card":    true,
	"bank_transfer": true,
	"other":         true,
}

var (
	ErrSaleNotFound           = errors.New("venda não encontrada")
	ErrReturnNotFound         = errors.New("devolução não encontrada")
	ErrSaleCanceled           = errors.New("venda cancelada não pode ter itens devolvidos")
	ErrInvalidReturn          = errors.New("dados inválidos para a devolução")
	ErrQuantityExceeded       = errors.New("quantidade maior que a disponível para devolução")
	ErrOutOfWindow            = errors.New("venda fora do prazo de troca: só um administrador pode fazer a devolução")
	ErrOverrideReasonRequired = errors.New("venda fora do prazo de troca: informe o motivo da exceção")
	ErrCustomerRequired       = errors.New("para gerar vale-troca, informe o cliente")
	ErrCustomerNotFound       = errors.New("cliente não encontrado")
	ErrProductNotFound        = errors.New("produto não encontrado")
	ErrInsufficientStock      = errors.New("estoque insuficiente")
)

type ReturnItemRequest struct {
	SaleItemID uuid.UUID `json:"sale_item_id" binding:"required"`
	Quantity   int32     `json:"quantity" binding:"required,gt=0"`
	// false = peça avariada, não volta ao estoque. Omitido = volta.
	Restock *bool `json:"restock"`
}

type NewItemRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int32     `json:"quantity" binding:"required,gt=0"`
}

type CreateReturnRequest struct {
	SaleID   uuid.UUID           `json:"sale_id" binding:"required"`
	Kind     string              `json:"kind" binding:"required,oneof=return exchange"`
	Items    []ReturnItemRequest `json:"items" binding:"required,min=1,dive"`
	NewItems []NewItemRequest    `json:"new_items" binding:"omitempty,dive"`
	// O que fazer com o valor que sobra: vale-troca ou dinheiro de volta
	Destination  string `json:"destination" binding:"omitempty,oneof=credit refund"`
	RefundMethod string `json:"refund_method"`
	// Cliente do vale-troca quando a venda não tem cliente
	CustomerID *uuid.UUID `json:"customer_id"`
	// Forma de pagamento da diferença quando os itens novos valem mais
	DifferenceMethod string `json:"difference_method"`
	// Dinheiro entregue para pagar a diferença (calcula o troco)
	AmountReceived *float64 `json:"amount_received"`
	Reason         string   `json:"reason" binding:"required,oneof=size defect regret wrong_item other"`
	Notes          string   `json:"notes" binding:"max=500"`
	OverrideReason string   `json:"override_reason" binding:"max=500"`
}

type ListReturnsParams struct {
	Page      int32  `form:"page" binding:"omitempty,gte=1"`
	PerPage   int32  `form:"per_page" binding:"omitempty,gte=1,lte=100"`
	Search    string `form:"search" binding:"max=100"`
	Kind      string `form:"kind" binding:"omitempty,oneof=return exchange"`
	StartDate string `form:"start_date" binding:"omitempty,datetime=2006-01-02"`
	EndDate   string `form:"end_date" binding:"omitempty,datetime=2006-01-02"`
}

type SearchSalesParams struct {
	Page    int32  `form:"page" binding:"omitempty,gte=1"`
	PerPage int32  `form:"per_page" binding:"omitempty,gte=1,lte=50"`
	Search  string `form:"search" binding:"max=100"`
}

type SaleSearchItem struct {
	ID            uuid.UUID `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TotalAmount   float64   `json:"total_amount"`
	PaymentMethod string    `json:"payment_method"`
	CustomerName  string    `json:"customer_name"`
	ItemsCount    int32     `json:"items_count"`
	ReturnedCount int32     `json:"returned_count"`
	Products      string    `json:"products"`
}

type SaleSearchResponse struct {
	Data    []SaleSearchItem `json:"data"`
	Page    int32            `json:"page"`
	PerPage int32            `json:"per_page"`
	Total   int64            `json:"total"`
}

type ReturnableItem struct {
	SaleItemID  uuid.UUID `json:"sale_item_id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	SellInBulk  bool      `json:"sell_in_bulk"`
	Quantity    int32     `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	// Valor pago por unidade, já com o desconto da venda
	UnitAmount       float64 `json:"unit_amount"`
	ReturnedQuantity int32   `json:"returned_quantity"`
	Available        int32   `json:"available"`
	// Valor pago pela linha e o que dela já foi devolvido: a tela calcula o
	// valor da devolução com a mesma regra do servidor (LineReturnAmount)
	NetTotal      float64 `json:"net_total"`
	ReturnedTotal float64 `json:"returned_total"`
}

type PreviousReturn struct {
	ID            uuid.UUID `json:"id"`
	Number        int32     `json:"number"`
	Kind          string    `json:"kind"`
	ReturnedTotal float64   `json:"returned_total"`
	CreatedAt     time.Time `json:"created_at"`
}

// ReturnableSale é o que a tela precisa para montar a devolução de uma venda
type ReturnableSale struct {
	SaleID         uuid.UUID  `json:"sale_id"`
	CustomerID     *uuid.UUID `json:"customer_id"`
	CustomerName   string     `json:"customer_name"`
	CustomerCredit float64    `json:"customer_credit"`
	PaymentMethod  string     `json:"payment_method"`
	Status         string     `json:"status"`
	Subtotal       float64    `json:"subtotal"`
	DiscountAmount float64    `json:"discount_amount"`
	TotalAmount    float64    `json:"total_amount"`
	CreatedAt      time.Time  `json:"created_at"`
	Canceled       bool       `json:"canceled"`
	// Saldo das parcelas em aberto: a devolução abate daqui antes de gerar crédito
	OpenDebt float64 `json:"open_debt"`
	// Prazo de troca: WindowDays = 0 é sem prazo
	WindowDays   int        `json:"window_days"`
	Deadline     *time.Time `json:"deadline"`
	WithinWindow bool       `json:"within_window"`
	CanOverride  bool       `json:"can_override"`

	Items           []ReturnableItem `json:"items"`
	PreviousReturns []PreviousReturn `json:"previous_returns"`
}

type ReturnListItem struct {
	ID             uuid.UUID  `json:"id"`
	Number         int32      `json:"number"`
	Kind           string     `json:"kind"`
	SaleID         uuid.UUID  `json:"sale_id"`
	ExchangeSaleID *uuid.UUID `json:"exchange_sale_id"`
	CustomerID     *uuid.UUID `json:"customer_id"`
	CustomerName   string     `json:"customer_name"`
	ReturnedTotal  float64    `json:"returned_total"`
	ExchangeTotal  float64    `json:"exchange_total"`
	DebtReduction  float64    `json:"debt_reduction"`
	CreditAmount   float64    `json:"credit_amount"`
	RefundAmount   float64    `json:"refund_amount"`
	RefundMethod   *string    `json:"refund_method"`
	DifferencePaid float64    `json:"difference_paid"`
	Reason         string     `json:"reason"`
	OutOfWindow    bool       `json:"out_of_window"`
	ItemsCount     int32      `json:"items_count"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedByName  string     `json:"created_by_name"`
}

type ListReturnsResponse struct {
	Data    []ReturnListItem `json:"data"`
	Page    int32            `json:"page"`
	PerPage int32            `json:"per_page"`
	Total   int64            `json:"total"`
}

type ReturnedItem struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Quantity    int32     `json:"quantity"`
	UnitAmount  float64   `json:"unit_amount"`
	Total       float64   `json:"total"`
	Restock     bool      `json:"restock"`
}

type ExchangeItem struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Quantity    int32     `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Total       float64   `json:"total"`
}

type ReturnDetail struct {
	ReturnListItem
	ExchangeCredit   float64        `json:"exchange_credit"`
	DifferenceMethod *string        `json:"difference_method"`
	Notes            *string        `json:"notes"`
	OverrideReason   *string        `json:"override_reason"`
	SaleCreatedAt    time.Time      `json:"sale_created_at"`
	SaleTotal        float64        `json:"sale_total"`
	Items            []ReturnedItem `json:"items"`
	NewItems         []ExchangeItem `json:"new_items"`
}

type CreditMovement struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Amount        float64    `json:"amount"`
	BalanceAfter  float64    `json:"balance_after"`
	SaleReturnID  *uuid.UUID `json:"sale_return_id"`
	ReturnNumber  *int32     `json:"return_number"`
	SaleID        *uuid.UUID `json:"sale_id"`
	CreatedAt     time.Time  `json:"created_at"`
	CreatedByName string     `json:"created_by_name"`
}

type CustomerCredit struct {
	CustomerID   uuid.UUID        `json:"customer_id"`
	CustomerName string           `json:"customer_name"`
	StoreCredit  float64          `json:"store_credit"`
	Movements    []CreditMovement `json:"movements"`
}
