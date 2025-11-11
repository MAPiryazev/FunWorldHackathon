package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"reminder-service/internal/infrastructure"
	"reminder-service/internal/models"
	"reminder-service/internal/sender"

	"github.com/google/uuid"
)

// NotificationService — интерфейс для работы с напоминаниями
type NotificationServiceIface interface {
	CreateTask(ctx context.Context, req *models.CreateNotificationRequest) (*models.RabbitMQMessage, error)
	CancelTask(ctx context.Context, id string) error
	GetTaskStatus(ctx context.Context, id string) (*models.RedisMessage, error)
	ExtendTaskDeadline(ctx context.Context, id string, newTime time.Time) error
	ListTasksByUser(ctx context.Context, userID string) ([]*models.RedisMessage, error)
}

// NotificationService — основной сервис для создания, отмены и обработки уведомлений
type NotificationService struct {
	cache            infrastructure.CacheClientIface
	queue            infrastructure.QueueRepositoryIface
	sender           sender.SenderIface
	queueNameDelayed string
	queueNameReady   string
	retryBaseSeconds int
	maxRetries       int
}

// NewNotificationService — конструктор сервиса
func NewNotificationService(
	cache infrastructure.CacheClientIface,
	queue infrastructure.QueueRepositoryIface,
	sender sender.SenderIface,
	delayedQueue string,
	readyQueue string,
) (*NotificationService, error) {
	if cache == nil {
		return nil, fmt.Errorf("Redis клиент не передан в конструктор")
	}
	if queue == nil {
		return nil, fmt.Errorf("RabbitMQ клиент не передан в конструктор")
	}
	if sender == nil {
		return nil, fmt.Errorf("Sender не передан в конструктор")
	}
	if delayedQueue == "" || readyQueue == "" {
		return nil, fmt.Errorf("названия очередей не могут быть пустыми")
	}

	return &NotificationService{
		cache:            cache,
		queue:            queue,
		sender:           sender,
		queueNameDelayed: delayedQueue,
		queueNameReady:   readyQueue,
		retryBaseSeconds: 2,
		maxRetries:       5,
	}, nil
}

// CreateTask — создаёт новую задачу/напоминание
func (s *NotificationService) CreateTask(
	ctx context.Context,
	req *models.CreateNotificationRequest,
) (*models.RabbitMQMessage, error) {

	if req == nil {
		return nil, fmt.Errorf("запрос не может быть nil")
	}
	if len(req.UserID) == 0 {
		return nil, fmt.Errorf("UserID обязателен")
	}
	if len(req.Text) == 0 {
		return nil, fmt.Errorf("текст уведомления обязателен")
	}
	if req.RemindAt.Before(time.Now()) {
		return nil, fmt.Errorf("время напоминания не может быть в прошлом")
	}
	if req.Complexity < 1 || req.Complexity > 5 {
		req.Complexity = 1
	}

	now := time.Now()

	msg := &models.RabbitMQMessage{
		ID:         uuid.NewString(),
		UserID:     req.UserID,
		Text:       req.Text,
		RemindAt:   req.RemindAt,
		Status:     "pending",
		RetryCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
		Complexity: req.Complexity,
		Priority:   req.Priority,
		Category:   req.Category,
		Notes:      req.Notes,
	}

	redisMsg := &models.RedisMessage{
		ID:         msg.ID,
		UserID:     req.UserID,
		Text:       req.Text,
		RemindAt:   req.RemindAt,
		Status:     msg.Status,
		RetryCount: msg.RetryCount,
		CreatedAt:  now,
		UpdatedAt:  now,
		Error:      "",
		Complexity: msg.Complexity,
		Priority:   msg.Priority,
		Category:   msg.Category,
		Notes:      msg.Notes,
	}
	if err := s.cache.Save(ctx, redisMsg); err != nil {
		return nil, fmt.Errorf("не удалось сохранить задачу в Redis: %w", err)
	}

	if err := s.queue.PublishDelayed(ctx, msg); err != nil {
		// При ошибке публикации лучше удалить запись из Redis
		_ = s.cache.Delete(ctx, msg.ID)
		return nil, fmt.Errorf("не удалось отправить задачу в очередь delayed: %w", err)
	}

	return msg, nil
}

