package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"L3.1/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisClient — клиент для взаимодействия с Redis.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient — конструктор клиента Redis.
func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisClient{client: rdb}
}

// Set — сохраняет сообщение в Redis (без TTL).
func (rc *RedisClient) Set(ctx context.Context, key string, message *models.RedisMessage) error {
	if message == nil {
		return nil // или можно вернуть ошибку, если хочешь контролировать
	}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return rc.client.Set(ctx, key, data, 0).Err()
}

// SetWithTTL — сохраняет сообщение в Redis с TTL.
func (rc *RedisClient) SetWithTTL(ctx context.Context, key string, message *models.RedisMessage, ttlSeconds int) error {
	if message == nil {
		return nil
	}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return rc.client.Set(ctx, key, data, time.Duration(ttlSeconds)*time.Second).Err()
}

// Get — получает сообщение из Redis.
func (rc *RedisClient) Get(ctx context.Context, key string) (*models.RedisMessage, error) {
	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var msg models.RedisMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// Exists — проверяет, есть ли запись в Redis по ключу.
func (rc *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Delete — удаляет запись из Redis.
func (rc *RedisClient) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}
