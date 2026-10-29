package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCashSessionAlreadyOpen = errors.New("este caixa já está aberto")
	ErrNoOpenCashSession      = errors.New("nenhum caixa aberto: abra o caixa antes de continuar")
	ErrCashSessionNotFound    = errors.New("caixa não encontrado")
	ErrInsufficientCash       = errors.New("valor maior que o dinheiro disponível no caixa")
	ErrCashRegisterRequired   = errors.New("abra o caixa para registrar vendas e recebimentos")
	ErrCashRegisterNotFound   = errors.New("caixa não encontrado ou desativado")
	// Mais de um caixa ativo e o computador não disse em qual está
	ErrCashRegisterNotSelected = errors.New("escolha em qual caixa este computador trabalha")
	ErrCashRegisterInUse       = errors.New("feche o turno deste caixa antes de desativá-lo")

	ErrTransferNotFound        = errors.New("transferência não encontrada")
	ErrTransferAlreadyResolved = errors.New("esta transferência já foi aceita, recusada ou cancelada")
	ErrTransferSameRegister    = errors.New("escolha outro caixa para receber o troco")
	ErrTransferTargetClosed    = errors.New("o caixa de destino está fechado")
	// Aceitar/recusar só no caixa de destino; cancelar só no de origem
	ErrTransferWrongRegister = errors.New("esta transferência é de outro caixa")
	ErrPendingTransfers      = errors.New("há transferência de troco pendente neste caixa: aceite, recuse ou cancele antes de fechar")
)

const (
	MovementWithdrawal = "withdrawal" // sangria
	MovementSupply     = "supply"     // suprimento
)

// RegisterHeader identifica o caixa físico em que o computador trabalha
const RegisterHeader = "CashRegisterId"

type registerKey struct{}

// WithRegisterID guarda no contexto o caixa escolhido pelo computador (header
// CashRegisterId), para venda, recebimento e devolução saberem em qual turno entram.
func WithRegisterID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, registerKey{}, id)
}

// RegisterIDFrom devolve o caixa escolhido, ou uuid.Nil quando o header não veio
func RegisterIDFrom(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(registerKey{}).(uuid.UUID)
	return id
}

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

type CreateTransferRequest struct {
	ToRegisterID uuid.UUID `json:"to_register_id" binding:"required"`
	Amount       float64   `json:"amount" binding:"required,gt=0"`
	Reason       string    `json:"reason" binding:"max=500"`
}

// PendingTransfer transferência de troco aguardando o caixa de destino aceitar
type PendingTransfer struct {
	ID               uuid.UUID `json:"id"`
	Amount           float64   `json:"amount"`
	Reason           *string   `json:"reason"`
	FromRegisterName string    `json:"from_register_name"`
	ToRegisterName   string    `json:"to_register_name"`
	CreatedByName    string    `json:"created_by_name"`
	CreatedAt        time.Time `json:"created_at"`
}

type SaveCashRegisterRequest struct {
	Name   string `json:"name" binding:"required,min=1,max=60"`
	Active *bool  `json:"active"`
}

type ListCashSessionsParams struct {
	Page       int32     `form:"page" binding:"omitempty,gte=1"`
	PerPage    int32     `form:"per_page" binding:"omitempty,gte=1,lte=100"`
	RegisterID uuid.UUID `form:"register_id"`
}

// OpenRegisterSession turno aberto de um caixa, para a lista de caixas
type OpenRegisterSession struct {
	ID            uuid.UUID `json:"id"`
	OpenedAt      time.Time `json:"opened_at"`
	OpeningAmount float64   `json:"opening_amount"`
	OpenedByName  string    `json:"opened_by_name"`
}

// LastClosing último fechamento do caixa: referência para o troco da próxima abertura
type LastClosing struct {
	ClosedAt     time.Time `json:"closed_at"`
	ClosedByName string    `json:"closed_by_name"`
	CountedCash  float64   `json:"counted_cash"`
}

type CashRegister struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Active bool      `json:"active"`
	// Nulo quando o caixa está fechado
	OpenSession *OpenRegisterSession `json:"open_session"`
	// Nulo quando o caixa nunca foi fechado
	LastClosing *LastClosing `json:"last_closing"`
}

type CashSession struct {
	ID           uuid.UUID `json:"id"`
	CompanyID    uuid.UUID `json:"-"`
	RegisterID   uuid.UUID `json:"register_id"`
	RegisterName string    `json:"register_name,omitempty"`
	// Quem abriu o turno
	UserID        uuid.UUID  `json:"user_id"`
	UserName      string     `json:"user_name,omitempty"`
	ClosedByName  string     `json:"closed_by_name,omitempty"`
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
	ID            uuid.UUID `json:"id"`
	Type          string    `json:"type"`
	Amount        float64   `json:"amount"`
	Reason        *string   `json:"reason"`
	CreatedByName string    `json:"created_by_name"`
	// Preenchido quando o movimento veio de uma transferência entre caixas
	TransferLabel string    `json:"transfer_label,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
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
	// Devolvido ao cliente em trocas/devoluções (já descontado de Received)
	RefundsCount int32   `json:"refunds_count"`
	RefundsTotal float64 `json:"refunds_total"`
}

// OperatorSummary quanto cada operador vendeu e recebeu de parcelas no turno
type OperatorSummary struct {
	UserID        uuid.UUID `json:"user_id"`
	UserName      string    `json:"user_name"`
	SalesCount    int32     `json:"sales_count"`
	SalesTotal    float64   `json:"sales_total"`
	ReceiptsTotal float64   `json:"receipts_total"`
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
	// Dinheiro devolvido a clientes em trocas/devoluções do turno
	RefundsCount int32   `json:"refunds_count"`
	RefundsTotal float64 `json:"refunds_total"`
	CashRefunds  float64 `json:"cash_refunds"`
	ExpectedCash float64 `json:"expected_cash"`
	// Troco devolvido nas vendas do turno (já descontado do dinheiro recebido)
	ChangeCount int32             `json:"change_count"`
	ChangeTotal float64           `json:"change_total"`
	ByMethod    []MethodSummary   `json:"by_method"`
	ByOperator  []OperatorSummary `json:"by_operator"`
}

type CashSessionDetail struct {
	Session   CashSession    `json:"session"`
	Summary   CashSummary    `json:"summary"`
	Movements []CashMovement `json:"movements"`
	// Transferências pendentes: chegando (aceitar/recusar) e saindo (cancelar)
	IncomingTransfers []PendingTransfer `json:"incoming_transfers"`
	OutgoingTransfers []PendingTransfer `json:"outgoing_transfers"`
}

type ListCashSessionsResponse struct {
	Data       []CashSession `json:"data"`
	Page       int32         `json:"page"`
	PerPage    int32         `json:"per_page"`
	TotalRows  int64         `json:"total_rows"`
	TotalPages int64         `json:"total_pages"`
}
