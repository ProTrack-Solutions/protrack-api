package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

const eventsExchange = "protrack.ex.eventos"

// Publisher publica eventos JSON persistentes na exchange de eventos
type Publisher struct {
	ch *amqp.Channel
}

func NewPublisher(ch *amqp.Channel) *Publisher {
	return &Publisher{ch: ch}
}

func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	return p.ch.PublishWithContext(ctx, eventsExchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}
