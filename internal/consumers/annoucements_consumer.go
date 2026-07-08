package consumers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/annoucements/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/annoucements/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func StartAnnouncementsConsumer(amqpChan *amqp.Channel, annoucementsService *service.Service) {
	go func() {
		q, err := amqpChan.QueueDeclare(
			"fila.annoucements", // Nome da fila
			true,                // Durable (sobrevive ao restart do RabbitMQ)
			false,               // Auto-delete
			false,               // Exclusive
			false,               // No-wait
			nil,                 // Arguments
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao declarar a fila de annoucements")
		}

		err = amqpChan.QueueBind(
			q.Name,                // Nome da fila
			"annoucements.#",      // Routing Key que ele quer escutar
			"protrack.ex.eventos", // Nome da Exchange
			false,
			nil,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao vincular fila à Exchange")
		}

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
			log.Fatal().Err(err).Msg("Falha ao registrar consumidor de annoucements")
		}

		for d := range msgs {
			var event events.Announcement

			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Error().Err(err).Msg("Erro ao desserializar JSON da fila")
				d.Ack(false) // Remove da fila se o JSON estiver corrompido para não travar
				continue
			}

			log.Info().Str("company_id", event.CompanyID.String()).Msg("Nova mensagem recebida na fila do annoucements. Processando...")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err = annoucementsService.CreateAnnoucements(ctx, uuid.Nil, event.CompanyID, domain.CreateAnnouncementsRequest{
				Title:     event.Title,
				Content:   event.Message,
				Type:      event.Type,
				StartsAt:  event.StartsAt,
				ExpiresAt: event.ExpiresAt,
			}); err != nil {
				log.Info().Str("company_id", event.CompanyID.String()).Msg("Erro ao processar o annoucements")

				d.Nack(false, false)
			} else {
				log.Info().Str("company_id", event.CompanyID.String()).Msg("Annoucements criado com sucesso! ")

				// Confirmação de sucesso (Acknowledge) -> Remove a mensagem definitivamente da fila
				d.Ack(false)
			}

		}
	}()
}
