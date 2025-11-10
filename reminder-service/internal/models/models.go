package models

import "time"

// RabbitMQMessage — полное сообщение, которое передаётся через очереди RabbitMQ
type RabbitMQMessage struct {
	ID         string    `json:"id"`          // UUID
	UserID     string    `json:"user_id"`     // кто создал задачу
	Text       string    `json:"text"`        // текст задачи / уведомления
	RemindAt   time.Time `json:"remind_at"`   // время для отправки
	Status     string    `json:"status"`      // pending | ready | sent | failed | cancelled
	RetryCount int       `json:"retry_count"` // количество повторных попыток
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Мета-поля
	Complexity int    `json:"complexity"` // 1–5 (сложность)
	Priority   string `json:"priority"`   // low | medium | high
	Category   string `json:"category"`   // работа | личное и т.п.
	Notes      string `json:"notes"`      // комментарий
}

// RedisMessage хранится в redis, выступает как кеш-хранилище напоминаний
type RedisMessage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`    // новый обязательный ключ
	Status    string    `json:"status"`     // pending | ready | sent | failed | cancelled
	Error     string    `json:"error"`      // описание ошибки, если есть
	UpdatedAt time.Time `json:"updated_at"` // когда обновилось
}

// CreateNotificationRequest модель запроса на создание уведомления
type CreateNotificationRequest struct {
	UserID     string    `json:"user_id"`
	Text       string    `json:"text"`
	RemindAt   time.Time `json:"remind_at"`
	Complexity int       `json:"complexity"`
	Priority   string    `json:"priority,omitempty"`
	Category   string    `json:"category,omitempty"`
	Notes      string    `json:"notes,omitempty"`
}
