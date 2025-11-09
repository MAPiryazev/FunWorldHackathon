package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"L3.1/internal/config"
	"L3.1/internal/models"
)

// RabbitMQClient параметры подключения к rabbitmq
type RabbitMQClient struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

// NewRabbitMQClient конструктор для структуры клиента
func NewRabbitMQClient(config config.RabbitMQConfig) (*RabbitMQClient, error) {
	url := fmt.Sprintf("amqp://%s:%s@localhost:5672/", config.User, config.Password)
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return &RabbitMQClient{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish отпрявляет сообщение в rabbitmq и создает очередь если ее нет
func (rc *RabbitMQClient) Publish(ctx context.Context, queueName string, message *models.RabbitMQMessage) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// durable=true → очередь сохранится после рестарта RabbitMQ
	_, err = rc.channel.QueueDeclare(
		queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	err = rc.channel.PublishWithContext(ctx,
		"", queueName, false, false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Consume — читает сообщения из очереди и вызывает обработчик для каждого.
func (rc *RabbitMQClient) Consume(ctx context.Context, queueName string, handler func(msg *models.RabbitMQMessage) error) error {
	// Создаем очередь если её нет
	_, err := rc.channel.QueueDeclare(
		queueName, true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	log.Printf("[RabbitMQ] Начинаем чтение из очереди: %s", queueName)

	msgs, err := rc.channel.Consume(
		queueName,
		"", false, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Printf("[RabbitMQ] Успешно начали чтение из очереди: %s", queueName)

	go func() {
		for d := range msgs {
			var m models.RabbitMQMessage
			if err := json.Unmarshal(d.Body, &m); err != nil {
				log.Printf("[RabbitMQ] Ошибка парсинга сообщения: %v", err)
				d.Nack(false, false) // отклоняем без повторной доставки
				continue
			}

			log.Printf("[RabbitMQ] Получено сообщение из очереди %s: ID=%s", queueName, m.ID)

			if err := handler(&m); err != nil {
				log.Printf("[RabbitMQ] Ошибка обработки сообщения ID=%s: %v", m.ID, err)
				// повторная доставка при ошибке
				d.Nack(false, true)
			} else {
				log.Printf("[RabbitMQ] Сообщение ID=%s успешно обработано", m.ID)
				d.Ack(false)
			}
		}
	}()

	// блокируем до завершения контекста
	<-ctx.Done()
	log.Printf("[RabbitMQ] Чтение из очереди %s остановлено", queueName)
	return ctx.Err()
}

// ConsumeSingleMessage — читает одно сообщение из очереди (неблокирующий метод)
func (rc *RabbitMQClient) ConsumeSingleMessage(ctx context.Context, queueName string) (*models.RabbitMQMessage, error) {
	_, err := rc.channel.QueueDeclare(
		queueName, true, false, false, false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("объявление очереди не удалось: %w", err)
	}

	msg, ok, err := rc.channel.Get(queueName, false)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить сообщение: %w", err)
	}

	if !ok {
		// очередь пуста
		return nil, fmt.Errorf("очередь пуста")
	}

	var m models.RabbitMQMessage
	if err := json.Unmarshal(msg.Body, &m); err != nil {
		msg.Nack(false, true) // возвращаем обратно если не смогли распарсить
		return nil, fmt.Errorf("не удалось распаковать сообщение: %w", err)
	}

	// подтверждаем получение сообщения
	msg.Ack(false)

	return &m, nil
}

// RetryMessage — повторная отправка сообщения через delay.
func (rc *RabbitMQClient) RetryMessage(ctx context.Context, queueName string, message *models.RabbitMQMessage, delaySeconds int) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	time.Sleep(time.Duration(delaySeconds) * time.Second)
	message.RetryCount++
	return rc.Publish(ctx, queueName, message)
}

// QueueLength — возвращает количество сообщений в очереди.
func (rc *RabbitMQClient) QueueLength(ctx context.Context, queueName string) (int, error) {
	q, err := rc.channel.QueueInspect(queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect queue: %w", err)
	}
	return q.Messages, nil
}
