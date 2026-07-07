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

	// 2. Cria a Fila (onde a mensagem vai ficar guardada)
	fila, err := ch.QueueDeclare(
		"fila.vendas.vencidas", // nome da fila
		true,                   // durable (mensagens não somem se o rabbit reiniciar)
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // argumentos
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Falha ao declarar a fila")
	}

	// 3. Conecta a Fila na Exchange especificando a chave de rota ("venda.vencida")
	err = ch.QueueBind(
		fila.Name,             // nome da fila que acabamos de criar
		"venda.vencida",       // routing key (precisa ser igual a do PublishWithContext)
		"protrack.ex.eventos", // exchange
		false,
		nil,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Falha ao fazer o bind da fila com a exchange")
	}

	log.Info().Msg("Canal do RabbitMQ aberto e pronto para uso.")
	return conn, ch, nil
}
