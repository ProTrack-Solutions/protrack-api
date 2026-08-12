package service

import (
	"context"
	"fmt"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
)

// ReconciliationSummary resume o resultado de uma rodada de reconciliação,
// usado só para log/observabilidade (ver internal/worker).
type ReconciliationSummary struct {
	Checked int
	Updated int
	Errors  int
}

// ReconcilePendingInvoices busca documentos fiscais presos em
// 'processing'/'cancel_processing' há mais de staleAfter e consulta o status
// atual direto no provedor (GET .../resumo). É a rede de segurança para
// quando o webhook do PlugNotas não chega — sem isso, uma nota cujo webhook
// se perdeu (ou cujo webhook nunca foi configurado, ver UploadCertificate)
// ficaria presa em "processing" para sempre.
func (s *Service) ReconcilePendingInvoices(ctx context.Context, staleAfter time.Duration) (ReconciliationSummary, error) {
	var summary ReconciliationSummary

	stale, err := s.repo.ListStaleProcessingFiscalInvoices(ctx, pgconv.TimeToPgTimestamptz(time.Now().Add(-staleAfter)))
	if err != nil {
		return summary, fmt.Errorf("fiscal_invoices: falha ao listar documentos pendentes de reconciliação: %w", err)
	}

	for _, invoice := range stale {
		summary.Checked++

		providerInvoiceID := pgconv.ParsePgTextToString(invoice.ProviderInvoiceID)
		if providerInvoiceID == "" {
			continue
		}

		var resumo struct {
			Status         string
			ChaveAcesso    string
			Numero         string
			Serie          string
			Protocolo      string
			MotivoRejeicao string
		}

		if invoice.Type == "nfe" {
			r, consultErr := s.fiscalProvider.ConsultarNFe(ctx, providerInvoiceID)
			if consultErr != nil {
				summary.Errors++
				continue
			}
			resumo.Status, resumo.ChaveAcesso, resumo.Numero, resumo.Serie, resumo.Protocolo, resumo.MotivoRejeicao =
				r.Status, r.ChaveAcesso, r.Numero, r.Serie, r.Protocolo, r.MotivoRejeicao
		} else {
			r, consultErr := s.fiscalProvider.ConsultarNFCe(ctx, providerInvoiceID)
			if consultErr != nil {
				summary.Errors++
				continue
			}
			resumo.Status, resumo.ChaveAcesso, resumo.Numero, resumo.Serie, resumo.Protocolo, resumo.MotivoRejeicao =
				r.Status, r.ChaveAcesso, r.Numero, r.Serie, r.Protocolo, r.MotivoRejeicao
		}

		s.recordEvent(ctx, invoice.ID, "webhook_received", map[string]any{
			"status": resumo.Status,
			"source": "reconciliation",
		})

		if err := s.applyProviderStatus(ctx, invoice, providerStatusUpdate{
			Status:      resumo.Status,
			ChaveAcesso: resumo.ChaveAcesso,
			Numero:      resumo.Numero,
			Serie:       resumo.Serie,
			Protocolo:   resumo.Protocolo,
			Mensagem:    resumo.MotivoRejeicao,
		}); err != nil {
			summary.Errors++
			continue
		}

		summary.Updated++
	}

	return summary, nil
}
