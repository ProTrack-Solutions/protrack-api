package worker

import (
	"context"
	"time"

	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/sales/service"
	"github.com/rs/zerolog/log"
)

func StartOverdueMonitor(saleService *service.Service) {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		runUpdate := func() {
			/* ctx := context.Background()
			err := saleService.UpdateOverdueSales(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Erro ao atualizar vendas vencidas")
			} else {
				log.Info().Msg("Rotina de monitoramento: Status de vendas atualizado com sucesso.")
			} */

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			log.Info().Msg("Executando rotina de verificação de débitos vencidos...")

			err := saleService.UpdateOverdueSales(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Erro crítico no worker de vendas vencidas")
			} else {
				log.Info().Msg("Status de contas e vendas atualizado com sucesso.")
			}
		}

		log.Info().Msg("Serviço de Monitoramento vendas iniciado")
		runUpdate()

		for range ticker.C {
			runUpdate()
		}
	}()
}
