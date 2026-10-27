package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCashSessionAlreadyOpen = errors.New("você já tem um caixa aberto")
	ErrNoOpenCashSession      = errors.New("nenhum caixa aberto: abra o caixa antes de continuar")
	ErrCashSessionNotFound    = errors.New("caixa não encontrado")
	ErrInsufficientCash       = errors.New("valor maior que o dinheiro disponível no caixa")
	ErrCashRegisterRequired   = errors.New("abra o caixa para registrar vendas e recebimentos")
)

const (
	MovementWithdrawal = "withdrawal" // sangria
	MovementSupply     = "supply"     // suprimento
)

type OpenCashSessionRequest struct {
	OpeningAmount float64 `json:"opening_amount" binding:"gte=0"`
	Notes         string  `json:"notes" binding:"max=500"`
}

type CreateMovementRequest struct {
	Type   string  `json:"type" binding:"required,oneof=withdrawal supply"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Reason string  `json:"reason" binding:"max=500"`
}

type CloseCashSessionRequest struct {
	CountedCash float64 `json:"counted_cash" binding:"gte=0"`
	Notes       string  `json:"notes" binding:"max=500"`
}

type ListCashSessionsParams struct {
	Page    int32 `form:"page" binding:"omitempty,gte=1"`
	PerPage int32 `form:"per_page" binding:"omitempty,gte=1,lte=100"`
}

type CashSession struct {
	ID            uuid.UUID  `json:"id"`
	CompanyID     uuid.UUID  `json:"-"`
	UserID        uuid.UUID  `json:"user_id"`
	UserName      string     `json:"user_name,omitempty"`
	Status        string     `json:"status"`
	OpeningAmount float64    `json:"opening_amount"`
	OpeningNotes  *string    `json:"opening_notes"`
	OpenedAt      time.Time  `json:"opened_at"`
	ClosedAt      *time.Time `json:"closed_at"`
	ClosingNotes  *string    `json:"closing_notes"`
	// Retrato gravado no fechamento; nulos enquanto o caixa está aberto
	SalesTotal    *float64 `json:"sales_total"`
	SalesCount    *int32   `json:"sales_count"`
	ReceivedTotal *float64 `json:"received_total"`
	ExpectedCash  *float64 `json:"expected_cash"`
	CountedCash   *float64 `json:"counted_cash"`
	Difference    *float64 `json:"difference"`
}

type CashMovement struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Amount    float64   `json:"amount"`
	Reason    *string   `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// MethodSummary junta vendas e recebimentos de parcelas de uma forma de pagamento.
// Received é o que de fato entrou: na venda a prazo, só a entrada.
type MethodSummary struct {
	PaymentMethod string  `json:"payment_method"`
	SalesCount    int32   `json:"sales_count"`
	SalesTotal    float64 `json:"sales_total"`
	ReceiptsCount int32   `json:"receipts_count"`
	ReceiptsTotal float64 `json:"receipts_total"`
	Received      float64 `json:"received"`
}

type CashSummary struct {
	SalesCount    int32   `json:"sales_count"`
	SalesTotal    float64 `json:"sales_total"`
	ReceiptsCount int32   `json:"receipts_count"`
	ReceiptsTotal float64 `json:"receipts_total"`
	// Tudo que entrou no turno, em qualquer forma de pagamento
	ReceivedTotal float64 `json:"received_total"`
	// Dinheiro em espécie que entrou (vendas, entradas de vendas a prazo e parcelas)
	CashReceived     float64 `json:"cash_received"`
	SuppliesTotal    float64 `json:"supplies_total"`
	WithdrawalsTotal float64 `json:"withdrawals_total"`
	ExpectedCash     float64 `json:"expected_cash"`
	// Troco devolvido nas vendas do turno (já descontado do dinheiro recebido)
	ChangeCount int32           `json:"change_count"`
	ChangeTotal float64         `json:"change_total"`
	ByMethod    []MethodSummary `json:"by_method"`
}

type CashSessionDetail struct {
	Session   CashSession    `json:"session"`
	Summary   CashSummary    `json:"summary"`
	Movements []CashMovement `json:"movements"`
}

type ListCashSessionsResponse struct {
	Data       []CashSession `json:"data"`
	Page       int32         `json:"page"`
	PerPage    int32         `json:"per_page"`
	TotalRows  int64         `json:"total_rows"`
	TotalPages int64         `json:"total_pages"`
}