// CancelTask — отменяет задачу по ID
func (s *NotificationService) CancelTask(ctx context.Context, id string) error {
	if len(id) == 0 {
		return fmt.Errorf("ID задачи обязателен")
	}

	exists, err := s.cache.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("ошибка проверки задачи в Redis: %w", err)
	}
	if !exists {
		return fmt.Errorf("задача с ID %s не найдена", id)
	}

	msg, err := s.cache.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("ошибка получения задачи из Redis: %w", err)
	}

	if msg.Status == "sent" || msg.Status == "cancelled" {
		return fmt.Errorf("задачу с ID %s нельзя отменить, статус: %s", id, msg.Status)
	}

	msg.Status = "cancelled"
	msg.Error = ""
	msg.UpdatedAt = time.Now()

	if err := s.cache.Save(ctx, msg); err != nil {
		return fmt.Errorf("не удалось обновить статус задачи в Redis: %w", err)
	}

	return nil
}

// GetTaskStatus — возвращает текущее состояние задачи по ID
func (s *NotificationService) GetTaskStatus(ctx context.Context, id string) (*models.RedisMessage, error) {
	if len(id) == 0 {
		return nil, fmt.Errorf("ID задачи обязателен")
	}

	// Проверяем, есть ли такая задача в Redis
	exists, err := s.cache.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка при проверке существования задачи: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("задача с ID %s не найдена", id)
	}

	// Получаем саму задачу
	msg, err := s.cache.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения задачи из Redis: %w", err)
	}

	return msg, nil
}

// StartReadyWorker — обрабатывает задачи из ready_queue:
// вызывает sender, обновляет статусы и делает ретраи при ошибках.
func (s *NotificationService) StartReadyWorker(ctx context.Context) error {
	log.Println("[ReadyWorker] запущен...")

	handler := func(msg *models.RabbitMQMessage) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[ReadyWorker] паника обработана: %v", r)
			}
		}()

		if msg == nil {
			return errors.New("nil сообщение получено")
		}

		cached, err := s.cache.Get(ctx, msg.ID)
		if err != nil {
			log.Printf("[ReadyWorker] задача %s не найдена в Redis: %v", msg.ID, err)
			return nil
		}

		if cached.Status == "cancelled" {
			log.Printf("[ReadyWorker] задача %s отменена — пропуск", msg.ID)
			return nil
		}

		// попытка отправки через sender
		err = s.sender.Send(ctx, msg)
		if err != nil {
			msg.RetryCount++
			cached.Status = "failed"
			cached.Error = err.Error()
			cached.RetryCount = msg.RetryCount
			cached.UpdatedAt = time.Now()
			_ = s.cache.Save(ctx, cached)

			if msg.RetryCount <= s.maxRetries {
				delay := s.retryBaseSeconds * int(math.Pow(2, float64(msg.RetryCount-1)))
				log.Printf("[ReadyWorker] отправка задачи %s не удалась, повтор через %ds", msg.ID, delay)
				_ = s.queue.Retry(ctx, msg, delay)
			} else {
				log.Printf("[ReadyWorker] задача %s достигла лимита попыток (%d) — failed", msg.ID, s.maxRetries)
			}
			return err
		}

		// успешная отправка
		cached.Status = "sent"
		cached.Error = ""
		cached.RetryCount = msg.RetryCount
		cached.UpdatedAt = time.Now()
		if err := s.cache.Save(ctx, cached); err != nil {
			log.Printf("[ReadyWorker] ошибка сохранения статуса sent в Redis: %v", err)
		}

		log.Printf("[ReadyWorker] задача %s успешно отправлена", msg.ID)
		return nil
	}

	return s.queue.ConsumeReady(ctx, handler)
}

