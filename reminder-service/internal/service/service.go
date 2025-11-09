package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"L3.1/internal/infrastructure"
	"L3.1/internal/models"
	"github.com/google/uuid"
)

// Notifier — интерфейс, через который мы отправляем уведомления (например, Gmail)
type Notifier interface {
	Send(ctx context.Context, msg *models.RabbitMQMessage) error
}

// NotificationService — основной сервис для создания, отмены и обработки уведомлений
type NotificationService struct {
	cache            infrastructure.CacheClient
	queue            infrastructure.QueueMQClient
	notifier         Notifier
	queueNameDelayed string
	queueNameReady   string

	retryBaseSeconds int
	maxRetries       int
}

// NewNotificationService — конструктор сервиса
func NewNotificationService(
	cache infrastructure.CacheClient,
	queue infrastructure.QueueMQClient,
	notifier Notifier,
	delayedQueue string,
	readyQueue string,
) (*NotificationService, error) {
	if cache == nil || queue == nil || notifier == nil {
		return nil, fmt.Errorf("не все зависимости переданы в конструктор")
	}

	return &NotificationService{
		cache:            cache,
		queue:            queue,
		notifier:         notifier,
		queueNameReady:   readyQueue,
		queueNameDelayed: delayedQueue,
		retryBaseSeconds: 2,
		maxRetries:       5,
	}, nil
}

// CreateNotification — создаёт новое уведомление
func (ns *NotificationService) CreateNotification(ctx context.Context, to, subject, body string, sendAt time.Time) (string, error) {
	id := uuid.New().String()

	rabbitMsg := &models.RabbitMQMessage{
		ID:         id,
		To:         to,
		Subject:    subject,
		Body:       body,
		SendAt:     sendAt,
		RetryCount: 0,
	}

	redisMsg := &models.RedisMessage{
		ID:        id,
		Status:    "scheduled",
		UpdatedAt: time.Now(),
	}

	if err := ns.cache.Set(ctx, id, redisMsg); err != nil {
		return "", fmt.Errorf("ошибка при добавлении в кэш: %w", err)
	}

	delay := time.Until(sendAt)
	if delay <= 0 {
		// если время отправки уже наступило — публикуем сразу
		if err := ns.queue.Publish(ctx, ns.queueNameReady, rabbitMsg); err != nil {
			return id, fmt.Errorf("немедленная отправка не удалась: %w", err)
		}
		return id, nil
	}

	// публикуем в отложенную очередь
	if err := ns.queue.Publish(ctx, ns.queueNameDelayed, rabbitMsg); err != nil {
		return id, fmt.Errorf("отложенная отправка не удалась: %w", err)
	}

	return id, nil
}

// GetStatus — получение статуса уведомления из Redis
func (ns *NotificationService) GetStatus(ctx context.Context, messageID string) (*models.RedisMessage, error) {
	return ns.cache.Get(ctx, messageID)
}

// CancelNotification — отменяет уведомление
func (ns *NotificationService) CancelNotification(ctx context.Context, messageID string) error {
	redisMessage, err := ns.cache.Get(ctx, messageID)
	if err != nil {
		return fmt.Errorf("не удалось получить сообщение из Redis: %w", err)
	}
	if redisMessage == nil {
		return fmt.Errorf("сообщение с ID %s не найдено", messageID)
	}
	if redisMessage.Status == "canceled" {
		return fmt.Errorf("сообщение уже было отменено")
	}

	cancelMsg := &models.RedisMessage{
		ID:        redisMessage.ID,
		Status:    "canceled",
		UpdatedAt: time.Now(),
	}

	if err := ns.cache.Set(ctx, cancelMsg.ID, cancelMsg); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса отмены: %w", err)
	}

	return nil
}

// StartWorker — запускает воркер для обработки готовых сообщений
func (ns *NotificationService) StartWorker(ctx context.Context) error {
	return ns.queue.Consume(ctx, ns.queueNameReady, func(msg *models.RabbitMQMessage) error {
		return ns.handleMessage(ctx, msg)
	})
}

// StartDelayedWorker — запускает воркер для переноса сообщений из delayed в ready очередь
func (ns *NotificationService) StartDelayedWorker(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second) // проверяем каждые 5 секунд
	defer ticker.Stop()

	log.Println("Delayed worker запущен")

	for {
		select {
		case <-ctx.Done():
			log.Println("Delayed worker остановлен")
			return ctx.Err()
		case <-ticker.C:
			// Пытаемся получить сообщение из delayed очереди
			msg, err := ns.queue.ConsumeSingleMessage(ctx, ns.queueNameDelayed)
			if err != nil {
				continue // если ошибка или очередь пуста - пропускаем
			}

			// Проверяем, наступило ли время отправки
			if time.Now().After(msg.SendAt) || time.Now().Equal(msg.SendAt) {
				// Время пришло - переносим в ready очередь
				if err := ns.queue.Publish(ctx, ns.queueNameReady, msg); err != nil {
					log.Printf("Ошибка при переносе сообщения в ready очередь: %v", err)
					continue
				}
				log.Printf("Сообщение %s перенесено из delayed в ready очередь", msg.ID)
			} else {
				// Время еще не пришло - возвращаем сообщение обратно в delayed очередь
				if err := ns.queue.Publish(ctx, ns.queueNameDelayed, msg); err != nil {
					log.Printf("Ошибка при возврате сообщения в delayed очередь: %v", err)
				}
			}
		}
	}
}

// handleMessage — основная логика обработки сообщения
func (ns *NotificationService) handleMessage(ctx context.Context, msg *models.RabbitMQMessage) error {
	redisMsg, err := ns.cache.Get(ctx, msg.ID)
	if err != nil {
		return fmt.Errorf("ошибка при получении из Redis для id=%s: %w", msg.ID, err)
	}

	// если сообщение было отменено — просто игнорируем
	if redisMsg.Status == "canceled" {
		return nil
	}

	// пробуем отправить
	if err := ns.notifier.Send(ctx, msg); err != nil {
		// достигнут лимит попыток
		if msg.RetryCount >= ns.maxRetries {
			failMsg := &models.RedisMessage{
				ID:        msg.ID,
				Status:    "failed",
				UpdatedAt: time.Now(),
			}
			_ = ns.cache.Set(ctx, msg.ID, failMsg)
			return nil
		}

		// экспоненциальная задержка
		delaySeconds := int(math.Pow(2, float64(msg.RetryCount))) * ns.retryBaseSeconds
		msg.RetryCount++

		if err := ns.queue.RetryMessage(ctx, ns.queueNameReady, msg, delaySeconds); err != nil {
			failMsg := &models.RedisMessage{
				ID:        msg.ID,
				Status:    "failed",
				UpdatedAt: time.Now(),
			}
			_ = ns.cache.Set(ctx, msg.ID, failMsg)
			return fmt.Errorf("ошибка при публикации retry: %w", err)
		}

		redisMsg.Status = "scheduled"
		redisMsg.UpdatedAt = time.Now()
		_ = ns.cache.Set(ctx, msg.ID, redisMsg)

		return fmt.Errorf("ошибка при отправке уведомления: %w", err)
	}

	// успешно отправлено
	done := &models.RedisMessage{
		ID:        msg.ID,
		Status:    "sent",
		UpdatedAt: time.Now(),
	}
	if err := ns.cache.Set(ctx, msg.ID, done); err != nil {
		return fmt.Errorf("ошибка при обновлении статуса после отправки: %w", err)
	}

	return nil
}
