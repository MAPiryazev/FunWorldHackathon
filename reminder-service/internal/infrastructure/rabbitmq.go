package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"reminder-service/internal/config"
	"reminder-service/internal/models"
)

// RabbitMQClient параметры подключения к rabbitmq
type RabbitMQClient struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

// NewRabbitMQClient конструктор для структуры клиента с использованием конфигурации из ENV
func NewRabbitMQClient(cfg *config.RabbitMQConfig) (*RabbitMQClient, error) {
	// формируем URL подключения
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.User, cfg.Password, cfg.Host, cfg.Port)

	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("не удалось открыть канал RabbitMQ: %w", err)
	}

	// Сообщения обрабатываем по одному на потребителя, чтобы не вымывать всю очередь в Unacked
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("не удалось настроить QoS RabbitMQ: %w", err)
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: ch,
	}, nil
}

// publishHelper отпрявляет сообщение в rabbitmq и создает очередь если ее нет
func (rc *RabbitMQClient) publishHelper(ctx context.Context, queueName string, message *models.RabbitMQMessage) error {
	if message == nil {
		return fmt.Errorf("nil сообщение нельзя записать в очередь rabbit")
	}

	if len(message.UserID) == 0 {
		return fmt.Errorf("сообщение без UserID нельзя записать в очередь rabbit")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка распаковки сообщения : %w", err)
	}

	// durable=true → очередь сохранится после рестарта RabbitMQ
	_, err = rc.channel.QueueDeclare(
		queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("ошибка объявления очереди: %w", err)
	}

	err = rc.channel.PublishWithContext(ctx,
		"", queueName, false, false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp091.Persistent, // гарантируем сохранность вместе с durable очередью
		},
	)
	if err != nil {
		return fmt.Errorf("не удалось опубликовать сообщение: %w", err)
	}

	log.Printf("[RabbitMQ] Сообщение %s опубликовано в очередь %s", message.ID, queueName)
	return nil
}

// PublishDelayed — отправляет сообщение в очередь delayed_queue
func (rc *RabbitMQClient) PublishDelayed(ctx context.Context, message *models.RabbitMQMessage) error {
	return rc.publishHelper(ctx, "delayed_queue", message)
}

// PublishReady — отправляет сообщение в очередь ready_queue
func (rc *RabbitMQClient) PublishReady(ctx context.Context, message *models.RabbitMQMessage) error {
	return rc.publishHelper(ctx, "ready_queue", message)
}

// универсальный метод consume
func (rc *RabbitMQClient) consumeHelper(ctx context.Context, queueName string, handler func(msg *models.RabbitMQMessage) error) error {
	_, err := rc.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("не удалось объявить очередь %s: %w", queueName, err)
	}

	msgs, err := rc.channel.Consume(
		queueName, "", false, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("не удалось начать потребление очереди %s: %w", queueName, err)
	}

	// запустим горутину для обработки сообщений
	go func() {
		for d := range msgs {
			var msg models.RabbitMQMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Ошибка при разборе сообщения: %v", err)
				d.Nack(false, false)
				continue
			}

			if err := handler(&msg); err != nil {
				log.Printf("Ошибка обработки сообщения: %v", err)
				d.Nack(false, true)
			} else {
				d.Ack(false)
			}
		}
	}()

	return nil
}

// ConsumeDelayed — читает сообщения из delayed_queue и вызывает handler
func (rc *RabbitMQClient) ConsumeDelayed(ctx context.Context, handler func(msg *models.RabbitMQMessage) error) error {
	return rc.consumeHelper(ctx, "delayed_queue", handler)
}

// ConsumeReady — читает сообщения из ready_queue и вызывает handler
func (rc *RabbitMQClient) ConsumeReady(ctx context.Context, handler func(msg *models.RabbitMQMessage) error) error {
	return rc.consumeHelper(ctx, "ready_queue", handler)
}

// Retry — отправляет сообщение обратно в очередь с задержкой (через time.AfterFunc)
func (rc *RabbitMQClient) Retry(ctx context.Context, message *models.RabbitMQMessage, delaySeconds int) error {
	if message == nil {
		return fmt.Errorf("сообщение nil")
	}

	if delaySeconds < 1 {
		delaySeconds = 1
	}

	time.AfterFunc(time.Duration(delaySeconds)*time.Second, func() {
		if err := rc.PublishDelayed(ctx, message); err != nil {
			log.Printf("Ошибка повторной отправки сообщения: %v", err)
		}
	})

	return nil
}

// QueueLength — возвращает количество сообщений в очереди
func (rc *RabbitMQClient) QueueLength(ctx context.Context, queueName string) (int, error) {
	q, err := rc.channel.QueueDeclarePassive(queueName, true, false, false, false, nil)
	if err != nil {
		return 0, fmt.Errorf("не удалось получить длину очереди %s: %w", queueName, err)
	}
	return int(q.Messages), nil
}
