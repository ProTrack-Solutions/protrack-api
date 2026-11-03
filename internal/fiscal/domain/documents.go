package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSaleNotFound           = errors.New("venda não encontrada")
	ErrSaleCanceled           = errors.New("a venda foi cancelada; não há nota a emitir")
	ErrDocumentNotFound       = errors.New("nota fiscal não encontrada")
	ErrDocumentNotPrintable   = errors.New("o DANFC-e só existe para nota autorizada")
	ErrDocumentNotCancellable = errors.New("só nota autorizada pode ser cancelada")
	ErrInvalidCancelReason    = errors.New("a justificativa do cancelamento deve ter de 15 a 255 caracteres")
)

// SaleBlockedError: a NFC-e impede cancelar/alterar a venda agora (o texto vai ao usuário).
type SaleBlockedError struct{ Reason string }

func (e *SaleBlockedError) Error() string { return e.Reason }

// Situações da nota no ERP: as da API fiscal + FAILED (não virou nota).
const (
	StatusPending     = "PENDING"
	StatusProcessing  = "PROCESSING"
	StatusAuthorized  = "AUTHORIZED"
	StatusRejected    = "REJECTED"
	StatusError       = "ERROR"
	StatusCancelled   = "CANCELLED"
	StatusContingency = "CONTINGENCY"
	StatusSuperseded  = "SUPERSEDED"
	StatusFailed      = "FAILED"
)

// FiscalDocument é uma tentativa de NFC-e da venda.
type FiscalDocument struct {
	ID           uuid.UUID  `json:"id"`
	SaleID       uuid.UUID  `json:"sale_id"`
	Environment  string     `json:"environment"`
	Attempt      int32      `json:"attempt"`
	Status       string     `json:"status"`
	Number       *int32     `json:"number"`
	AccessKey    *string    `json:"access_key"`
	Protocol     *string    `json:"protocol"`
	StatusCode   *string    `json:"status_code"`
	RejectReason *string    `json:"reject_reason"`
	ErrorMessage *string    `json:"error_message"`
	Contingency  bool       `json:"contingency"`
	AuthorizedAt *time.Time `json:"authorized_at"`
	CancelReason *string    `json:"cancel_reason"`
	CancelledAt  *time.Time `json:"cancelled_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Printable: autorizada ou emitida em contingência (já vale e pode imprimir).
func (d FiscalDocument) Printable() bool {
	return d.Status == StatusAuthorized || d.Status == StatusContingency
}

type FiscalDocumentListItem struct {
	FiscalDocument
	SaleTotal    float64   `json:"sale_total"`
	SaleAt       time.Time `json:"sale_at"`
	CustomerName *string   `json:"customer_name"`
}

type FiscalDocumentList struct {
	Documents []FiscalDocumentListItem `json:"documents"`
	Total     int64                    `json:"total"`
	Page      int                      `json:"page"`
	PerPage   int                      `json:"per_page"`
}

type CancelInput struct {
	Reason string `json:"reason"`
}
