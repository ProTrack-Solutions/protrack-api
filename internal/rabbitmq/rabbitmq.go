package rabbitmq

import (
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

func InitializeRabbitMQ(cfg *config.Config) (*amqp.Connection, *amqp.Channel, error) {
	var conn *amqp.Connection
	var err error

	for i := 1; i <= 10; i++ {
		log.Info().Msgf("Tentando conectar ao RabbitMQ (Tentativa %d de 10)...", i)

		conn, err = amqp.Dial(cfg.RabbitURL)
		if err == nil {
			log.Info().Msg("Conectado ao RabbitMQ com sucesso!")
			break
		}

		log.Warn().Err(err).Msg("Broker indisponível ou iniciando. Aguardando 3 segundos...")
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("não foi possível conectar ao RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close() // Fecha a conexão caso falhe em abrir o canal para não vazar recursos
		return nil, nil, fmt.Errorf("falha ao abrir canal de comunicação: %w", err)
	}

	// 1. Cria a Exchange (se já existir com a mesma configuração, ele só ignora)
	err = ch.ExchangeDeclare(
		"protrack.ex.eventos", // nome da exchange
		"topic",               // tipo (direct, topic, fanout...)
		true,                  // durable (sobrevive a restarts)
		false,                 // auto-deleted
		false,                 // internal
		false,                 // no-wait
		nil,                   // argumentos extras
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Falha ao declarar a Exchange")
	}

	log.Info().Msg("Canal do RabbitMQ aberto e pronto para uso.")
	return conn, ch, nil
}
