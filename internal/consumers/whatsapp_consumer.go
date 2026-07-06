package consumers

import (
	"encoding/json"

	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func StartWhatsAppConsumer(amqpChan *amqp.Channel, whatsAppService *whatsapp.Whatsapp) {
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
			"venda.vencida",       // Routing Key que ele quer escutar
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
			var event domain.UpdateOverdueSalesResponse

			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Error().Err(err).Msg("Erro ao desserializar JSON da fila")
				d.Ack(false) // Remove da fila se o JSON estiver corrompido para não travar
				continue
			}

			log.Info().Str("sale_id", event.IDSale.String()).Msg("Nova mensagem recebida na fila do WhatsApp. Processando...")

			err = whatsAppService.SendWhatsAppMessage(event.PhoneNumber, event.Message, event.InstanceName)
			if err != nil {
				log.Error().Err(err).Str("sale_id", event.IDSale.String()).Msg("Erro ao disparar WhatsApp. Mensagem será devolvida para a fila.")

				// Nack com 'requeue=true' faz a mensagem voltar para o topo da fila para tentar novamente
				// Dica: No futuro você pode acoplar uma DLQ aqui para evitar loops infinitos de erro
				d.Nack(false, true)
			} else {
				log.Info().Str("sale_id", event.IDSale.String()).Msg("WhatsApp enviado com sucesso!")

				// Confirmação de sucesso (Acknowledge) -> Remove a mensagem definitivamente da fila
				d.Ack(false)
			}
		}
	}()
}
