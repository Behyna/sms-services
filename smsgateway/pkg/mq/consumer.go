package mq

import (
	"context"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Handle func(ctx context.Context, body []byte) error

type Consumer interface {
	Consume(ctx context.Context, prefetch int, queue string, handler Handle) error
}

type RabbitConsumer struct {
	ch *amqp.Channel
}

func NewRabbitConsumer(ch *amqp.Channel) Consumer {
	return &RabbitConsumer{ch: ch}
}

func (c *RabbitConsumer) Consume(ctx context.Context, prefetch int, queue string, handler Handle) error {
	if prefetch <= 0 {
		prefetch = 1
	}

	if err := c.ch.Qos(prefetch, 0, false); err != nil {
		return err
	}

	deliveries, err := c.ch.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			_ = c.ch.Cancel("", false)
			time.Sleep(50 * time.Millisecond)
			return ctx.Err()

		case d, ok := <-deliveries:
			if !ok {
				return nil
			}

			if err := handler(ctx, d.Body); err == nil {
				_ = d.Ack(false)
				continue
			} else {
				_ = d.Nack(false, shouldRequeue(err))
			}
		}
	}
}

func shouldRequeue(err error) bool {
	type temp interface{ Temporary() bool }
	var te TempError
	if errors.As(err, &te) && te.Temporary() {
		return true
	}
	return false
}
