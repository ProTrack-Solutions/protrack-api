package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/service"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func StartOverdueMonitor(saleService *service.Service, amqpChan *amqp.Channel, discordLog *discord.DiscordLogger) {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		runUpdate := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			log.Info().Msg("Executando rotina de verificação de débitos vencidos...")

			result, err := saleService.UpdateOverdueSales(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Erro crítico no worker de vendas vencidas")
				discordLog.Send(domain.LevelError, "Erro crítico no worker de vendas vencidas", err.Error())
				return
			}

			if len(result.WhatsAppEvents) == 0 && len(result.AnnouncementEvents) == 0 {
				log.Info().Msg("Nenhuma nova venda vencida para processar.")
				discordLog.Send(domain.LevelInfo, "Resumo de execução de vendas vencidas", "Nenhuma nova venda vencida para processar.")
				return
			}

			for _, sale := range result.WhatsAppEvents {
				body, err := json.Marshal(sale)
				if err != nil {
					log.Error().Err(err).Interface("venda", sale).Msg("Erro ao serializar evento")
					discordLog.Send(domain.LevelError, "Erro ao serializar evento WhatsApp", fmt.Sprintf("sale_id: %s, erro: %v", sale.CustomerName, err))
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
					discordLog.Send(domain.LevelError, "Falha ao enviar evento WhatsApp para RabbitMQ", fmt.Sprintf("sale_id: %s, erro: %v", sale.CustomerName, err))
				}
			}

			for _, ann := range result.AnnouncementEvents {
				body, err := json.Marshal(ann)
				if err != nil {
					log.Error().Err(err).Interface("announcement", ann).Msg("Erro ao serializar announcement")
					discordLog.Send(domain.LevelError, "Erro ao serializar announcement", fmt.Sprintf("company_id: %s, erro: %v", ann.CompanyID.String(), err))
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
					discordLog.Send(domain.LevelError, "Falha ao enviar evento announcement para RabbitMQ", fmt.Sprintf("company_id: %s, erro: %v", ann.CompanyID.String(), err))
				}
			}

			log.Info().
				Int("whatsapp_enviados", len(result.WhatsAppEvents)).
				Int("announcements_enviados", len(result.AnnouncementEvents)).
				Msg("Eventos de inadimplência gerados com sucesso.")
			discordLog.Send(domain.LevelInfo, "Resumo de execução de vendas vencidas", fmt.Sprintf("Eventos de inadimplência processados com sucesso. WhatsApp=%d, Announcements=%d", len(result.WhatsAppEvents), len(result.AnnouncementEvents)))
		}

		log.Info().Msg("Serviço de Monitoramento vendas iniciado")
		discordLog.Send(domain.LevelInfo, "Worker vendas iniciado", "Serviço de monitoramento de vendas iniciado")
		runUpdate()

		for range ticker.C {
			runUpdate()
		}
	}()
}
