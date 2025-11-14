package infrastructure

import (
	"context"
	"time"

	"reminder-service/internal/models"
)

// CacheClient интерфейс для взаимодействия с redis
type CacheClientIface interface {
	Save(ctx context.Context, message *models.RedisMessage) error
	SaveWithTTL(ctx context.Context, message *models.RedisMessage, ttl time.Duration) error
	Get(ctx context.Context, id string) (*models.RedisMessage, error)
	Exists(ctx context.Context, id string) (bool, error)
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string) ([]*models.RedisMessage, error)
}

// QueueMQClient интерфейс для взаимодействия с rabbitmq
type QueueRepositoryIface interface {
	PublishDelayed(ctx context.Context, message *models.RabbitMQMessage) error
	PublishReady(ctx context.Context, message *models.RabbitMQMessage) error
	ConsumeDelayed(ctx context.Context, handler func(msg *models.RabbitMQMessage) error) error
	ConsumeReady(ctx context.Context, handler func(msg *models.RabbitMQMessage) error) error
	Retry(ctx context.Context, message *models.RabbitMQMessage, delaySeconds int) error
	QueueLength(ctx context.Context, queueName string) (int, error)
}
