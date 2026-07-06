package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/sales/service"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func StartOverdueMonitor(saleService *service.Service, amqpChan *amqp.Channel) {
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

			sales, err := saleService.UpdateOverdueSales(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Erro crítico no worker de vendas vencidas")
			} else {
				log.Info().Msg("Status de contas e vendas atualizado com sucesso.")
			}

			if len(sales) == 0 {
				log.Info().Msg("Nenhuma nova venda vencida para processar.")
				return
			}

			for _, sale := range sales {
				body, err := json.Marshal(sale)
				if err != nil {
					log.Error().Err(err).Interface("venda", sale).Msg("Erro ao serializar evento")
					continue
				}

				err = amqpChan.PublishWithContext(
					ctx,
					"protrack.ex.eventos",
					"venda.vencida",
					false,
					false,
					amqp.Publishing{
						ContentType:  "application/json",
						DeliveryMode: amqp.Persistent,
						Body:         body,
					},
				)
				if err != nil {
					log.Error().Err(err).Str("sale_id", sale.IDSale.String()).Msg("Falha ao enviar para o RabbitMQ")
				}
			}

			log.Info().Int("total_enviado", len(sales)).Msg("Eventos de inadimplência gerados com sucesso.")
		}

		log.Info().Msg("Serviço de Monitoramento vendas iniciado")
		runUpdate()

		for range ticker.C {
			runUpdate()
		}
	}()
}
