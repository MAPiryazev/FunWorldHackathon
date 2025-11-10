package sender

import (
	"context"
	"fmt"
	"log"

	"reminder-service/internal/models"
)

// TODO: сделать отправку уведов в max
// SenderIface — интерфейс для любого способа отправки уведомлений
// Реализация будет зависеть от того, как работает MAX API
type SenderIface interface {
	Send(ctx context.Context, msg *models.RabbitMQMessage) error
}

// LogSender — заглушка для разработки и тестирования
// Позже заменить на реальную реализацию для MAX API
type LogSender struct{}

// NewLogSender создает новый LogSender
func NewLogSender() *LogSender {
	return &LogSender{}
}

// Send логирует уведомление вместо реальной отправки
// TODO: заменить на реальную отправку через MAX API
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

	// TODO: здесь будет отправка через MAX API
	// Пример: maxClient.SendNotification(ctx, msg.UserID, msg.Text, ...)
	return nil
}
