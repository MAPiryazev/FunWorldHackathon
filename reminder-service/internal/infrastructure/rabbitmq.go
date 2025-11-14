package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"reminder-service/internal/config"
	"reminder-service/internal/models"
)

// RabbitMQClient инкапсулирует работу с RabbitMQ и обеспечивает автоматическое восстановление соединения.
type RabbitMQClient struct {
	url     string
	conn    *amqp091.Connection
	channel *amqp091.Channel
	mu      sync.Mutex
}

// NewRabbitMQClient создаёт новый экземпляр клиента RabbitMQ.
func NewRabbitMQClient(cfg *config.RabbitMQConfig) (*RabbitMQClient, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.User, cfg.Password, cfg.Host, cfg.Port)

	client := &RabbitMQClient{
		url: url,
	}

	if err := client.connectLocked(); err != nil {
		return nil, fmt.Errorf("не удалось установить соединение с RabbitMQ: %w", err)
	}

	return client, nil
}

func (rc *RabbitMQClient) connectLocked() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if err := rc.dialLocked(); err != nil {
		return err
	}

	return nil
}

func (rc *RabbitMQClient) dialLocked() error {
	conn, err := amqp091.Dial(rc.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	rc.conn = conn
	rc.channel = ch
	rc.watchClosures(conn, ch)
	log.Printf("[RabbitMQ] Соединение установлено")
	return nil
}

func (rc *RabbitMQClient) watchClosures(conn *amqp091.Connection, ch *amqp091.Channel) {
	connClose := conn.NotifyClose(make(chan *amqp091.Error, 1))
	chClose := ch.NotifyClose(make(chan *amqp091.Error, 1))

	go func() {
		for err := range connClose {
			log.Printf("[RabbitMQ] Соединение закрыто: %v", err)
			rc.mu.Lock()
			if rc.conn == conn {
				rc.closeLocked()
			}
			rc.mu.Unlock()
		}
	}()

	go func() {
		for err := range chClose {
			log.Printf("[RabbitMQ] Канал закрыт: %v", err)
			rc.mu.Lock()
			if rc.channel == ch {
				_ = rc.channel.Close()
				rc.channel = nil
			}
			rc.mu.Unlock()
		}
	}()
}

func (rc *RabbitMQClient) ensureConnection() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.conn != nil && !rc.conn.IsClosed() && rc.channel != nil && !rc.channel.IsClosed() {
		return nil
	}

	rc.closeLocked()
	return rc.dialLocked()
}

func (rc *RabbitMQClient) closeLocked() {
	if rc.channel != nil {
		_ = rc.channel.Close()
		rc.channel = nil
	}
	if rc.conn != nil {
		_ = rc.conn.Close()
		rc.conn = nil
	}
}

func (rc *RabbitMQClient) resetConnection() {
	rc.mu.Lock()
	rc.closeLocked()
	rc.mu.Unlock()
}

// publishHelper публикует сообщение в указанную очередь с автоматическим восстановлением соединения.
func (rc *RabbitMQClient) publishHelper(ctx context.Context, queueName string, message *models.RabbitMQMessage) error {
	if message == nil {
		return fmt.Errorf("nil сообщение нельзя отправить в очередь rabbit")
	}

	if len(message.UserID) == 0 {
		return fmt.Errorf("сообщение без userID нельзя отправить в очередь rabbit")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка сериализации сообщения: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := rc.ensureConnection(); err != nil {
			lastErr = fmt.Errorf("не удалось восстановить соединение с RabbitMQ: %w", err)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		rc.mu.Lock()
		ch := rc.channel
		err = rc.publishLocked(ctx, ch, queueName, data)
		rc.mu.Unlock()

		if err == nil {
			log.Printf("[RabbitMQ] Сообщение %s опубликовано в очередь %s", message.ID, queueName)
			return nil
		}

		lastErr = err
		log.Printf("[RabbitMQ] Ошибка публикации в очередь %s (попытка %d): %v", queueName, attempt+1, err)
		rc.resetConnection()
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	return fmt.Errorf("ошибка публикации сообщения в очередь %s: %w", queueName, lastErr)
}

func (rc *RabbitMQClient) publishLocked(ctx context.Context, ch *amqp091.Channel, queueName string, data []byte) error {
	if ch == nil {
		return amqp091.ErrClosed
	}

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return err
	}

	return ch.PublishWithContext(ctx,
		"", queueName, false, false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp091.Persistent,
		},
	)
}

