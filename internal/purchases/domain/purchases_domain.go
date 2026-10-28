package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPurchase        = errors.New("compra inválida")
	ErrOrderNotFound          = errors.New("pedido de compra não encontrado")
	ErrReceiptNotFound        = errors.New("recebimento não encontrado")
	ErrVendorNotFound         = errors.New("fornecedor não encontrado")
	ErrVendorInactive         = errors.New("fornecedor inativo")
	ErrProductNotFound        = errors.New("produto não encontrado")
	ErrOrderNotEditable       = errors.New("o pedido só pode ser alterado enquanto está em rascunho ou enviado, sem recebimentos")
	ErrOrderStatus            = errors.New("operação não permitida no status atual do pedido")
	ErrQuantityExceeded       = errors.New("quantidade recebida maior que a pendente no pedido")
	ErrInstallmentsMismatch   = errors.New("a soma das parcelas deve ser igual ao total do recebimento")
	ErrReceiptAlreadyReversed = errors.New("este recebimento já foi estornado")
	ErrReceiptHasPaidBills    = errors.New("não é possível estornar: há parcelas deste recebimento já pagas")
	ErrInsufficientStock      = errors.New("não é possível estornar: o estoque atual não cobre a devolução")
	ErrVendorWithoutEmail     = errors.New("o fornecedor não tem um e-mail válido cadastrado; atualize o cadastro em Configurações > Fornecedores")
	ErrEmailUnavailable       = errors.New("o envio de e-mail está indisponível no momento; tente novamente em instantes")
)

// Tipo do e-mail do pedido ao fornecedor
const (
	EmailKindSent    = "sent"    // pedido enviado
	EmailKindUpdated = "updated" // pedido alterado depois de enviado
	EmailKindResend  = "resend"  // reenvio manual
)

type EmailOrderRequest struct {
	Kind string `json:"kind" binding:"required,oneof=sent updated resend"`
}

type OrderEmail struct {
	Kind       string    `json:"kind"`
	Recipient  string    `json:"recipient"`
	SentByName string    `json:"sent_by_name"`
	CreatedAt  time.Time `json:"created_at"`
}

// Status do pedido
const (
	StatusDraft    = "draft"
	StatusSent     = "sent"
	StatusPartial  = "partial"
	StatusReceived = "received"
	StatusClosed   = "closed"
	StatusCanceled = "canceled"
)

// ─── Requests ────────────────────────────────────────────────────────────────

type OrderItemRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int32     `json:"quantity" binding:"required,gt=0"`
	UnitCost  float64   `json:"unit_cost" binding:"gte=0"`
}

type SaveOrderRequest struct {
	VendorID     uuid.UUID          `json:"vendor_id" binding:"required"`
	ExpectedDate string             `json:"expected_date" binding:"omitempty,datetime=2006-01-02"`
	Notes        string             `json:"notes" binding:"max=1000"`
	Items        []OrderItemRequest `json:"items" binding:"required,min=1,max=300,dive"`
	// Na criação: true já marca o pedido como enviado ao fornecedor
	Send bool `json:"send"`
}

type ListOrdersParams struct {
	Status   string    `form:"status" binding:"omitempty,oneof=draft sent partial received closed canceled"`
	VendorID uuid.UUID `form:"vendor_id"`
	Search   string    `form:"search" binding:"max=100"`
	Page     int32     `form:"page" binding:"omitempty,gte=1"`
	PerPage  int32     `form:"per_page" binding:"omitempty,gte=1,lte=100"`
}

type CloseOrderRequest struct {
	Reason string `json:"reason" binding:"max=500"`
}

type ReceiptItemRequest struct {
	OrderItemID uuid.UUID `json:"order_item_id" binding:"required"`
	Quantity    int32     `json:"quantity" binding:"required,gt=0"`
	// Preço unitário da nota (pode diferir do pedido)
	UnitCost float64 `json:"unit_cost" binding:"gte=0"`
	// Novo preço de venda; omitido mantém o atual
	SalePrice *float64 `json:"sale_price" binding:"omitempty,gte=0"`
}

type InstallmentRequest struct {
	DueDate string  `json:"due_date" binding:"required,datetime=2006-01-02"`
	Amount  float64 `json:"amount" binding:"required,gt=0"`
}

type ReceiveRequest struct {
	ReceivedAt      string               `json:"received_at" binding:"omitempty,datetime=2006-01-02"`
	InvoiceNumber   string               `json:"invoice_number" binding:"max=20"`
	InvoiceSeries   string               `json:"invoice_series" binding:"max=5"`
	InvoiceKey      string               `json:"invoice_key" binding:"omitempty,len=44,numeric"`
	Freight         float64              `json:"freight" binding:"gte=0"`
	OtherExpenses   float64              `json:"other_expenses" binding:"gte=0"`
	Discount        float64              `json:"discount" binding:"gte=0"`
	Notes           string               `json:"notes" binding:"max=1000"`
	Items           []ReceiptItemRequest `json:"items" binding:"required,min=1,dive"`
	Installments    []InstallmentRequest `json:"installments" binding:"max=60,dive"`
	BillCategoryID  *uuid.UUID           `json:"bill_category_id"`
	PaymentMethodID *uuid.UUID           `json:"payment_method_id"`
}

type ReverseReceiptRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=500"`
}

// ─── Responses ───────────────────────────────────────────────────────────────

type OrderListItem struct {
	ID               uuid.UUID `json:"id"`
	Number           int32     `json:"number"`
	VendorName       string    `json:"vendor_name"`
	Status           string    `json:"status"`
	ExpectedDate     *string   `json:"expected_date"`
	CreatedAt        time.Time `json:"created_at"`
	ItemsCount       int32     `json:"items_count"`
	Total            float64   `json:"total"`
	Quantity         int32     `json:"quantity"`
	ReceivedQuantity int32     `json:"received_quantity"`
	ReceivedTotal    float64   `json:"received_total"`
	Late             bool      `json:"late"`
}

type OrdersOverview struct {
	PendingTotal      float64          `json:"pending_total"`
	ReceivedThisMonth float64          `json:"received_this_month"`
	LateOrders        int32            `json:"late_orders"`
	CountByStatus     map[string]int32 `json:"count_by_status"`
}

type ListOrdersResponse struct {
	Data       []OrderListItem `json:"data"`
	Page       int32           `json:"page"`
	PerPage    int32           `json:"per_page"`
	TotalRows  int32           `json:"total_rows"`
	TotalPages int32           `json:"total_pages"`
	Overview   OrdersOverview  `json:"overview"`
}

type OrderItem struct {
	ID               uuid.UUID `json:"id"`
	ProductID        uuid.UUID `json:"product_id"`
	ProductName      string    `json:"product_name"`
	ModelName        string    `json:"model_name"`
	Color            string    `json:"color"`
	Size             string    `json:"size"`
	Barcode          string    `json:"barcode"`
	Unit             string    `json:"unit"`
	SellInBulk       bool      `json:"sell_in_bulk"`
	StockQuantity    int32     `json:"stock_quantity"`
	CurrentCost      float64   `json:"current_cost"`
	SalePrice        float64   `json:"sale_price"`
	Quantity         int32     `json:"quantity"`
	UnitCost         float64   `json:"unit_cost"`
	Total            float64   `json:"total"`
	ReceivedQuantity int32     `json:"received_quantity"`
	PendingQuantity  int32     `json:"pending_quantity"`
}

type ReceiptItem struct {
	OrderItemID    uuid.UUID `json:"order_item_id"`
	ProductID      uuid.UUID `json:"product_id"`
	ProductName    string    `json:"product_name"`
	Quantity       int32     `json:"quantity"`
	UnitCost       float64   `json:"unit_cost"`
	LandedUnitCost float64   `json:"landed_unit_cost"`
	PreviousCost   float64   `json:"previous_cost"`
	NewCost        float64   `json:"new_cost"`
}

type ReceiptBill struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	DueDate     string    `json:"due_date"`
	Status      string    `json:"status"`
	AmountPaid  float64   `json:"amount_paid"`
	PaymentDate *string   `json:"payment_date"`
}

type Receipt struct {
	ID             uuid.UUID     `json:"id"`
	Status         string        `json:"status"`
	ReceivedAt     time.Time     `json:"received_at"`
	InvoiceNumber  string        `json:"invoice_number"`
	InvoiceSeries  string        `json:"invoice_series"`
	InvoiceKey     string        `json:"invoice_key"`
	ItemsTotal     float64       `json:"items_total"`
	Freight        float64       `json:"freight"`
	OtherExpenses  float64       `json:"other_expenses"`
	Discount       float64       `json:"discount"`
	Total          float64       `json:"total"`
	Notes          *string       `json:"notes"`
	CreatedByName  string        `json:"created_by_name"`
	ReversedAt     *time.Time    `json:"reversed_at"`
	ReversedByName string        `json:"reversed_by_name"`
	ReversalReason *string       `json:"reversal_reason"`
	Items          []ReceiptItem `json:"items"`
	Bills          []ReceiptBill `json:"bills"`
}

type OrderDetail struct {
	ID               uuid.UUID    `json:"id"`
	Number           int32        `json:"number"`
	VendorID         uuid.UUID    `json:"vendor_id"`
	VendorName       string       `json:"vendor_name"`
	Status           string       `json:"status"`
	ExpectedDate     *string      `json:"expected_date"`
	Notes            *string      `json:"notes"`
	CreatedByName    string       `json:"created_by_name"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	SentAt           *time.Time   `json:"sent_at"`
	ClosedAt         *time.Time   `json:"closed_at"`
	CloseReason      *string      `json:"close_reason"`
	Total            float64      `json:"total"`
	ReceivedTotal    float64      `json:"received_total"`
	Quantity         int32        `json:"quantity"`
	ReceivedQuantity int32        `json:"received_quantity"`
	Items            []OrderItem  `json:"items"`
	Receipts         []Receipt    `json:"receipts"`
	VendorEmail      string       `json:"vendor_email"`
	Emails           []OrderEmail `json:"emails"`
	// Ações liberadas no status atual, para o front não duplicar a regra
	CanEdit    bool `json:"can_edit"`
	CanSend    bool `json:"can_send"`
	CanReceive bool `json:"can_receive"`
	CanClose   bool `json:"can_close"`
	CanCancel  bool `json:"can_cancel"`
	CanEmail   bool `json:"can_email"`
}
