package worker

import (
	"context"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/service"
	"github.com/rs/zerolog/log"
)

func StartBillPayableOverdueMonitor(billPayableService *service.Service) {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		runUpdate := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			log.Info().Msg("Executando rotina de verificação de contas a pagar vencidos...")

			if err := billPayableService.UpdateOverdueBillsPayable(ctx); err != nil {
				log.Error().Err(err).Msg("Erro crítico no worker de contas a pagar vencidas")
			} else {
				log.Info().Msg("Status de contas a pagar atualizado com sucesso.")
			}
		}

		log.Info().Msg("Serviço de Monitoramento contas a pagar iniciado")
		runUpdate()

		for range ticker.C {
			runUpdate()
		}
	}()
}