// PublishDelayed публикует сообщение в delayed_queue.
func (rc *RabbitMQClient) PublishDelayed(ctx context.Context, message *models.RabbitMQMessage) error {
	return rc.publishHelper(ctx, "delayed_queue", message)
}

// PublishReady публикует сообщение в ready_queue.
func (rc *RabbitMQClient) PublishReady(ctx context.Context, message *models.RabbitMQMessage) error {
	return rc.publishHelper(ctx, "ready_queue", message)
}

// consumeHelper обеспечивает устойчивое потребление указанной очереди.
func (rc *RabbitMQClient) consumeHelper(ctx context.Context, queueName string, handler func(msg *models.RabbitMQMessage) error) error {
	for {
		if err := rc.ensureConnection(); err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("[RabbitMQ] Не удалось подготовить потребление %s: %v", queueName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		rc.mu.Lock()
		ch := rc.channel
		if ch == nil {
			rc.mu.Unlock()
			time.Sleep(2 * time.Second)
			continue
		}

		if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
			rc.mu.Unlock()
			rc.resetConnection()
			log.Printf("[RabbitMQ] Ошибка объявления очереди %s: %v", queueName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		msgs, err := ch.Consume(queueName, "", false, false, false, false, nil)
		if err != nil {
			rc.mu.Unlock()
			rc.resetConnection()
			log.Printf("[RabbitMQ] Ошибка подписки на очередь %s: %v", queueName, err)
			time.Sleep(2 * time.Second)
			continue
		}
		rc.mu.Unlock()

		log.Printf("[RabbitMQ] Начато потребление очереди %s", queueName)
		go rc.handleDeliveries(ctx, queueName, msgs, handler)
		return nil
	}
}

// ConsumeDelayed запускает потребитель для delayed_queue.
func (rc *RabbitMQClient) ConsumeDelayed(ctx context.Context, handler func(msg *models.RabbitMQMessage) error) error {
	return rc.consumeHelper(ctx, "delayed_queue", handler)
}

// ConsumeReady запускает потребитель для ready_queue.
func (rc *RabbitMQClient) ConsumeReady(ctx context.Context, handler func(msg *models.RabbitMQMessage) error) error {
	return rc.consumeHelper(ctx, "ready_queue", handler)
}

// Retry повторно публикует сообщение в delayed_queue с задержкой.
func (rc *RabbitMQClient) Retry(ctx context.Context, message *models.RabbitMQMessage, delaySeconds int) error {
	if message == nil {
		return fmt.Errorf("сообщение nil")
	}

	if delaySeconds < 1 {
		delaySeconds = 1
	}

	time.AfterFunc(time.Duration(delaySeconds)*time.Second, func() {
		if err := rc.PublishDelayed(ctx, message); err != nil {
			log.Printf("[RabbitMQ] Ошибка повторной отправки сообщения: %v", err)
		}
	})

	return nil
}

// QueueLength возвращает количество сообщений в очереди.
func (rc *RabbitMQClient) QueueLength(ctx context.Context, queueName string) (int, error) {
	if err := rc.ensureConnection(); err != nil {
		return 0, err
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.channel == nil {
		return 0, amqp091.ErrClosed
	}

	q, err := rc.channel.QueueDeclarePassive(queueName, true, false, false, false, nil)
	if err != nil {
		return 0, fmt.Errorf("не удалось получить длину очереди %s: %w", queueName, err)
	}
	return int(q.Messages), nil
}

func (rc *RabbitMQClient) handleDeliveries(
	ctx context.Context,
	queueName string,
	deliveries <-chan amqp091.Delivery,
	handler func(msg *models.RabbitMQMessage) error,
) {
	for delivery := range deliveries {
		var msg models.RabbitMQMessage
		if err := json.Unmarshal(delivery.Body, &msg); err != nil {
			log.Printf("[RabbitMQ] Ошибка десериализации сообщения из %s: %v", queueName, err)
			delivery.Nack(false, false)
			continue
		}

		if err := handler(&msg); err != nil {
			log.Printf("[RabbitMQ] Ошибка обработки сообщения %s из %s: %v", msg.ID, queueName, err)
			delivery.Nack(false, true)
			continue
		}

		delivery.Ack(false)
	}

	log.Printf("[RabbitMQ] Потребление очереди %s завершено", queueName)
	if ctx.Err() != nil {
		return
	}

	go func() {
		if err := rc.consumeHelper(ctx, queueName, handler); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("[RabbitMQ] Не удалось перезапустить потребление %s: %v", queueName, err)
		}
	}()
}
