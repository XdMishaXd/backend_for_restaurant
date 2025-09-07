package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"main_service/internal/models"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func New(urlForConn string, queueName string) (*RabbitMQClient, error) {
	const op = "rabbimq.New"

	conn, err := amqp.Dial(urlForConn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	q, err := ch.QueueDeclare(
		queueName, true, false, false, false, nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (r *RabbitMQClient) SendNotification(ctx context.Context, booking models.Booking) error {
	const op = "rabbimq.SendNotification"

	body, err := json.Marshal(booking)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return r.channel.PublishWithContext(
		ctx,
		"",
		r.queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func (r *RabbitMQClient) Close() {
	_ = r.channel.Close()
	_ = r.conn.Close()
}
