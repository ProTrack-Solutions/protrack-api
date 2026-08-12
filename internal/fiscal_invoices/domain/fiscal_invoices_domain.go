package domain

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Requests
// ---------------------------------------------------------------------------

// EmitFiscalInvoiceRequest é o payload recebido em POST /fiscal-invoices.
type EmitFiscalInvoiceRequest struct {
	SaleID uuid.UUID `json:"saleId" validate:"required"`
	Type   string    `json:"type" validate:"required,oneof=nfe nfce"`
}

// CancelFiscalInvoiceRequest é o payload recebido em
// POST /fiscal-invoices/:id/cancel.
type CancelFiscalInvoiceRequest struct {
	ID            uuid.UUID `json:"-"` // vem da URL, não do body
	CompanyID     uuid.UUID `json:"-"` // vem do contexto autenticado
	Justificativa string    `json:"justificativa" validate:"required"`
}

// ValidateCancelFiscalInvoiceRequest aplica a regra da SEFAZ para o campo
// xJust: mínimo de 15 caracteres.
func ValidateCancelFiscalInvoiceRequest(req CancelFiscalInvoiceRequest) error {
	if len(req.Justificativa) < 15 {
		return ErrInvalidJustificativa
	}
	return nil
}

// UploadCertificateRequest é o payload recebido em POST /company-certificates.
type UploadCertificateRequest struct {
	CompanyID uuid.UUID `json:"-"` // vem do contexto autenticado
	CertFile  []byte    `json:"-"` // vem do multipart form
	Password  string    `json:"password" validate:"required"`
}

// ---------------------------------------------------------------------------
// Validação de campos fiscais obrigatórios na emissão
// ---------------------------------------------------------------------------

// MissingFiscalFields carrega a lista legível de campos fiscais ausentes,
// usada para retornar um erro 422 claro em vez de uma rejeição genérica.
type MissingFiscalFields struct {
	CompanyFields []string
	ProductFields map[uuid.UUID][]string // productID -> campos faltando
}

func (m MissingFiscalFields) HasMissing() bool {
	return len(m.CompanyFields) > 0 || len(m.ProductFields) > 0
}

// ---------------------------------------------------------------------------
// Response
// ---------------------------------------------------------------------------

// FiscalInvoiceResponse é o DTO devolvido pelos endpoints de consulta.
// Type/Status ficam como interface{} — mesmo padrão de sales.Status neste
// repo, já que o sqlc gera enums nativos do Postgres como interface{} (sem
// overrides em sqlc.yaml). Dá para comparar direto contra string literal
// (ex: resp.Status == "authorized"), sem type assertion.
type FiscalInvoiceResponse struct {
	ID                   uuid.UUID   `json:"id"`
	CompanyID            uuid.UUID   `json:"companyId"`
	SaleID               uuid.UUID   `json:"saleId"`
	Type                 interface{} `json:"type"`
	Status               interface{} `json:"status"`
	ProviderInvoiceID    string      `json:"providerInvoiceId,omitempty"`
	ChaveAcesso          string      `json:"chaveAcesso,omitempty"`
	Numero               string      `json:"numero,omitempty"`
	Serie                string      `json:"serie,omitempty"`
	ProtocoloAutorizacao string      `json:"protocoloAutorizacao,omitempty"`
	XMLUrl               string      `json:"xmlUrl,omitempty"`
	DANFEUrl             string      `json:"danfeUrl,omitempty"`
	ErrorMessage         string      `json:"errorMessage,omitempty"`
	CancelledReason      string      `json:"cancelledReason,omitempty"`
	AuthorizedAt         time.Time   `json:"authorizedAt,omitempty"`
	CancelledAt          time.Time   `json:"cancelledAt,omitempty"`
	CreatedAt            time.Time   `json:"createdAt"`
}

// ToFiscalInvoiceResponse converte a row gerada pelo sqlc (com pgtype.*) para
// o DTO de resposta em tipos nativos, usando os helpers pgconv já existentes
// no repo.
func ToFiscalInvoiceResponse(row db.FiscalInvoice) FiscalInvoiceResponse {
	return FiscalInvoiceResponse{
		ID:                   pgconv.PgUUIDToUUID(row.ID),
		CompanyID:            pgconv.PgUUIDToUUID(row.CompanyID),
		SaleID:               pgconv.PgUUIDToUUID(row.SaleID),
		Type:                 row.Type,
		Status:               row.Status,
		ProviderInvoiceID:    pgconv.ParsePgTextToString(row.ProviderInvoiceID),
		ChaveAcesso:          pgconv.ParsePgTextToString(row.ChaveAcesso),
		Numero:               pgconv.ParsePgTextToString(row.Numero),
		Serie:                pgconv.ParsePgTextToString(row.Serie),
		ProtocoloAutorizacao: pgconv.ParsePgTextToString(row.ProtocoloAutorizacao),
		XMLUrl:               pgconv.ParsePgTextToString(row.XmlUrl),
		DANFEUrl:             pgconv.ParsePgTextToString(row.DanfeUrl),
		ErrorMessage:         pgconv.ParsePgTextToString(row.ErrorMessage),
		CancelledReason:      pgconv.ParsePgTextToString(row.CancelledReason),
		AuthorizedAt:         pgconv.PgTimestamptzToTime(row.AuthorizedAt),
		CancelledAt:          pgconv.PgTimestamptzToTime(row.CancelledAt),
		CreatedAt:            pgconv.PgTimestamptzToTime(row.CreatedAt),
	}
}
