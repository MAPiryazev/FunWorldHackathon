package models

import "time"

// RabbitMQMessage то что будет в redis о сообщении
type RabbitMQMessage struct {
	ID         string
	To         string
	Subject    string
	Body       string
	SendAt     time.Time
	RetryCount int
}

// RedisMessage то что будет в redis о сообщении
type RedisMessage struct {
	ID        string
	Status    string
	UpdatedAt time.Time
}

// CreateNotificationRequest модель запроса на создание уведомления
type CreateNotificationRequest struct {
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	SendAt  time.Time `json:"sendAt"`
}
