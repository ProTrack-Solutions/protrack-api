package service

import (
	"context"
	"fmt"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	nfeemissionDomain "github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
)

// HandlePlugNotasWebhook processa a notificação assíncrona de status vinda
// do provedor fiscal. A autenticação (validação do segredo compartilhado no
// header) já deve ter sido feita no handler antes de chamar este método —
// mesmo padrão do SyncSubscriptionWebhook do Stripe neste repo.
func (s *Service) HandlePlugNotasWebhook(ctx context.Context, notification nfeemissionDomain.WebhookNotification) error {
	invoice, err := s.repo.GetFiscalInvoiceByProviderInvoiceID(ctx, pgconv.ParseStringToPgType(notification.ID))
	if err != nil {
		return fmt.Errorf("fiscal_invoices: documento fiscal não encontrado para provider_invoice_id=%s: %w", notification.ID, err)
	}

	s.recordEvent(ctx, invoice.ID, "webhook_received", map[string]any{
		"status": notification.Status,
	})

	return s.applyProviderStatus(ctx, invoice, providerStatusUpdate{
		Status:      notification.Status,
		ChaveAcesso: notification.Chave,
		Numero:      notification.Numero,
		Serie:       notification.Serie,
		Protocolo:   notification.Protocolo,
		XMLUrl:      notification.XML,
		DanfeUrl:    notification.PDF,
		Mensagem:    notification.Mensagem,
	})
}

// providerStatusUpdate normaliza os campos retornados tanto pelo webhook
// (nfeemissionDomain.WebhookNotification) quanto pela consulta de resumo
// (nfeemissionDomain.ConsultaResumoResponse), para que os dois caminhos
// (webhook e reconciliação — ver fiscal_invoices_reconciliation.go)
// atualizem o status do documento fiscal da mesma forma.
type providerStatusUpdate struct {
	Status      string
	ChaveAcesso string
	Numero      string
	Serie       string
	Protocolo   string
	XMLUrl      string
	DanfeUrl    string
	Mensagem    string
}

// applyProviderStatus aplica no documento fiscal o status reportado pelo
// provedor, qualquer que seja a origem (webhook ou reconciliação por
// polling).
func (s *Service) applyProviderStatus(ctx context.Context, invoice db.FiscalInvoice, update providerStatusUpdate) error {
	var err error

	switch update.Status {
	case "authorized", "autorizado":
		_, err = s.repo.UpdateFiscalInvoiceAuthorized(ctx, db.UpdateFiscalInvoiceAuthorizedParams{
			ID:                   invoice.ID,
			ChaveAcesso:          pgconv.ParseStringToPgType(update.ChaveAcesso),
			Numero:               pgconv.ParseStringToPgType(update.Numero),
			Serie:                pgconv.ParseStringToPgType(update.Serie),
			ProtocoloAutorizacao: pgconv.ParseStringToPgType(update.Protocolo),
			XmlUrl:               pgconv.ParseStringToPgType(update.XMLUrl),
			DanfeUrl:             pgconv.ParseStringToPgType(update.DanfeUrl),
		})
		if err == nil {
			s.recordEvent(ctx, invoice.ID, "authorized", nil)
		}

	case "rejected", "rejeitado":
		_, err = s.repo.UpdateFiscalInvoiceRejected(ctx, db.UpdateFiscalInvoiceRejectedParams{
			ID:           invoice.ID,
			ErrorMessage: pgconv.ParseStringToPgType(update.Mensagem),
		})
		if err == nil {
			s.recordEvent(ctx, invoice.ID, "rejected", map[string]any{"motivo": update.Mensagem})
		}

	case "cancelled", "cancelado":
		_, err = s.repo.UpdateFiscalInvoiceCancelled(ctx, db.UpdateFiscalInvoiceCancelledParams{
			ID:              invoice.ID,
			CancelledReason: pgconv.ParseStringToPgType(update.Mensagem),
		})
		if err == nil {
			s.recordEvent(ctx, invoice.ID, "cancelled", nil)
		}

	default:
		// Status intermediário/desconhecido — não altera o status principal
		// do documento fiscal (o evento webhook_received/reconciled já foi
		// gravado por quem chamou).
		return nil
	}

	if err != nil {
		return fmt.Errorf("fiscal_invoices: falha ao processar atualização de status: %w", err)
	}

	return nil
}
