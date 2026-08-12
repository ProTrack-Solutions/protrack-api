package worker

import (
	"context"
	"time"

	fiscalInvoicesService "github.com/ProTrack-Solutions/protrack-api/internal/fiscal_invoices/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/rs/zerolog/log"
)

// StartFiscalInvoiceReconciliation inicia o polling periódico de status das
// notas fiscais presas em "processing"/"cancel_processing" — rede de
// segurança para quando o webhook do provedor fiscal não chega (não
// configurado, perdido, etc). Roda a cada 10 minutos, reconciliando notas
// sem atualização há mais de 15 minutos.
func StartFiscalInvoiceReconciliation(fiscalService *fiscalInvoicesService.Service, discordLog *discord.DiscordLogger) {
	const (
		interval   = 10 * time.Minute
		staleAfter = 15 * time.Minute
	)

	ticker := time.NewTicker(interval)

	go func() {
		runReconciliation := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			summary, err := fiscalService.ReconcilePendingInvoices(ctx, staleAfter)
			if err != nil {
				log.Error().Err(err).Msg("Erro crítico no worker de reconciliação fiscal")
				discordLog.Send(domain.LevelError, "Erro crítico no worker de reconciliação fiscal", err.Error())
				return
			}

			if summary.Checked == 0 {
				return
			}

			log.Info().
				Int("verificados", summary.Checked).
				Int("atualizados", summary.Updated).
				Int("erros", summary.Errors).
				Msg("Reconciliação de documentos fiscais executada")

			if summary.Errors > 0 {
				discordLog.Send(domain.LevelError, "Reconciliação fiscal com falhas",
					"Alguns documentos fiscais não puderam ser reconciliados com o provedor — ver logs")
			}
		}

		log.Info().Msg("Worker de reconciliação fiscal iniciado")
		runReconciliation()

		for range ticker.C {
			runReconciliation()
		}
	}()
}
