package consumers

import (
	"encoding/json"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/email"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func StartEmailCOnsumer(amqpChan *amqp.Channel, sender *email.Sender) {
	go func() {
		q, err := amqpChan.QueueDeclare(
			"fila.email",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao declarar a fila de e-mail")
		}

		err = amqpChan.QueueBind(
			q.Name,
			"email.#",
			"protrack.ex.eventos",
			false,
			nil,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao vincular fila de e-mail à Exchange")
		}

		msgs, err := amqpChan.Consume(
			q.Name,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Falha ao registrar consumidor de e-mail")
		}

		log.Info().Msg("Consumidor do RabbitMQ para e-mail iniciado com sucesso e aguardando mensagens...")

		for d := range msgs {
			var sendErr error

			switch d.RoutingKey {
			case "email.password_reset":
				var event events.PasswordResetEmail

				if err := json.Unmarshal(d.Body, &event); err != nil {
					log.Error().Err(err).Msg("Erro ao desserializar evento de reset de senha")
					d.Ack(false)
					continue
				}
				sendErr = sender.SendPasswordReset(event.Email, event.Name, event.ResetURL)

			case "email.password_changed":
				var event events.PasswordChangedEmail
				if err := json.Unmarshal(d.Body, &event); err != nil {
					log.Error().Err(err).Msg("Erro ao desserializar evento de senha alterada")
					d.Ack(false)
					continue
				}
				sendErr = sender.SendPasswordChanged(event.Email, event.Name)
			default:
				log.Warn().Str("routing_key", d.RoutingKey).Msg("Routing key de e-mail desconhecida, descartando mensagem")
				d.Ack(false)
				continue

			}
			if sendErr != nil {
				log.Error().Err(sendErr).Str("routing_key", d.RoutingKey).Msg("Falha ao enviar e-mail. Mensagem será devolvida para a fila.")
				d.Nack(false, false)
			} else {
				d.Ack(false)
			}
		}
	}()
}
