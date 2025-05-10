package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

// NotificationServiceClient defines the interface for sending notifications
type NotificationServiceClient interface {
	PublishTransactionEvent(ctx context.Context, event NotificationEvent) error
	Close() error
}

// NotificationType represents different types of notifications
type NotificationType string

const (
	NotifyDeposit         NotificationType = "deposit"
	NotifyWithdrawal      NotificationType = "withdrawal"
	NotifyTransferSent    NotificationType = "transfer_sent"
	NotifyTransferReceived NotificationType = "transfer_received"
	NotifyFailedTransaction NotificationType = "failed_transaction"
)

// NotificationEvent represents an event to be published to the notification service
type NotificationEvent struct {
	Type      NotificationType     `json:"type"`
	UserID    uuid.UUID            `json:"user_id"`
	WalletID  uuid.UUID            `json:"wallet_id"`
	Amount    float64              `json:"amount,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time            `json:"timestamp"`
}

// RabbitMQNotificationClient implements NotificationServiceClient using RabbitMQ
type RabbitMQNotificationClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

// NewNotificationClient creates a new notification client
func NewNotificationClient(rabbitMQURL, queueName string) (NotificationServiceClient, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare the queue
	_, err = ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	return &RabbitMQNotificationClient{
		conn:    conn,
		channel: ch,
		queue:   queueName,
	}, nil
	return &RabbitMQNotificationClient{
		conn:    conn,
		channel: ch,
		queue:   queueName,
	}, nil
}

// PublishTransactionEvent publishes a transaction-related notification event
func (c *RabbitMQNotificationClient) PublishTransactionEvent(ctx context.Context, event NotificationEvent) error {
	// Default to current time if timestamp is zero
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Marshal the event to JSON
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal notification event: %w", err)
	}

	// Publish the message
	err = c.channel.Publish(
		"",        // exchange
		c.queue,   // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    event.Timestamp,
		})
	if err != nil {
		return fmt.Errorf("failed to publish notification: %w", err)
	}

	return nil
}

// Close closes the connection to RabbitMQ
func (c *RabbitMQNotificationClient) Close() error {
	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}