package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"reminder-service/internal/models"
)

const (
	// DefaultNotificationEndpoint — фиксированный endpoint для отправки уведомлений
	// DefaultNotificationEndpoint = "http://localhost:8081/notifications"
	DefaultNotificationEndpoint = "https://webhook.site/db281a47-ed45-44d6-b329-c44db8b86fb9"
)

// SenderIface — интерфейс для любого способа отправки уведомлений
type SenderIface interface {
	Send(ctx context.Context, msg *models.RabbitMQMessage) error
}

// HTTPSender — реализация отправки уведомлений через HTTP POST запрос
type HTTPSender struct {
	client      *http.Client
	endpointURL string
}

// NewHTTPSender создает новый HTTPSender с фиксированным endpoint
func NewHTTPSender(timeout time.Duration) *HTTPSender {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &HTTPSender{
		client: &http.Client{
			Timeout: timeout,
		},
		endpointURL: DefaultNotificationEndpoint,
	}
}

// Send отправляет уведомление на внешний endpoint в формате JSON
func (hs *HTTPSender) Send(ctx context.Context, msg *models.RabbitMQMessage) error {
	if msg == nil {
		return fmt.Errorf("сообщение не может быть nil")
	}

	// Подготавливаем JSON payload
	payload := map[string]interface{}{
		"id":          msg.ID,
		"user_id":     msg.UserID,
		"text":        msg.Text,
		"remind_at":   msg.RemindAt.Format(time.RFC3339),
		"status":      msg.Status,
		"retry_count": msg.RetryCount,
		"created_at":  msg.CreatedAt.Format(time.RFC3339),
		"updated_at":  msg.UpdatedAt.Format(time.RFC3339),
	}

	// Добавляем опциональные поля, если они есть
	if msg.Complexity > 0 {
		payload["complexity"] = msg.Complexity
	}
	if msg.Priority != "" {
		payload["priority"] = msg.Priority
	}
	if msg.Category != "" {
		payload["category"] = msg.Category
	}
	if msg.Notes != "" {
		payload["notes"] = msg.Notes
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hs.endpointURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания HTTP запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	log.Printf("[HTTPSender] Отправка уведомления %s для пользователя %s на %s", msg.ID, msg.UserID, hs.endpointURL)
	resp, err := hs.client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа для логирования
	body, _ := io.ReadAll(resp.Body)

	// Проверяем статус код
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("неуспешный статус код %d от внешнего API: %s", resp.StatusCode, string(body))
	}

	log.Printf("[HTTPSender] Уведомление %s успешно отправлено (статус: %d)", msg.ID, resp.StatusCode)
	return nil
}

// LogSender — заглушка для разработки и тестирования
type LogSender struct{}

// NewLogSender создает новый LogSender
func NewLogSender() *LogSender {
	return &LogSender{}
}

// Send логирует уведомление вместо реальной отправки
func (ls *LogSender) Send(ctx context.Context, msg *models.RabbitMQMessage) error {
	if msg == nil {
		return fmt.Errorf("сообщение не может быть nil")
	}

	log.Printf("[LogSender]   УВЕДОМЛЕНИЕ для пользователя %s:", msg.UserID)
	log.Printf("[LogSender]   ID: %s", msg.ID)
	log.Printf("[LogSender]   Текст: %s", msg.Text)
	log.Printf("[LogSender]   Время: %s", msg.RemindAt.Format("2006-01-02 15:04:05"))
	if msg.Priority != "" {
		log.Printf("[LogSender]   Приоритет: %s", msg.Priority)
	}
	if msg.Category != "" {
		log.Printf("[LogSender]   Категория: %s", msg.Category)
	}
	if msg.Notes != "" {
		log.Printf("[LogSender]   Заметки: %s", msg.Notes)
	}

	return nil
}
