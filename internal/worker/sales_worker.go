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
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			log.Info().Msg("Executando rotina de verificação de débitos vencidos...")

			result, err := saleService.UpdateOverdueSales(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Erro crítico no worker de vendas vencidas")
			} else {
				log.Info().Msg("Status de contas e vendas atualizado com sucesso.")
			}

			if len(result.WhatsAppEvents) == 0 {
				log.Info().Msg("Nenhuma nova venda vencida para processar.")
				return
			}

			for _, sale := range result.WhatsAppEvents {
				body, err := json.Marshal(sale)
				if err != nil {
					log.Error().Err(err).Interface("venda", sale).Msg("Erro ao serializar evento")
					continue
				}

				err = amqpChan.PublishWithContext(
					ctx,
					"protrack.ex.eventos",
					"whatsapp.venda.vencida",
					false,
					false,
					amqp.Publishing{
						ContentType:  "application/json",
						DeliveryMode: amqp.Persistent,
						Body:         body,
					},
				)
				if err != nil {
					log.Error().Err(err).Str("sale_id", sale.CustomerName).Msg("Falha ao enviar para o RabbitMQ")
				}

			}

			for _, ann := range result.AnnouncementEvents {
				body, err := json.Marshal(ann)
				if err != nil {
					log.Error().Err(err).Interface("announcement", ann).Msg("Erro ao serializar announcement")
					continue
				}

				err = amqpChan.PublishWithContext(
					ctx,
					"protrack.ex.eventos",
					"annoucements.venda.vencida.resumo",
					false,
					false,
					amqp.Publishing{
						ContentType:  "application/json",
						DeliveryMode: amqp.Persistent,
						Body:         body,
					},
				)
				if err != nil {
					log.Error().Err(err).Str("company_id", ann.CompanyID.String()).Msg("Falha ao enviar para o RabbitMQ (announcement)")
				}
			}

			log.Info().
				Int("whatsapp_enviados", len(result.WhatsAppEvents)).
				Int("announcements_enviados", len(result.AnnouncementEvents)).
				Msg("Eventos de inadimplência gerados com sucesso.")
		}

		log.Info().Msg("Serviço de Monitoramento vendas iniciado")
		runUpdate()

		for range ticker.C {
			runUpdate()
		}
	}()
}