// StartDelayedWorker — переносит задачи из delayed_queue в ready_queue,
// когда наступает время RemindAt.
func (s *NotificationService) StartDelayedWorker(ctx context.Context) error {
	log.Println("[DelayedWorker] запущен...")

	handler := func(msg *models.RabbitMQMessage) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[DelayedWorker] паника обработана: %v", r)
			}
		}()

		if msg == nil {
			return errors.New("получено пустое сообщение")
		}

		cached, err := s.cache.Get(ctx, msg.ID)
		if err != nil {
			log.Printf("[DelayedWorker] сообщение %s не найдено в Redis: %v", msg.ID, err)
			return nil // пропускаем, чтобы не застревать
		}

		if cached.Status == "cancelled" || cached.Status == "sent" {
			log.Printf("[DelayedWorker] задача %s уже %s, пропуск", msg.ID, cached.Status)
			return nil
		}

		now := time.Now()
		if now.Before(msg.RemindAt) {
			// ещё рано — откладываем повторную проверку
			secondsUntil := msg.RemindAt.Sub(now).Seconds()
			// используем потолок, чтобы избежать нулевых значений при дробных секундах
			delay := int(math.Ceil(math.Min(secondsUntil, 30)))
			if delay < 1 {
				delay = 1
			}
			_ = s.queue.Retry(ctx, msg, delay)
			return nil
		}

		// переносим задачу в ready очередь
		if err := s.queue.PublishReady(ctx, msg); err != nil {
			log.Printf("[DelayedWorker] не удалось перенести %s в ready_queue: %v", msg.ID, err)
			return err
		}

		// обновляем статус в Redis
		cached.Status = "ready"
		cached.RetryCount = msg.RetryCount
		cached.UpdatedAt = time.Now()
		if err := s.cache.Save(ctx, cached); err != nil {
			log.Printf("[DelayedWorker] ошибка обновления статуса Redis: %v", err)
		}

		log.Printf("[DelayedWorker] задача %s перенесена в ready очередь", msg.ID)
		return nil
	}

	return s.queue.ConsumeDelayed(ctx, handler)
}

// ExtendTaskDeadline — изменяет время напоминания, если оно ещё не отправлено
func (s *NotificationService) ExtendTaskDeadline(ctx context.Context, id string, newTime time.Time) error {
	if len(id) == 0 {
		return fmt.Errorf("ID задачи обязателен")
	}
	if newTime.Before(time.Now()) {
		return fmt.Errorf("нельзя перенести задачу в прошлое время")
	}

	msg, err := s.cache.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("ошибка получения задачи из Redis: %w", err)
	}

	if msg.Status == "sent" || msg.Status == "cancelled" {
		return fmt.Errorf("невозможно изменить дедлайн задачи со статусом %s", msg.Status)
	}

	msg.RemindAt = newTime
	msg.RetryCount = 0
	msg.Status = "pending"
	msg.Error = ""
	msg.UpdatedAt = time.Now()

	// обновляем и в RabbitMQ
	rabbitMsg := &models.RabbitMQMessage{
		ID:         msg.ID,
		UserID:     msg.UserID,
		Text:       msg.Text,
		RemindAt:   newTime,
		Status:     "pending",
		RetryCount: msg.RetryCount,
		CreatedAt:  msg.CreatedAt,
		UpdatedAt:  msg.UpdatedAt,
		Complexity: msg.Complexity,
		Priority:   msg.Priority,
		Category:   msg.Category,
		Notes:      msg.Notes,
	}

	// публикуем обратно в delayed очередь
	if err := s.queue.PublishDelayed(ctx, rabbitMsg); err != nil {
		return fmt.Errorf("ошибка при обновлении очереди delayed: %w", err)
	}

	if err := s.cache.Save(ctx, msg); err != nil {
		log.Printf("[ExtendTaskDeadline] не удалось сохранить задачу в Redis: %v", err)
	}

	log.Printf("[ExtendTaskDeadline] задача %s перенесена на %s", id, newTime.Format(time.RFC3339))
	return nil
}

// ListTasksByUser — возвращает все задачи пользователя
func (s *NotificationService) ListTasksByUser(ctx context.Context, userID string) ([]*models.RedisMessage, error) {
	if len(userID) == 0 {
		return nil, fmt.Errorf("userID обязателен")
	}

	tasks, err := s.cache.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении задач пользователя %s: %w", userID, err)
	}

	if len(tasks) == 0 {
		log.Printf("[ListTasksByUser] у пользователя %s нет активных задач", userID)
	}

	return tasks, nil
}
