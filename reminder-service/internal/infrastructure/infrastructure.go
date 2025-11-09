package infrastructure

import (
	"context"

	"L3.1/internal/models"
)

// CacheClient интерфейс для взаимодействия с redis
type CacheClient interface {
	Set(ctx context.Context, key string, message *models.RedisMessage) error
	SetWithTTL(ctx context.Context, key string, message *models.RedisMessage, ttlSeconds int) error
	Get(ctx context.Context, key string) (*models.RedisMessage, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
}

// QueueMQClient интерфейс для взаимодействия с rabbitmq
type QueueMQClient interface {
	Publish(ctx context.Context, queueName string, message *models.RabbitMQMessage) error
	Consume(ctx context.Context, queueName string, handler func(msg *models.RabbitMQMessage) error) error
	ConsumeSingleMessage(ctx context.Context, queueName string) (*models.RabbitMQMessage, error)
	RetryMessage(ctx context.Context, queueName string, message *models.RabbitMQMessage, delaySecons int) error
	QueueLength(ctx context.Context, queueName string) (int, error)
}
