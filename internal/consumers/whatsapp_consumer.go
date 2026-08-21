package consumers

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	metaWhatsapp "github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func StartWhatsAppConsumer(amqpChan *amqp.Channel, metaWhatsapp *metaWhatsapp.Service) {
	go func() {
		// 1. Garante que a fila existe (Boa prática caso o container caia ou resete)
		q, err := amqpChan.QueueDeclare(
			"fila.whatsapp", // Nome da fila
			true,            // Durable (sobrevive ao restart do RabbitMQ)
			false,           // Auto-delete
			false,           // Exclusive
			false,           // No-wait
			nil,             // Arguments
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao declarar a fila de WhatsApp")
		}

		// 2. Faz o vinculo (Binding) da fila com a Exchange usando a tag correta
		err = amqpChan.QueueBind(
			q.Name,                // Nome da fila
			"whatsapp.#",          // Routing Key que ele quer escutar
			"protrack.ex.eventos", // Nome da Exchange
			false,
			nil,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao vincular fila à Exchange")
		}

		// 3. Abre o canal de consumo (escuta permanente)
		msgs, err := amqpChan.Consume(
			q.Name, // queue
			"",     // consumer tag (vazio o RabbitMQ gera automático)
			false,  // auto-ack (FALSO: nós avisaremos quando processar com sucesso)
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao registrar consumidor de WhatsApp")
		}

		log.Info().Msg("Consumidor do RabbitMQ para WhatsApp iniciado com sucesso e aguardando mensagens...")

		// 4. Loop infinito processando as mensagens conforme elas entram na fila
		for d := range msgs {
			var event events.WhatsApp

			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Error().Err(err).Msg("Erro ao desserializar JSON da fila")
				d.Ack(false) // Remove da fila se o JSON estiver corrompido para não travar
				continue
			}

			log.Info().Str("sale_id", event.IDSale.String()).Msg("Nova mensagem recebida na fila do WhatsApp. Processando...")

			_, err = metaWhatsapp.SendMessage(context.Background(), event.CompanyID, domain.SendMessageRequest{
				TemplateName:   event.TemplateName,
				LanguageCode:   event.LanguageCode,
				RecipientPhone: event.PhoneNumber,
				Category:       "utility",
				HeaderVariables: map[string]string{
					"1": event.CompanyName,
				},
				Variables: map[string]string{
					"1": event.CustomerName,
					"2": event.PurchaseDate.Format("02/01/2006"),
					"3": strconv.FormatFloat(event.Value, 'f', 2, 64),
					"4": event.DueDate.Format("02/01/2006"),
					"5": event.ContactInfo,
				},
			})

			if err != nil {
				log.Error().Err(err).Str("sale_id", event.IDSale.String()).Msg("Erro ao disparar WhatsApp. Mensagem será devolvida para a fila.")

				// requeue=false: em caso de falha, a mensagem é descartada (não há DLQ configurada ainda).
				// TODO: considerar DLQ + retry com backoff antes de produção com volume real.
				d.Nack(false, false)
			} else {
				log.Info().Str("sale_id", event.IDSale.String()).Msg("WhatsApp enviado com sucesso!")

				// Confirmação de sucesso (Acknowledge) -> Remove a mensagem definitivamente da fila
				d.Ack(false)
			}
		}
	}()
}
