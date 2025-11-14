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
	// Нормализуем время к UTC для корректного сравнения
	now := time.Now().UTC()
	remindAtUTC := req.RemindAt.UTC()
	
	log.Printf("[CreateTask] Время напоминания: %s (UTC: %s), текущее время: %s (UTC)", 
		req.RemindAt.Format(time.RFC3339), 
		remindAtUTC.Format(time.RFC3339),
		now.Format(time.RFC3339))
	
	if remindAtUTC.Before(now) {
		return nil, fmt.Errorf("время напоминания не может быть в прошлом (remind_at: %s, now: %s)", 
			remindAtUTC.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	if req.Complexity < 1 || req.Complexity > 5 {
		req.Complexity = 1
	}

	msg := &models.RabbitMQMessage{
		ID:         uuid.NewString(),
		UserID:     req.UserID,
		ChatID:     req.ChatID,
		Text:       req.Text,
		RemindAt:   remindAtUTC,
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
		ChatID:     req.ChatID,
		Text:       req.Text,
		RemindAt:   remindAtUTC,
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
		log.Printf("[ReadyWorker]  Отправка уведомления для задачи %s (user_id=%s, chat_id=%s, text=%s)",
			msg.ID, msg.UserID, msg.ChatID, msg.Text)
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

		log.Printf("[ReadyWorker]  задача %s успешно отправлена пользователю %s (chat_id=%s)",
			msg.ID, msg.UserID, msg.ChatID)
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

		now := time.Now().UTC()
		remindAtUTC := msg.RemindAt.UTC()
		log.Printf("[DelayedWorker] Проверка задачи %s: remind_at=%s (UTC: %s), now=%s (UTC), before=%v", 
			msg.ID, 
			msg.RemindAt.Format(time.RFC3339), 
			remindAtUTC.Format(time.RFC3339),
			now.Format(time.RFC3339), 
			now.Before(remindAtUTC))
		
		if now.Before(remindAtUTC) {
			// ещё рано — откладываем повторную проверку
			secondsUntil := remindAtUTC.Sub(now).Seconds()
			// используем потолок, чтобы избежать нулевых значений при дробных секундах
			delay := int(math.Ceil(math.Min(secondsUntil, 30)))
			if delay < 1 {
				delay = 1
			}
			log.Printf("[DelayedWorker] Задача %s ещё не готова, откладываем на %d секунд", msg.ID, delay)
			_ = s.queue.Retry(ctx, msg, delay)
			return nil
		}

		// переносим задачу в ready очередь
		log.Printf("[DelayedWorker] ⏰ Время наступило для задачи %s (remind_at=%s, now=%s), переносим в ready_queue", 
			msg.ID, remindAtUTC.Format(time.RFC3339), now.Format(time.RFC3339))
		if err := s.queue.PublishReady(ctx, msg); err != nil {
			log.Printf("[DelayedWorker]  не удалось перенести %s в ready_queue: %v", msg.ID, err)
			return err
		}

		// обновляем статус в Redis
		cached.Status = "ready"
		cached.RetryCount = msg.RetryCount
		cached.UpdatedAt = time.Now().UTC()
		if err := s.cache.Save(ctx, cached); err != nil {
			log.Printf("[DelayedWorker] ⚠️ ошибка обновления статуса Redis: %v", err)
		}

		log.Printf("[DelayedWorker]  задача %s перенесена в ready очередь, ожидаем обработки ReadyWorker", msg.ID)
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
		ChatID:     msg.ChatID,
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

// RecoverPendingTasks — восстанавливает задачи со статусом pending из Redis в delayed_queue
// Вызывается при старте сервиса для восстановления задач после перезапуска
func (s *NotificationService) RecoverPendingTasks(ctx context.Context) error {
	log.Println("[Recovery] Начало восстановления задач из Redis...")

	var cursor uint64
	recovered := 0
	skipped := 0

	// Используем прямой доступ к Redis клиенту для сканирования
	redisClient, ok := s.cache.(*infrastructure.RedisClient)
	if !ok {
		return fmt.Errorf("не удалось получить Redis клиент для восстановления")
	}

	for {
		// Сканируем все ключи reminder:*
		keys, newCursor, err := redisClient.Scan(ctx, cursor, "reminder:*", 100)
		if err != nil {
			return fmt.Errorf("ошибка при сканировании Redis: %w", err)
		}

		for _, key := range keys {
			// Извлекаем ID из ключа (формат: reminder:uuid)
			id := key[9:] // пропускаем "reminder:"
			if len(id) == 0 {
				continue
			}

			msg, err := s.cache.Get(ctx, id)
			if err != nil {
				skipped++
				continue
			}

			// Восстанавливаем все pending задачи
			// Если время уже прошло, DelayedWorker сразу перенесет их в ready_queue
			if msg.Status == "pending" {
				rabbitMsg := &models.RabbitMQMessage{
					ID:         msg.ID,
					UserID:     msg.UserID,
					ChatID:     msg.ChatID,
					Text:       msg.Text,
					RemindAt:   msg.RemindAt,
					Status:     msg.Status,
					RetryCount: msg.RetryCount,
					CreatedAt:  msg.CreatedAt,
					UpdatedAt:  msg.UpdatedAt,
					Complexity: msg.Complexity,
					Priority:   msg.Priority,
					Category:   msg.Category,
					Notes:      msg.Notes,
				}

				if err := s.queue.PublishDelayed(ctx, rabbitMsg); err != nil {
					log.Printf("[Recovery] Не удалось восстановить задачу %s: %v", id, err)
					skipped++
				} else {
					log.Printf("[Recovery] Задача %s восстановлена в delayed_queue", id)
					recovered++
				}
			} else {
				skipped++
			}
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	log.Printf("[Recovery] Восстановление завершено: восстановлено %d, пропущено %d", recovered, skipped)
	return nil
}