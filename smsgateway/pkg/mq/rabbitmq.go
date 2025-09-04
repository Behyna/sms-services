package mq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Config struct {
	URL string `mapstructure:"url"`
}

type RabbitMQ struct {
	conn   *amqp.Connection
	logger *zap.Logger
}

func NewConnection(cfg Config, logger *zap.Logger) (*RabbitMQ, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		logger.Error("Failed to connect to RabbitMQ",
			zap.Error(err),
			zap.String("url", cfg.URL),
		)
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	logger.Info("Successfully connected to RabbitMQ",
		zap.String("url", cfg.URL),
	)

	return &RabbitMQ{conn: conn, logger: logger}, nil
}

func (r *RabbitMQ) OpenChannel() (*amqp.Channel, error) {
	if r.conn == nil || r.conn.IsClosed() {
		return nil, fmt.Errorf("connection is closed")
	}

	ch, err := r.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return ch, nil
}

func (r *RabbitMQ) DeclareTopology(queues []string) error {
	ch, err := r.OpenChannel()
	if err != nil {
		return fmt.Errorf("failed to open channel for topology: %w", err)
	}
	defer ch.Close()

	for _, queue := range queues {
		_, err := ch.QueueDeclare(queue, true, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queue, err)
		}
	}

	r.logger.Info("Queues declared successfully",
		zap.Int("count", len(queues)),
		zap.Strings("queues", queues),
	)

	return nil
}

func (r *RabbitMQ) CreatePublisher() (Publisher, error) {
	ch, err := r.OpenChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel for publisher: %w", err)
	}

	return NewRabbitPublisher(ch), nil
}

func (r *RabbitMQ) CreateConsumer() (Consumer, error) {
	ch, err := r.OpenChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel for consumer: %w", err)
	}

	return NewRabbitConsumer(ch), nil
}

func (r *RabbitMQ) Close() error {
	if r.conn != nil && !r.conn.IsClosed() {
		return r.conn.Close()
	}

	return nil
}
