package mq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	Publish(ctx context.Context, exchange string, routingKey string, body []byte) error
}

type RabbitPublisher struct {
	ch *amqp.Channel
}

func NewRabbitPublisher(ch *amqp.Channel) Publisher { return &RabbitPublisher{ch: ch} }

func (r *RabbitPublisher) Publish(ctx context.Context, exchange string, routingKey string, body []byte) error {
	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	}

	err := r.ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg)

	return err
}

func (r *RabbitPublisher) Close() error {
	if r.ch != nil {
		return r.ch.Close()
	}

	return nil
}
