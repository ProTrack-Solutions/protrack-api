package service

import (
	"context"
	"encoding/json"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

// recordEvent grava um evento na trilha de auditoria (fiscal_invoice_events).
// Falhas aqui são "best-effort" — não interrompem o fluxo principal de
// emissão/cancelamento, apenas deixam de registrar o evento.
func (s *Service) recordEvent(ctx context.Context, invoiceID pgtype.UUID, eventType string, payload map[string]any) {
	var payloadBytes []byte
	if payload != nil {
		payloadBytes, _ = json.Marshal(payload)
	}
	_, _ = s.repo.CreateFiscalInvoiceEvent(ctx, db.CreateFiscalInvoiceEventParams{
		FiscalInvoiceID: invoiceID,
		EventType:       eventType,
		Payload:         payloadBytes,
	})
}
